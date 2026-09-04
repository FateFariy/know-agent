package tool

import (
	"context"
	"errors"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/conversation"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/route"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/enum"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
	"github.com/swiftbit/know-agent/internal/svc"
)

const (
	queryCharMinLength = 5
)

type SearchKnowledgeBaseInput struct {
	QueryType        enum.QueryType         `json:"query_type" jsonschema_description:"用户原始问题类型，用于标识问题的分类以便采用不同的处理策略（如文档问答、跟进提问、结构导航等）" jsonschema:"enum=document_qa,enum=follow_up,enum=structure_navigation,enum=table_query,enum=graph_relation,enum=global_summary,enum=capability_query,required"`
	SubQuestions     []string               `json:"subquestions" jsonschema_description:"待检索的子问题字符串数组：若原问题包含多个独立子问题，可拆分后一次传入；每一项必须是一句完整的自然语言问句（含主谓宾、必要上下文与问号），而非一组关键词碎片" jsonschema:"required"`
	RetrievalIntents []enum.RetrievalIntent `json:"retrieval_intents" jsonschema_description:"检索意图列表，用于指定工具应以何种方式检索知识库（如通用检索、表格检索、图谱检索等）" jsonschema:"enum=general,enum=table,enum=graph_rag,enum=raptor,enum=structure,required"`
	// todo 暂未启用，可忽略
	//Entities         []string                      `json:"entities,omitempty"`
	//TargetEntities   []string                      `json:"targetEntities,omitempty"`
	//ExcludedEntities []string                      `json:"excludedEntities,omitempty"`
	//TableOps         []string                      `json:"tableOps,omitempty"`
	//AnswerShapePlan  []enum.AnswerShapeRequirement `json:"answerShapePlan,omitempty"`
	//Reason           string                        `json:"reason,omitempty"`
}

// KnowledgeBaseSearchTool 知识库检索工具
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

func (t *KnowledgeBaseSearchTool) Info(_ context.Context) *Info {
	return &Info{
		Name: "knowledge_base_search",
		Description: `根据用户输入的问题在知识库中检索证据。
		使用场景：当用户需要查询内部文档、表格、图谱等知识库内容时调用。
		参数要求：
		- query_type：标识原始问题的类型。
		- subquestions：若原问题包含多个独立子问题，必须拆分为多个完整的子问句传入，工具会并行检索并返回合并证据。
		- retrieval_intents：指定检索方式（如通用、表格、图谱等）。
		`,
	}
}

// Invoke 执行一次知识库检索并返回渲染后的证据文本
func (t *KnowledgeBaseSearchTool) Invoke(ctx context.Context, input *SearchKnowledgeBaseInput) (string, error) {
	convCtx := conversation.AgentContextFrom(ctx)
	execPlan := convCtx.ExecutionPlan.Load()
	if execPlan == nil {
		return "", errors.New("知识库检索时执行计划未就绪")
	}

	question := utils.BlankToDefault(execPlan.RewriteQuestion, execPlan.OriginalQuestion)
	input.Normalize(question)

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
		return "未检索到相关结果，建议调整查询条件后重试，如修改检索问题", nil
	}
	result.EvidenceText = t.evidenceRenderer.RenderEvidence(result)

	return result.EvidenceText, nil
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

// Normalize 兜底检索工具入参。
//
// 规则：
//  1. QueryType 空值 → 补 document_qa（不覆写 Agent 已声明的合法值）；
//  2. RetrievalIntents 为空 → 补 general；
//  3. 问题文本含显式章节锚点（章节号/第N章/引号标题）但未声明 structure → 追加 structure 通道。
func (i *SearchKnowledgeBaseInput) Normalize(question string) {
	if i == nil {
		return
	}
	if i.QueryType == "" {
		i.QueryType = enum.QueryTypeDocumentQA
	}
	if len(i.RetrievalIntents) == 0 {
		i.RetrievalIntents = []enum.RetrievalIntent{enum.RetrievalIntentGeneral}
	}
	if !utils.ContainsAny(i.RetrievalIntents, enum.RetrievalIntentStructure) &&
		route.HasExplicitSectionAnchorText(question) {
		i.RetrievalIntents = append(i.RetrievalIntents, enum.RetrievalIntentStructure)
	}
}

//func (i *SearchKnowledgeBaseInput) ToTableIntent() *vo.TableIntent {
//	if i == nil {
//		return &vo.TableIntent{
//			Source: "NONE",
//		}
//	}
//	return &vo.TableIntent{
//		Requested: i.channelRequested(enum.QueryTypeTableQuery, "TABLE"),
//		TableOps:  normalizeStringsLimit(i.TableOps, 8),
//	}
//}
//
//func (i *SearchKnowledgeBaseInput) ToGraphIntent(maxHops int) *vo.GraphIntent {
//	if i == nil {
//		return &vo.GraphIntent{
//			MaxHops: maxHops,
//			Source:  "NONE",
//		}
//	}
//	return &vo.GraphIntent{
//		Requested:      i.channelRequested("GRAPH_RELATION", "GRAPH_RAG"),
//		Entities:       normalizeStringsLimit(i.Entities, 8),
//		TargetEntities: normalizeStringsLimit(i.TargetEntities, 8),
//		MaxHops:        maxHops,
//	}
//}
//
//func (i *SearchKnowledgeBaseInput) ToRaptorIntent(sourceChunkTopK int) *vo.RaptorIntent {
//	if i == nil {
//		return &vo.RaptorIntent{
//			SourceChunkTopK: sourceChunkTopK,
//		}
//	}
//	requested := i.channelRequested("GLOBAL_SUMMARY", "RAPTOR")
//	return &vo.RaptorIntent{
//		Requested:        requested,
//		SummaryRequested: requested,
//		SourceChunkTopK:  sourceChunkTopK,
//	}
//}
//
//// channelRequested 检查通道是否被请求
//func (i *SearchKnowledgeBaseInput) channelRequested(queryType, intent string) bool {
//	if i == nil {
//		return false
//	}
//	return i.QueryType == queryType || utils.ContainsAny(i.RetrievalIntents, intent)
//}
//
//// normalizeStringsLimit 标准化字符串列表并限制数量
//func normalizeStringsLimit(items []string, limit int) []string {
//	return utils.FilterMapUniqueLimit(items, limit, func(item string) (string, string, bool) {
//		item = utils.CompactWhitespace(item)
//		return item, item, item != ""
//	})
//}
