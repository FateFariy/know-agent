package convert

import (
	"github.com/swiftbit/know-agent/api/chat"
	"github.com/swiftbit/know-agent/api/document"
	"github.com/swiftbit/know-agent/api/knowledge"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/aggregate"
	cen "github.com/swiftbit/know-agent/internal/domain/chat/model/entity"
	cvo "github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
	dagg "github.com/swiftbit/know-agent/internal/domain/document/model/aggregate"
	den "github.com/swiftbit/know-agent/internal/domain/document/model/entity"
	dvo "github.com/swiftbit/know-agent/internal/domain/document/model/vo"
	klen "github.com/swiftbit/know-agent/internal/domain/knowledge/model/entity"
	"github.com/swiftbit/know-agent/internal/infrastructure/persistence/model"
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

	// goverter:map . Model
	ToDocumentModel(src *den.Document) *model.Document
	// goverter:map . Model
	ToDocumentProfileModel(src *den.DocumentProfile) *model.DocumentProfile
	// goverter:map . Model
	ToDocumentTaskModel(src *den.DocumentTask) *model.DocumentTask
	// goverter:map . Model
	ToDocumentTaskLogModel(src *den.DocumentTaskLog) *model.DocumentTaskLog
	// goverter:map . Model
	ToDocumentStrategyPlanModel(src *den.DocumentStrategyPlan) *model.DocumentStrategyPlan
	// goverter:map . Model
	ToDocumentStrategyStepModel(src *den.DocumentStrategyStep) *model.DocumentStrategyStep
	ToDocumentStrategyStepModelList(src []*den.DocumentStrategyStep) []*model.DocumentStrategyStep
	// goverter:map . Model
	ToDocumentStructureNodeModel(src *den.StructureNode) *model.DocumentStructureNode
	ToDocumentStructureNodeModelList(src []*den.StructureNode) []*model.DocumentStructureNode
	// goverter:map . Model
	ToDocumentChunkModel(src *den.DocumentChunk) *model.DocumentChunk
	ToDocumentChunkModelList(src []*den.DocumentChunk) []*model.DocumentChunk
	// goverter:map . Model
	ToDocumentParentBlockModel(src *den.DocumentParentChunk) *model.DocumentParentChunk
	ToDocumentParentBlockModelList(src []*den.DocumentParentChunk) []*model.DocumentParentChunk
	// // goverter:map . Model
	// ToDocumentTableCandidateModel(src *den.DocumentTableCandidate) *model.DocumentTableCandidate
	// ToDocumentTableCandidateModelList(src []*den.DocumentTableCandidate) []*model.DocumentTableCandidate
	// goverter:map . Model
	ToParseArtifactModel(src *den.ParseArtifact) *model.DocumentParseArtifact
	ToParseArtifactModelList(src []*den.ParseArtifact) []*model.DocumentParseArtifact

	// goverter:map . Model
	ToDocumentBlockModel(src *den.DocumentBlock) *model.DocumentBlock
	ToDocumentBlockModelList(src []*den.DocumentBlock) []*model.DocumentBlock

	// goverter:map . Model
	ToDocumentTableModel(src *den.DocumentTable) *model.DocumentTable
	ToDocumentTableModelList(src []*den.DocumentTable) []*model.DocumentTable
	// goverter:map . Model
	ToDocumentTableColumnModel(src *den.TableColumn) *model.DocumentTableColumn
	ToDocumentTableColumnModelList(src []*den.TableColumn) []*model.DocumentTableColumn
	// goverter:map . Model
	ToDocumentTableRowModel(src *den.TableRow) *model.DocumentTableRow
	ToDocumentTableRowModelList(src []*den.TableRow) []*model.DocumentTableRow
	// goverter:map . Model
	ToDocumentTableCellModel(src *den.TableCell) *model.DocumentTableCell
	ToDocumentTableCellModelList(src []*den.TableCell) []*model.DocumentTableCell
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
	ToConversationSessionResp(src *aggregate.ConversationArchiveRecord) *chat.ConversationSessionResp
	ToConversationSessionRespList(src []*aggregate.ConversationArchiveRecord) []*chat.ConversationSessionResp
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

	// goverter:map . Model
	ToChatDialogueModel(src *cen.ChatDialogue) *model.ChatDialogue
	// goverter:map . Model
	ToChatExchangeModel(src *cen.ChatExchange) *model.ChatExchange
	// goverter:map . Model
	ToChatExchangeTraceStageModel(src *cen.ChatExchangeTraceStage) *model.ChatExchangeTraceStage
	// goverter:map . Model
	ToChatMemorySummaryModel(src *cen.ChatMemorySummary) *model.ChatMemorySummary
	// goverter:map . Model
	ToChatRetrievalResultModel(src *cvo.ChatRetrievalResult) *model.ChatRetrievalResult
	ToChatRetrievalResultModelList(src []*cvo.ChatRetrievalResult) []*model.ChatRetrievalResult
	// goverter:map . Model
	ToChatChannelExecutionModel(src *cvo.ChatChannelExecution) *model.ChatChannelExecution
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

	// goverter:map . Model
	ToKnowledgeScopeNodeModel(src *klen.KnowledgeScopeNode) *model.KnowledgeScopeNode
	// goverter:map . Model
	ToKnowledgeTopicNodeModel(src *klen.KnowledgeTopicNode) *model.KnowledgeTopicNode
	// goverter:map . Model
	ToKnowledgeTopicDocumentRelationModel(src *klen.KnowledgeTopicDocumentRelation) *model.KnowledgeTopicDocumentRelation
	// goverter:map . Model
	ToKnowledgeRouteTraceModel(src *klen.KnowledgeRouteTrace) *model.KnowledgeRouteTrace
	// goverter:map Model.ID ID
	ToKnowledgeBaseEntity(src *model.KnowledgeBase) *klen.KnowledgeConfig
	ToKnowledgeBaseEntities(src []*model.KnowledgeBase) []*klen.KnowledgeConfig
	// goverter:map . Model
	ToKnowledgeBaseModel(src *klen.KnowledgeConfig) *model.KnowledgeBase
	ToKnowledgeBaseModelList(src []*klen.KnowledgeConfig) []*model.KnowledgeBase
}
