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

	// SelectAllDocuments 查询所有文档
	SelectAllDocuments(ctx context.Context) ([]*vo.KnowledgeDocument, error)

	// SelectDocumentsByIDs  查询指定文档
	SelectDocumentsByIDs(ctx context.Context, documentIDs []int64) ([]*vo.KnowledgeDocument, error)

	// SearchByVector 向量检索
	SearchByVector(ctx context.Context, query string, documentIDs, taskIDs []int64, topK int, filters *vo.DocumentRetrieveFilters) ([]*data.EmbeddingChunk, error)

	// SearchByKeyword 关键词检索
	SearchByKeyword(ctx context.Context, query string, documentIDs, taskIDs []int64, topK int, filters *vo.DocumentRetrieveFilters) ([]*data.EmbeddingChunk, error)

	// SelectParentBlocks 查询父级块
	SelectParentBlocks(ctx context.Context, parentBlockIDs []int64) ([]*entity.DocumentParentBlock, error)
}
