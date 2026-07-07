package convert

import (
	"time"

	"github.com/swiftbit/know-agent/api/chat"
	"github.com/swiftbit/know-agent/api/document"
	"github.com/swiftbit/know-agent/common"
	cen "github.com/swiftbit/know-agent/internal/domain/chat/model/entity"
	cvo "github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
	dagg "github.com/swiftbit/know-agent/internal/domain/document/model/aggregate"
	den "github.com/swiftbit/know-agent/internal/domain/document/model/entity"
	dvo "github.com/swiftbit/know-agent/internal/domain/document/model/vo"
	klen "github.com/swiftbit/know-agent/internal/domain/knowledge/model/entity"
	"github.com/swiftbit/know-agent/internal/infrastructure/model"
)

// goverter:converter
// goverter:output:format function
// goverter:output:file ./converter_gen.go
// goverter:useZeroValueOnPointerInconsistency
// goverter:ignoreMissing
// goverter:extend TimeToString
// goverter:skipCopySameType
//
//go:generate goverter gen .
type DocumentConverter interface {
	FromUploadDocumentReq(src *document.UploadDocumentReq) *den.Document
	FromConfirmStrategyReq(req *document.ConfirmStrategyReq) *dvo.DocumentStrategyConfirmCmd

	ToUploadDocumentResp(src *dvo.DocumentUpload) *document.UploadDocumentResp
	ToDocumentListItem(src *den.Document) *document.DocumentListItem
	ToDocumentListItemList(src []*den.Document) []*document.DocumentListItem
	ToQueryStrategyPlanResp(src *den.Document) *document.QueryStrategyPlanResp
	ToDocumentStrategyPlan(src *den.DocumentStrategyPlan) *document.DocumentStrategyPlan
	ToConfirmStrategyResp(plan *den.DocumentStrategyPlan) *document.ConfirmStrategyResp
	ToBuildIndexResp(src *dvo.DocumentIndexBuild) *document.BuildIndexResp
	ToDocumentChunkItemList(src []*den.DocumentChunk) []*document.DocumentChunkItem
	ToQueryDocumentChunkDetailResp(src *dagg.DocumentChunkDetail) *document.QueryDocumentChunkDetailResp
	ToQueryTaskLogsResp(src *den.DocumentTask) *document.QueryTaskLogsResp

	ToDocumentModel(src *den.Document) *model.Document
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
// goverter:extend TimeToString ToChatQueryMode ToChatQueryModeName JsonArrayToStringSlice JsonArrayToSearchReferences
// goverter:skipCopySameType
type ChatConverter interface {
	FromChatReq(src *chat.ChatReq) *cvo.ChatCommand

	ToRetrievalResultRespList(src []*cvo.ChatRetrievalResult) []*chat.RetrievalResultResp
	ToConversationSessionResp(src *cvo.ConversationArchiveRecord) *chat.ConversationSessionResp
	ToConversationSessionRespList(src []*cvo.ConversationArchiveRecord) []*chat.ConversationSessionResp
	ToConversationResetResp(src *cvo.ConversationReset) *chat.ConversationResetResp
	ToConversationExchangeResp(src *cen.ChatExchange) *chat.ConversationExchangeResp
	ToConversationStageTraceRespList(src []*cen.ChatExchangeTraceStage) []*chat.ConversationTraceStageResp
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

func JsonArrayToStringSlice(src common.JSONArray) []string {
	return common.JSONArrayTo(src, func(item any) string {
		return item.(string)
	})
}

func JsonArrayToSearchReferences(src common.JSONArray) []*chat.SearchReferenceResp {
	return common.JSONArrayTo(src, func(item any) *chat.SearchReferenceResp {
		return item.(*chat.SearchReferenceResp)
	})
}
