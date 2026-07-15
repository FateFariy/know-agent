package convert

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/duke-git/lancet/v2/strutil"

	"github.com/swiftbit/know-agent/api/chat"
	"github.com/swiftbit/know-agent/api/document"
	"github.com/swiftbit/know-agent/api/knowledge"
	"github.com/swiftbit/know-agent/common"
	cen "github.com/swiftbit/know-agent/internal/domain/chat/model/entity"
	cvo "github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
	dagg "github.com/swiftbit/know-agent/internal/domain/document/model/aggregate"
	den "github.com/swiftbit/know-agent/internal/domain/document/model/entity"
	dvo "github.com/swiftbit/know-agent/internal/domain/document/model/vo"
	klen "github.com/swiftbit/know-agent/internal/domain/knowledge/model/entity"
	klvo "github.com/swiftbit/know-agent/internal/domain/knowledge/model/vo"
	"github.com/swiftbit/know-agent/internal/infrastructure/model"
)

// goverter:converter
// goverter:output:format function
// goverter:output:file ./converter_gen.go
// goverter:useZeroValueOnPointerInconsistency
// goverter:ignoreMissing
// goverter:extend TimeToString StringToStringSlice Int64ToString StringToInt64
// goverter:skipCopySameType
//
//go:generate goverter gen .
type DocumentConverter interface {
	FromUploadDocumentReq(src *document.UploadDocumentReq) *den.Document
	FromConfirmStrategyReq(src *document.ConfirmStrategyReq) *dvo.DocumentStrategyConfirmCmd

	ToUploadDocumentResp(src *dvo.DocumentUpload) *document.UploadDocumentResp
	ToDocumentDetailResp(src *den.Document) *document.DocumentDetailResp
	ToDocumentDetailRespList(src []*den.Document) []*document.DocumentDetailResp
	ToKnowledgeDocumentOptionRespList(src []*dvo.KnowledgeDocument) []*document.KnowledgeDocumentOptionResp
	ToQueryStrategyPlanResp(src *den.Document) *document.QueryStrategyPlanResp
	ToDocumentStrategyPlan(src *den.DocumentStrategyPlan) *document.DocumentStrategyPlan
	ToConfirmStrategyResp(plan *den.DocumentStrategyPlan) *document.ConfirmStrategyResp
	ToBuildIndexResp(src *dvo.DocumentIndexBuild) *document.BuildIndexResp
	ToDocumentChunkItemList(src []*den.DocumentChunk) []*document.DocumentChunkItem
	ToQueryDocumentChunkDetailResp(src *dagg.DocumentChunkDetail) *document.QueryDocumentChunkDetailResp
	ToQueryTaskLogsResp(src *den.DocumentTask) *document.QueryTaskLogsResp
	ToDocumentProfileResp(src *den.DocumentProfile) *document.DocumentProfileResp
	ToDocumentProfileRespList(src []*den.DocumentProfile) []*document.DocumentProfileResp

	ToDocumentModel(src *den.Document) *model.Document
	ToDocumentProfileModel(src *den.DocumentProfile) *model.DocumentProfile
	ToDocumentTaskModel(src *den.DocumentTask) *model.DocumentTask
	ToDocumentTaskLogModel(src *den.DocumentTaskLog) *model.DocumentTaskLog
	ToDocumentStrategyPlanModel(src *den.DocumentStrategyPlan) *model.DocumentStrategyPlan
	ToDocumentStrategyStepModel(src *den.DocumentStrategyStep) *model.DocumentStrategyStep
	ToDocumentStrategyStepModelList(src []*den.DocumentStrategyStep) []*model.DocumentStrategyStep
	ToDocumentStructureNodeModelList(src []*den.DocumentStructureNode) []*model.DocumentStructureNode
	ToDocumentChunkModel(src *den.DocumentChunk) *model.DocumentChunk
	ToDocumentChunkModelList(src []*den.DocumentChunk) []*model.DocumentChunk
	ToDocumentParentBlockModel(src *den.DocumentParentBlock) *model.DocumentParentBlock
	ToDocumentParentBlockModelList(src []*den.DocumentParentBlock) []*model.DocumentParentBlock
}

// goverter:converter
// goverter:output:format function
// goverter:output:file ./converter_gen.go
// goverter:useZeroValueOnPointerInconsistency
// goverter:ignoreMissing
// goverter:extend TimeToString ToChatQueryMode ToChatQueryModeName JsonArrayToStringSlice JsonArrayToSearchReferences Int64ToString StringToInt64
// goverter:skipCopySameType
type ChatConverter interface {
	FromChatReq(src *chat.ChatReq) *cvo.ChatCommand

	ToRetrievalResultRespList(src []*cvo.ChatRetrievalResult) []*chat.RetrievalResultResp
	ToConversationSessionResp(src *cvo.ConversationArchiveRecord) *chat.ConversationSessionResp
	ToConversationSessionRespList(src []*cvo.ConversationArchiveRecord) []*chat.ConversationSessionResp
	ToConversationResetResp(src *cvo.ConversationReset) *chat.ConversationResetResp
	// goverter:map DebugTrace | ToChatDebugTrace
	ToConversationExchange(src *cen.ChatExchange) *chat.ConversationExchange
	ToConversationStageTraces(src []*cen.ChatExchangeTraceStage) []*chat.ConversationTraceStage
	// goverter:map UpdateTime | TimeToStringMs
	ToConversationMemorySummaryResp(src *cen.ChatMemorySummary) *chat.ConversationMemorySummaryResp
	// goverter:map StartTime | TimeToStringMs
	// goverter:map EndTime | TimeToStringMs
	ToChannelExecutionResp(src *cvo.ChatChannelExecution) *chat.ChannelExecutionResp
	ToChannelExecutionRespList(src []*cvo.ChatChannelExecution) []*chat.ChannelExecutionResp

	ToChatDialogueModel(src *cen.ChatDialogue) *model.ChatDialogue
	ToChatExchangeModel(src *cen.ChatExchange) *model.ChatExchange
	ToChatExchangeTraceStageModel(src *cen.ChatExchangeTraceStage) *model.ChatExchangeTraceStage
	ToChatMemorySummaryModel(src *cen.ChatMemorySummary) *model.ChatMemorySummary
	ToChatRetrievalResultModelList(src []*cvo.ChatRetrievalResult) []*model.ChatRetrievalResult
	ToChatChannelExecutionModelList(src []*cvo.ChatChannelExecution) []*model.ChatChannelExecution
}

// goverter:converter
// goverter:output:format function
// goverter:output:file ./converter_gen.go
// goverter:useZeroValueOnPointerInconsistency
// goverter:ignoreMissing
// goverter:extend .*
// goverter:skipCopySameType
type KnowledgeConverter interface {
	FromKnowledgeScopeSaveReq(src *knowledge.KnowledgeScopeSaveReq) *klen.KnowledgeScopeNode
	FromKnowledgeTopicSaveReq(src *knowledge.KnowledgeTopicSaveReq) *klen.KnowledgeTopicNode
	FromKnowledgeTopicDocumentRelationSaveReq(src *knowledge.TopicDocumentRelationSaveReq) *klen.KnowledgeTopicDocumentRelation

	ToKnowledgeScopeResp(src *klen.KnowledgeScopeNode) *knowledge.KnowledgeScopeResp
	ToKnowledgeTopicResp(src *klen.KnowledgeTopicNode) *knowledge.KnowledgeTopicResp
	ToKnowledgeScopeRespList(src []*klen.KnowledgeScopeNode) []*knowledge.KnowledgeScopeResp
	ToKnowledgeTopicRespList(src []*klen.KnowledgeTopicNode) []*knowledge.KnowledgeTopicResp

	ToTopicDocumentRelationResp(src *klen.KnowledgeTopicDocumentRelation) *knowledge.TopicDocumentRelationResp
	ToTopicDocumentRelationRespList(src []*klen.KnowledgeTopicDocumentRelation) []*knowledge.TopicDocumentRelationResp
	// goverter:map RouteStatus | ToRouteStatus
	ToKnowledgeRouteTraceItem(src *klen.KnowledgeRouteTrace) *knowledge.KnowledgeRouteTraceItem
	ToKnowledgeRouteTraceItemList(src []*klen.KnowledgeRouteTrace) []*knowledge.KnowledgeRouteTraceItem

	ToKnowledgeScopeNodeModel(src *klen.KnowledgeScopeNode) *model.KnowledgeScopeNode
	ToKnowledgeTopicNodeModel(src *klen.KnowledgeTopicNode) *model.KnowledgeTopicNode
	ToKnowledgeTopicDocumentRelationModel(src *klen.KnowledgeTopicDocumentRelation) *model.KnowledgeTopicDocumentRelation
	ToKnowledgeRouteTraceModel(src *klen.KnowledgeRouteTrace) *model.KnowledgeRouteTrace
}

func TimeToString(t time.Time) string {
	return t.Format(time.DateTime)
}

func TimeToStringMs(t time.Time) string {
	return t.Format("2006-01-02 15:04:05.000")
}

func ToChatQueryMode(name string) cvo.ChatQueryMode {
	return cvo.ToChatQueryMode(name)
}

func ToChatQueryModeName(code int) string {
	return cvo.ChatQueryModeName(code)
}

func ToChatDebugTrace(debugTraceJson string) *chat.ChatDebugTrace {
	var debugTrace chat.ChatDebugTrace
	if err := json.Unmarshal([]byte(debugTraceJson), &debugTrace); err != nil {
		return nil
	}
	return &debugTrace
}

func JsonArrayToStringSlice(src common.JSONArray) []string {
	return common.JSONArrayTo(src, func(item any) string {
		return item.(string)
	})
}

func StringToStringSlice(src string) []string {
	return strings.Split(src, ",")
}

func JsonArrayToSearchReferences(src common.JSONArray) []*chat.SearchReference {
	return common.JSONArrayTo(src, func(item any) *chat.SearchReference {
		return item.(*chat.SearchReference)
	})
}

func ToRouteStatus(code int) string {
	return klvo.RouteStatusName(code)
}

func NormalizeString(s string) string {
	return strutil.Trim(s)
}

func Int64ToString(i int64) string {
	return strconv.FormatInt(i, 10)
}

func StringToInt64(s string) int64 {
	i, _ := strconv.ParseInt(s, 10, 64)
	return i
}
