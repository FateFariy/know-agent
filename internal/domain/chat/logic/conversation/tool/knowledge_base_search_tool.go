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
	toolDescription        = "根据用户输入的问题在知识库中检索证据。若原问题包含多个独立子问题，请拆分为多个子问题后传入，工具会并行检索并返回合并证据。"
	queryCharMinLength     = 5
)

type SearchKnowledgeBaseInput struct {
	QueryType        enum.QueryType         `json:"query_type" jsonschema_description:"用户原始问题类型" jsonschema:"enum=document_qa,enum=follow_up,enum=structure_navigation,enum=table_query,enum=graph_relation,enum=global_summary,enum=capability_query"`
	SubQuestions     []string               `json:"subquestions" jsonschema_description:"待检索的子问题字符串数组：若原问题包含多个独立子问题，可拆分后一次传入，每一项必须是一句完整的自然语言问句（含主谓宾、必要上下文与问号），而非一组关键词碎片。必填" jsonschema:"required"`
	RetrievalIntents []enum.RetrievalIntent `json:"retrieval_intents" jsonschema_description:"用户原始问题检索意图" jsonschema:"enum=general,enum=table,enum=graph_rag,enum=raptor,enum=structure"`
	// todo 暂未启用
	Entities         []string                      `json:"entities,omitempty"`
	TargetEntities   []string                      `json:"targetEntities,omitempty"`
	ExcludedEntities []string                      `json:"excludedEntities,omitempty"`
	TableOps         []string                      `json:"tableOps,omitempty"`
	AnswerShapePlan  []enum.AnswerShapeRequirement `json:"answerShapePlan,omitempty"`
	Reason           string                        `json:"reason,omitempty"`
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
		Name:        "knowledge_base_search",
		Description: toolDescription,
	}
}

// Invoke 执行一次知识库检索并返回渲染后的证据文本
func (t *KnowledgeBaseSearchTool) Invoke(ctx context.Context, input *SearchKnowledgeBaseInput) (string, error) {
	convCtx := conversation.AgentContextFrom(ctx)
	if convCtx == nil {
		return "", errors.New("知识库检索缺少会话上下文")
	}
	execPlan := convCtx.ExecutionPlan.Load()
	if execPlan == nil {
		return "", errors.New("知识库检索时执行计划未就绪")
	}

	question := utils.BlankToDefault(execPlan.RewriteQuestion, execPlan.OriginalQuestion)
	NormalizeSearchInput(question, input)

	retrievalPlan := execPlan.RetrievalPlan
	retrievalPlan.QuestionPlan = newQuestionPlan(execPlan, input)
	retrievalPlan.SuggestedIntents = input.RetrievalIntents
	result, err := t.retriever.Retrieve(ctx, retrievalPlan)
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
	question := utils.BlankToDefault(exec.RewriteQuestion, exec.OriginalQuestion)
	var inheritedAnchors []*vo.RetrievalContextAnchor
	if input.IsFollowUpQuestion(question) {
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

// IsFollowUpQuestion 判断是否为追问
func (i *SearchKnowledgeBaseInput) IsFollowUpQuestion(question string) bool {
	if utils.IsBlank(question) {
		return false
	}
	return i != nil && i.QueryType == enum.QueryTypeFollowUp
}

func (i *SearchKnowledgeBaseInput) ToTableIntent() *vo.TableIntent {
	if i == nil {
		return &vo.TableIntent{
			Source: "NONE",
		}
	}
	return &vo.TableIntent{
		Requested: i.channelRequested(enum.QueryTypeTableQuery, "TABLE"),
		TableOps:  normalizeStringsLimit(i.TableOps, 8),
	}
}

func (i *SearchKnowledgeBaseInput) ToGraphIntent(maxHops int) *vo.GraphIntent {
	if i == nil {
		return &vo.GraphIntent{
			MaxHops: maxHops,
			Source:  "NONE",
		}
	}
	return &vo.GraphIntent{
		Requested:      i.channelRequested("GRAPH_RELATION", "GRAPH_RAG"),
		Entities:       normalizeStringsLimit(i.Entities, 8),
		TargetEntities: normalizeStringsLimit(i.TargetEntities, 8),
		MaxHops:        maxHops,
	}
}

func (i *SearchKnowledgeBaseInput) ToRaptorIntent(sourceChunkTopK int) *vo.RaptorIntent {
	if i == nil {
		return &vo.RaptorIntent{
			SourceChunkTopK: sourceChunkTopK,
		}
	}
	requested := i.channelRequested("GLOBAL_SUMMARY", "RAPTOR")
	return &vo.RaptorIntent{
		Requested:        requested,
		SummaryRequested: requested,
		SourceChunkTopK:  sourceChunkTopK,
	}
}

// channelRequested 检查通道是否被请求
func (i *SearchKnowledgeBaseInput) channelRequested(queryType, intent string) bool {
	if i == nil {
		return false
	}
	return i.QueryType == queryType || utils.ContainsAny(i.RetrievalIntents, intent)
}

// normalizeStringsLimit 标准化字符串列表并限制数量
func normalizeStringsLimit(items []string, limit int) []string {
	return utils.FilterMapUniqueLimit(items, limit, func(item string) (string, string, bool) {
		item = utils.CompactWhitespace(item)
		return item, item, item != ""
	})
}
