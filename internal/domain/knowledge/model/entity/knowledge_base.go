package entity

import (
	"encoding/json"
)

type KnowledgeBase struct {
	ID                       int64           // 知识库ID
	BaseName                 string          // 知识库名称
	Description              string          // 描述
	EmbeddingModel           string          // 嵌入模型
	RetrievalConfigJson      json.RawMessage // 检索配置JSON
	GraphRagConfigJson       json.RawMessage // 图谱RAG配置JSON
	RaptorConfigJson         json.RawMessage // RAPTOR配置JSON
	MetadataFilterJson       json.RawMessage // 元数据过滤JSON
	IsDefault                *int            // 是否默认
	SortOrder                int             // 排序
	DocumentCount            int             // 文档数量
	RetrievableDocumentCount int             // 可检索文档数量
}
