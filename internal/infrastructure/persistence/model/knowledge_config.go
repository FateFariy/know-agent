package model

import (
	"gorm.io/datatypes"

	"github.com/swiftbit/know-agent/common"
)

type KnowledgeConfig struct {
	common.Model
	BaseName            string         `gorm:"column:base_name;type:varchar(255)"`       // 基础名称
	Description         string         `gorm:"column:description;type:text"`             // 描述
	EmbeddingModel      string         `gorm:"column:embedding_model;type:varchar(100)"` // 嵌入模型
	RetrievalConfigJson datatypes.JSON `gorm:"column:retrieval_config_json;type:json"`   // 检索配置JSON
	GraphRagConfigJson  datatypes.JSON `gorm:"column:graph_rag_config_json;type:json"`   // 图谱RAG配置JSON
	RaptorConfigJson    datatypes.JSON `gorm:"column:raptor_config_json;type:json"`      // Raptor配置JSON
	MetadataFilterJson  datatypes.JSON `gorm:"column:metadata_filter_json;type:json"`    // 元数据过滤JSON
	IsDefault           int            `gorm:"column:is_default;type:tinyint"`           // 是否默认
	SortOrder           int            `gorm:"column:sort_order;type:int"`               // 排序顺序
}
