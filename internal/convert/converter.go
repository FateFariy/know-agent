package convert

import (
	"time"

	"github.com/swiftbit/know-agent/api/document"
	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
	"github.com/swiftbit/know-agent/internal/infrastructure/model"
)

// goverter:converter
// goverter:output:format function
// goverter:output:file ./converter_gen.go
// goverter:useZeroValueOnPointerInconsistency
// goverter:ignoreMissing
// goverter:extend .*
//
//go:generate goverter gen .
type DocumentConverter interface {
	// goverter:map ID DocumentId
	ToDocumentListItem(src *entity.Document) *document.DocumentListItem
	ToDocumentListItemList(src []*entity.Document) []*document.DocumentListItem

	ToDocumentModel(src *entity.Document) *model.Document
	ToDocumentTaskModel(src *entity.DocumentTask) *model.DocumentTask
	ToDocumentTaskLogModel(src *entity.DocumentTaskLog) *model.DocumentTaskLog
}

func TimeToString(t time.Time) string {
	return t.Format("2006-01-02 15:04:05")
}
