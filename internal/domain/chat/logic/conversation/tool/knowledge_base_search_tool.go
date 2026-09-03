package tool

import (
	"context"
	"errors"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/conversation"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/enum"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
	"github.com/swiftbit/know-agent/internal/svc"
)

const (
	defaultNoEvidenceReply = "未检索到相关结果，建议调整查询条件后重试，如修改检索问题"
	toolDescription        = "根据用户输入的问题在知识库中检索证据。query 为字符串数组：若原问题包含多个独立子问题，请拆分为多个子问题后传入，工具会并行检索并返回合并证据。"
	queryCharMinLength     = 5
)

type SearchKnowledgeBaseInput struct {
	QueryType        enum.QueryType         `json:"query_type" jsonschema_description:"用户原始问题类型" jsonschema:"enum=document_qa,enum=follow_up,enum=structure_navigation,enum=table_query,enum=graph_relation,enum=global_summary,enum=capability_query"`
	SubQuestions     []string               `json:"subquestions" jsonschema_description:"待检索的子问题字符串数组：若原问题包含多个独立子问题，可拆分后一次传入，每一项必须是一句完整的自然语言问句（含主谓宾、必要上下文与问号），而非一组关键词碎片。必填" jsonschema:"required"`
	RetrievalIntents []enum.RetrievalIntent `json:"retrieval_intents" jsonschema_description:"检索意图" jsonschema:"enum=general,enum=table,enum=graph_rag,enum=raptor,enum=structure"`
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

	execPlan.RetrievalPlan.QuestionPlan = newQuestionPlan(execPlan, input)
	result, err := t.retriever.Retrieve(ctx, execPlan.RetrievalPlan)
	if err != nil {
		return "", err
	}
	execPlan.RetrievalResult = result
	convCtx.SetExecutePlan(execPlan)
	t.afterRetrieve(convCtx, result)

	if result.IsEmpty() {
		return defaultNoEvidenceReply, nil
	}
	return t.evidenceRenderer.RenderEvidence(result), nil
}

// afterRetrieve 检索完成后添加已用渠道
func (t *KnowledgeBaseSearchTool) afterRetrieve(convCtx *conversation.Context, result *vo.RetrievalResult) {
	channels := result.UsedChannels()
	convCtx.AddUsedTools(channels...)
	debugTrace := convCtx.DebugTrace.Load()
	debugTrace.AddUsedChannels(channels...)
	debugTrace.AddRetrievalNotes(result.RetrievalNotes()...)
}

// newQuestionPlan 构建检索问题计划
func newQuestionPlan(exec *vo.ConversationExecutionPlan, input *SearchKnowledgeBaseInput) *vo.RetrievalQuestionPlan {
	currentQuestion := utils.CompactWhitespace(exec.OriginalQuestion)
	rewrittenQuestion := utils.CompactWhitespace(exec.RewriteQuestion)
	question := utils.BlankToDefault(rewrittenQuestion, currentQuestion)

	var inheritedAnchors []*vo.RetrievalContextAnchor
	if input.QueryType == enum.QueryTypeFollowUp {
		keyOf := func(anchor *vo.EvidenceAnchor) (string, *vo.RetrievalContextAnchor, bool) {
			if inherited := anchor.ToRetrievalContextAnchor(); inherited != nil {
				return inherited.UniqueKey(), inherited, true
			}
			return "", nil, false
		}
		inheritedAnchors = utils.FilterMapUniqueLimit(exec.RecentEvidenceAnchors, 5, keyOf)
	}
	contextHints := make([]string, 0, len(inheritedAnchors))
	for _, anchor := range inheritedAnchors {
		contextHints = append(contextHints, anchor.AnchorHint())
	}

	of := func(sq string) (string, string, bool) {
		sq = utils.CompactWhitespace(sq)
		return sq, sq, utils.Len(sq) >= queryCharMinLength
	}
	subQuestions := utils.FilterMapUniqueLimit(input.SubQuestions, 5, of)
	if len(subQuestions) == 0 && utils.IsNotBlank(question) {
		subQuestions = append(subQuestions, question)
	}

	executionQueries := make([]*vo.RetrievalExecutionQuery, 0, len(subQuestions))
	for i, sq := range subQuestions {
		executionQueries = append(executionQueries, &vo.RetrievalExecutionQuery{
			Index:        i + 1,
			SubQuestion:  sq,
			ContextHints: append([]string{}, contextHints...),
		})
	}

	return &vo.RetrievalQuestionPlan{
		Question:                 question,
		ExecutionQueries:         executionQueries,
		FollowUp:                 len(inheritedAnchors) > 0,
		HistoryInherited:         len(inheritedAnchors) > 0,
		HistoryInheritanceSource: utils.Ternary(len(inheritedAnchors) > 0, "FINAL_EVIDENCE_ANCHOR", "NONE"),
		InheritedContextAnchors:  inheritedAnchors,
		SubQuestions:             subQuestions,
	}
}
