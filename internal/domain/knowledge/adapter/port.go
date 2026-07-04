package adapter

import (
	"context"

	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
)

type KnowledgePort struct {
	VectorDB
}

type VectorDB interface {
	// SearchByVector 基于 Milvus 向量数据库进行相似度检索
	// query 为用于生成 embedding 的查询文本；documentIDs/taskIDs 限制检索范围；
	// topK 为返回条数上限；filters 提供细粒度过滤（section_path/structure_node_id/canonical_path/item_index 等）。
	SearchByVector(ctx context.Context, query string, documentIds, taskIds []int64, topK int, filters *vo.DocumentRetrieveFilters) ([]*vo.DocumentChunk, error)

	// SearchByKeyword 基于关键词的 BM25 风格检索（在 SQL/外部索引上按 keyword 叠加分数）
	SearchByKeyword(ctx context.Context, query string, documentIds, taskIds []int64, topK int, filters *vo.DocumentRetrieveFilters) ([]*vo.DocumentChunk, error)
}

// Embedder 文本嵌入模型
type Embedder interface {
	// EmbedStrings 文本向量化
	EmbedStrings(ctx context.Context, texts ...string) ([][]float64, error)
}

// RouteLexicalIndex 路由侧的词面索引能力
type RouteLexicalIndex interface {
	// Search 在指定实体类型下进行词面检索，返回命中 (entityCode/documentId, score) 列表
	Search(ctx context.Context, routingText string, entityType string, size int) ([]*vo.RouteLexicalHit, error)
}
