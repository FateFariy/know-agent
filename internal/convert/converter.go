package convert

import (
	"time"

	"github.com/swiftbit/know-agent/api/chat"
	"github.com/swiftbit/know-agent/api/document"
	cvo "github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
	den "github.com/swiftbit/know-agent/internal/domain/document/model/entity"
	dvo "github.com/swiftbit/know-agent/internal/domain/document/model/vo"
	dmo "github.com/swiftbit/know-agent/internal/infrastructure/model"
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

	ToDocumentModel(src *den.Document) *dmo.Document
	ToDocumentTaskModel(src *den.DocumentTask) *dmo.DocumentTask
	ToDocumentTaskLogModel(src *den.DocumentTaskLog) *dmo.DocumentTaskLog
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
