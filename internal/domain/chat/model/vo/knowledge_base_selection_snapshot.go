package vo

import (
	"github.com/swiftbit/know-agent/internal/domain/chat/model/enum"
)

type KnowledgeBaseSelectionSnapshot struct {
	SelectionMode              enum.KnowledgeBaseSelectionMode `json:"selectionMode"`
	SelectedKnowledgeBaseIds   []int64                         `json:"selectedKnowledgeBaseIds"`
	SelectedKnowledgeBaseNames []string                        `json:"selectedKnowledgeBaseNames"`
	SelectedKnowledgeBases     []*KnowledgeBase                `json:"selectedKnowledgeBases"`
	AllowedDocuments           []*DocumentMetadata             `json:"allowedDocuments"`
	AllowedDocumentIds         []int64                         `json:"allowedDocumentIds"`
	AllowedTaskIds             []int64                         `json:"allowedTaskIds"`
	//RagRuntimeOptions        *RagRuntimeOptions           `json:"ragRuntimeOptions,omitempty"`
}
