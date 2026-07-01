package adapter

import (
	"context"

	vo2 "github.com/swiftbit/know-agent/internal/domain/rag/model/vo"
)

type KnowledgePort struct {
	VectorDB
}

type VectorDB interface {
	// SearchByVector 基于 Milvus 向量数据库进行相似度检索
	// query 为用于生成 embedding 的查询文本；documentIDs/taskIDs 限制检索范围；
	// topK 为返回条数上限；filters 提供细粒度过滤（section_path/structure_node_id/canonical_path/item_index 等）。
	SearchByVector(ctx context.Context, query string, documentIds, taskIds []int64, topK int, filters *vo2.DocumentRetrieveFilters) ([]*vo2.DocumentChunk, error)

	// SearchByKeyword 基于关键词的 BM25 风格检索（在 SQL/外部索引上按 keyword 叠加分数）
	SearchByKeyword(ctx context.Context, query string, documentIds, taskIds []int64, topK int, filters *vo2.DocumentRetrieveFilters) ([]*vo2.DocumentChunk, error)
}
