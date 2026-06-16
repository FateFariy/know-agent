package convert

import (
	"time"

	"github.com/swiftbit/know-agent/api/chat"
	"github.com/swiftbit/know-agent/api/document"
	cen "github.com/swiftbit/know-agent/internal/domain/chat/model/entity"
	cvo "github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
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

	ToUploadDocumentResp(src *dvo.DocumentUpload) *document.UploadDocumentResp
	// goverter:map ID DocumentId
	ToDocumentListItem(src *den.Document) *document.DocumentListItem
	ToDocumentListItemList(src []*den.Document) []*document.DocumentListItem
	// goverter:map ID DocumentId
	ToQueryStrategyPlanResp(src *den.Document) *document.QueryStrategyPlanResp
	// goverter:map ID PlanId
	ToDocumentStrategyPlan(src *den.DocumentStrategyPlan) *document.DocumentStrategyPlan
	ToBuildIndexResp(src *dvo.DocumentIndexBuild) *document.BuildIndexResp

	// goverter:map . Model
	ToDocumentModel(src *den.Document) *model.Document
	// goverter:map . Model
	ToDocumentTaskModel(src *den.DocumentTask) *model.DocumentTask
	// goverter:map . Model
	ToDocumentTaskLogModel(src *den.DocumentTaskLog) *model.DocumentTaskLog
	// goverter:map . Model
	ToChatDialogueModel(src *cen.ChatDialogue) *model.ChatDialogue
	// goverter:map . Model
	ToChatExchangeModel(src *cen.ChatExchange) *model.ChatExchange
	// goverter:map . Model
	ToChatExchangeTraceStageModel(src *cen.ChatExchangeTraceStage) *model.ChatExchangeTraceStage
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
}

func TimeToString(t time.Time) string {
	return t.Format("2006-01-02 15:04:05")
}

func ToChatQueryMode(name string) int {
	return cvo.ChatQueryModeMap[name]
}
