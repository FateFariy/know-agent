package adapter

import (
	"context"

	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/knowledge/model/data"
	"github.com/swiftbit/know-agent/internal/domain/knowledge/model/vo"
)

type KnowledgeRepository interface {
	// Do 运行一个事务
	Do(ctx context.Context, fn func(ctx context.Context) error) error

	// SelectRetrievableDocuments 查询可检索的文档
	SelectRetrievableDocuments(ctx context.Context, documentIds ...int64) ([]*vo.KnowledgeDocument, error)

	// SelectParentBlocks 根据ID列表查询父级块
	SelectParentBlocks(ctx context.Context, parentBlockIDs []int64) ([]*entity.DocumentParentBlock, error)

	// SearchByVector 基于 Milvus 向量数据库进行相似度检索
	// query 为用于生成 embedding 的查询文本；documentIDs/taskIDs 限制检索范围；
	// topK 为返回条数上限；filters 提供细粒度过滤（section_path/structure_node_id/canonical_path/item_index 等）。
	SearchByVector(ctx context.Context, query string, documentIDs, taskIDs []int64, topK int, filters *vo.DocumentRetrieveFilters) ([]*data.EmbeddingChunk, error)

	// SearchByKeyword 基于关键词的 BM25 风格检索（在 SQL/外部索引上按 keyword 叠加分数）
	SearchByKeyword(ctx context.Context, query string, documentIDs, taskIDs []int64, topK int, filters *vo.DocumentRetrieveFilters) ([]*data.EmbeddingChunk, error)
}
