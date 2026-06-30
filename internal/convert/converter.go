package convert

import (
	"time"

	"github.com/swiftbit/know-agent/api/chat"
	"github.com/swiftbit/know-agent/api/document"
	cen "github.com/swiftbit/know-agent/internal/domain/chat/model/entity"
	cvo "github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
	dagg "github.com/swiftbit/know-agent/internal/domain/document/model/aggregate"
	den "github.com/swiftbit/know-agent/internal/domain/document/model/entity"
	dvo "github.com/swiftbit/know-agent/internal/domain/document/model/vo"
	"github.com/swiftbit/know-agent/internal/infrastructure/model"
)

// goverter:converter
// goverter:output:format function
// goverter:output:file ./converter_gen.go
// goverter:useZeroValueOnPointerInconsistency
// goverter:ignoreMissing
// goverter:extend .*
// goverter:skipCopySameType
//
//go:generate goverter gen .
type DocumentConverter interface {
	FromUploadDocumentReq(src *document.UploadDocumentReq) *den.Document
	FromConfirmStrategyReq(req *document.ConfirmStrategyReq) *dvo.DocumentStrategyConfirmCmd

	ToUploadDocumentResp(src *dvo.DocumentUpload) *document.UploadDocumentResp
	// goverter:map ID DocumentId
	ToDocumentListItem(src *den.Document) *document.DocumentListItem
	ToDocumentListItemList(src []*den.Document) []*document.DocumentListItem
	// goverter:map ID DocumentId
	ToQueryStrategyPlanResp(src *den.Document) *document.QueryStrategyPlanResp
	// goverter:map ID PlanId
	ToDocumentStrategyPlan(src *den.DocumentStrategyPlan) *document.DocumentStrategyPlan
	// goverter:map ID PlanId
	ToConfirmStrategyResp(plan *den.DocumentStrategyPlan) *document.ConfirmStrategyResp
	ToBuildIndexResp(src *dvo.DocumentIndexBuild) *document.BuildIndexResp
	ToDocumentChunkItemList(src []*den.DocumentChunk) []*document.DocumentChunkItem
	ToQueryDocumentChunkDetailResp(src *dagg.DocumentChunkDetail) *document.QueryDocumentChunkDetailResp
	// goverter:map ID TaskId
	ToQueryTaskLogsResp(src *den.DocumentTask) *document.QueryTaskLogsResp

	ToDocumentModel(src *den.Document) *model.Document
	ToDocumentTaskModel(src *den.DocumentTask) *model.DocumentTask
	ToDocumentTaskLogModel(src *den.DocumentTaskLog) *model.DocumentTaskLog
	ToDocumentStrategyPlanModel(src *den.DocumentStrategyPlan) *model.DocumentStrategyPlan
	ToDocumentStrategyStepModel(src *den.DocumentStrategyStep) *model.DocumentStrategyStep
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
// goverter:extend .*
// goverter:skipCopySameType
type ChatConverter interface {
	FromChatReq(src *chat.ChatReq) *cvo.ChatCommand

	ToChatDialogueModel(src *cen.ChatDialogue) *model.ChatDialogue
	ToChatExchangeModel(src *cen.ChatExchange) *model.ChatExchange
	ToChatExchangeTraceStageModel(src *cen.ChatExchangeTraceStage) *model.ChatExchangeTraceStage
	ToChatMemorySummaryModel(src *cen.ChatMemorySummary) *model.ChatMemorySummary
}

func TimeToString(t time.Time) string {
	return t.Format("2006-01-02 15:04:05")
}

func ToChatQueryMode(name string) cvo.ChatQueryMode {
	return cvo.ToChatQueryMode(name)
}
