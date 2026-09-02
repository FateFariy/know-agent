package conversation

import (
	"context"
	"errors"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
)

const (
	defaultNoEvidenceReply = "当前没有足够证据支持明确回答。"
)

// KnowledgeBaseSearchTool 知识库检索工具
// 基于当前执行计划执行检索、把结果回填执行计划、发布检索笔记与已用渠道，最终将证据渲染为带编号引用的文本返回给 agent
type KnowledgeBaseSearchTool struct {
	retriever        Retriever
	evidenceRenderer *EvidenceRenderer
}

// NewKnowledgeBaseSearchTool 创建知识库检索工具
func NewKnowledgeBaseSearchTool(retriever Retriever, evidenceRenderer *EvidenceRenderer) *KnowledgeBaseSearchTool {
	return &KnowledgeBaseSearchTool{
		retriever:        retriever,
		evidenceRenderer: evidenceRenderer,
	}
}

// Search 执行一次知识库检索并返回渲染后的证据文本。
//
// query 非空且与当前计划检索问题不同时，临时以 query 单子问题执行检索；
// topK 大于 0 时覆盖计划最终预算。检索结果会回填执行计划供后续引用发布与缓存写入。
func (t *KnowledgeBaseSearchTool) Search(ctx context.Context, query string, topK int) (string, error) {
	convCtx := AgentContextFrom(ctx)
	if convCtx == nil {
		return "", errors.New("知识库检索缺少会话上下文")
	}
	execPlan := convCtx.ExecutionPlan.Load()
	if execPlan == nil {
		return "", errors.New("知识库检索时执行计划未就绪")
	}

	plan := t.resolvePlan(execPlan, query, topK)
	if plan == nil {
		return "", errors.New("知识库检索计划未就绪，请先完成知识范围/文档结构路由")
	}

	result, err := t.retriever.Retrieve(ctx, plan)
	if err != nil {
		return "", err
	}
	execPlan.RetrievalPlan = plan
	execPlan.RetrievalResult = result
	convCtx.SetExecutePlan(execPlan)

	if err = t.afterRetrieve(ctx, convCtx, result); err != nil {
		return "", err
	}

	if result.IsEmpty() {
		return utils.BlankToDefault(execPlan.NoEvidenceReply, defaultNoEvidenceReply), nil
	}
	return t.evidenceRenderer.RenderEvidence(result), nil
}

// resolvePlan 解析待执行的检索计划：优先复用路由阶段已生成的计划，query/topK
// 仅在明确给出时覆盖计划（浅拷贝，避免污染执行计划对象）。
func (t *KnowledgeBaseSearchTool) resolvePlan(execPlan *vo.ConversationExecutionPlan, query string, topK int) *vo.RetrievalPlan {
	plan := execPlan.RetrievalPlan
	if plan == nil {
		return nil
	}
	overrideQuery := ""
	if q := utils.CompactWhitespace(query); q != "" &&
		(plan.QuestionPlan == nil || plan.QuestionPlan.RetrievalQuestion != q) {
		overrideQuery = q
	}
	if overrideQuery == "" && (topK <= 0 || topK == plan.FinalTopK) {
		return plan
	}

	cloned := *plan
	if overrideQuery != "" {
		cloned.QuestionPlan = &vo.RetrievalQuestionPlan{
			CurrentQuestion:   overrideQuery,
			RewrittenQuestion: overrideQuery,
			RetrievalQuestion: overrideQuery,
			ExecutionQueries:  []*vo.RetrievalExecutionQuery{{Index: 1, SubQuestion: overrideQuery}},
			SubQuestions:      []string{overrideQuery},
		}
	}
	if topK > 0 {
		cloned.FinalTopK = topK
	}
	return &cloned
}

// afterRetrieve 检索完成后下发思考事件与已用渠道（引用发布由答案输出中间件统一负责）。
func (t *KnowledgeBaseSearchTool) afterRetrieve(ctx context.Context, convCtx *Context, result *vo.RetrievalResult) error {
	for _, note := range result.RetrievalNotes() {
		if err := convCtx.PublishThinking(note); err != nil {
			return err
		}
	}
	channels := result.UsedChannels()
	convCtx.AddUsedTools(channels...)
	if debugTrace := convCtx.DebugTrace.Load(); debugTrace != nil {
		debugTrace.SetUsedChannels(channels...)
		debugTrace.SetRetrievalNotes(result.RetrievalNotes()...)
	}
	return nil
}
