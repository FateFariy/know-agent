package model

import (
	"gorm.io/datatypes"

	"github.com/swiftbit/know-agent/common"
)

type KnowledgeConfig struct {
	common.Model
	BaseName            string         `gorm:"column:base_name"`
	Description         string         `gorm:"column:description"`
	EmbeddingModel      string         `gorm:"column:embedding_model"`
	RetrievalConfigJson datatypes.JSON `gorm:"column:retrieval_config_json,type:json"`
	GraphRagConfigJson  datatypes.JSON `gorm:"column:graph_rag_config_json,type:json"`
	RaptorConfigJson    datatypes.JSON `gorm:"column:raptor_config_json,type:json"`
	MetadataFilterJson  datatypes.JSON `gorm:"column:metadata_filter_json,type:json"`
	IsDefault           int            `gorm:"column:is_default"`
	SortOrder           int            `gorm:"column:sort_order"`
}
