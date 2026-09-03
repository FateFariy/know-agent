package tool

import (
	"context"
	"errors"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/conversation"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
	"github.com/swiftbit/know-agent/internal/svc"
)

const (
	defaultNoEvidenceReply = "未检索到相关结果，建议调整查询条件后重试，如修改检索问题"
	toolDescription        = "根据用户输入的问题在知识库中检索证据。query 为字符串数组：若原问题包含多个独立子问题，请拆分为多个子问题后传入，工具会并行检索并返回合并证据。"
)

type SearchKnowledgeBaseInput struct {
	Query []string `json:"query" jsonschema_description:"待检索的子问题字符串数组：若原问题包含多个独立子问题，可拆分后一次传入，每一项必须是一句完整的自然语言问句（含主谓宾、必要上下文与问号），而非一组关键词碎片。必填" jsonschema:"required"`
	TopK  int      `json:"top_k,omitempty" jsonschema_description:"每个子问题返回的检索结果数量，默认 5"`
}

// KnowledgeBaseSearchTool 知识库检索工具
// 基于当前执行计划执行检索、把结果回填执行计划、发布检索笔记与已用渠道，最终将证据渲染为带编号引用的文本返回给 agent
type KnowledgeBaseSearchTool struct {
	retriever        Retriever
	evidenceRenderer *EvidenceRenderer
}

// NewKnowledgeBaseSearchTool 创建知识库检索工具
func NewKnowledgeBaseSearchTool(svcCtx *svc.ServiceContext, retriever Retriever) *KnowledgeBaseSearchTool {
	return &KnowledgeBaseSearchTool{
		retriever:        retriever,
		evidenceRenderer: NewEvidenceRenderer(svcCtx),
	}
}

func (t *KnowledgeBaseSearchTool) Info(ctx context.Context) *Info {
	return &Info{
		Name:        "knowledge_base_search_tool",
		Description: toolDescription,
	}
}

// Invoke 执行一次知识库检索并返回渲染后的证据文本。
//
// query 非空且与当前计划检索问题不同时，临时以 query 单子问题执行检索；
// topK 大于 0 时覆盖计划最终预算。检索结果会回填执行计划供后续引用发布与缓存写入。
func (t *KnowledgeBaseSearchTool) Invoke(ctx context.Context, input *SearchKnowledgeBaseInput) (string, error) {
	convCtx := conversation.AgentContextFrom(ctx)
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
		return defaultNoEvidenceReply, nil
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
	if q := utils.CompactWhitespace(query); q != "" {
		overrideQuery = q
	}
	if overrideQuery == "" && (topK <= 0 || topK == plan.FinalTopK) {
		return plan
	}

	cloned := *plan
	if overrideQuery != "" {
		cloned.QuestionPlan = &vo.RetrievalQuestionPlan{
			Question:         overrideQuery,
			ExecutionQueries: []*vo.RetrievalExecutionQuery{{Index: 1, SubQuestion: overrideQuery}},
			SubQuestions:     []string{overrideQuery},
		}
	}
	if topK > 0 {
		cloned.FinalTopK = topK
	}
	return &cloned
}

// afterRetrieve 检索完成后下发思考事件与已用渠道（引用发布由答案输出中间件统一负责）。
func (t *KnowledgeBaseSearchTool) afterRetrieve(ctx context.Context, convCtx *conversation.Context, result *vo.RetrievalResult) error {
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
