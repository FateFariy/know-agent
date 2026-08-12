package entity

import (
	"encoding/json"
)

type KnowledgeBase struct {
	ID                  int64
	BaseName            string
	Description         string
	EmbeddingModel      string
	RetrievalConfigJson json.RawMessage
	GraphRagConfigJson  json.RawMessage
	RaptorConfigJson    json.RawMessage
	MetadataFilterJson  json.RawMessage
	IsDefault           int
	SortOrder           int
}
