package adapter

import (
	"context"

	"github.com/swiftbit/know-agent/internal/domain/knowledge/model/data"
	"github.com/swiftbit/know-agent/internal/domain/knowledge/model/vo"
)

type KnowledgeRepository interface {
	// ListDocuments 列出所有文档
	ListDocuments(ctx context.Context) ([]*vo.KnowledgeDocument, error)

	// ListDocumentsByIDs 根据文档ID列表列出文档
	ListDocumentsByIDs(ctx context.Context, documentIDs []int64) ([]*vo.KnowledgeDocument, error)

	SearchByVector(ctx context.Context, query string, documentIDs, taskIDs []int64, topK int, filters *vo.DocumentRetrieveFilters) ([]*data.EmbeddingChunk, error)

	SearchByKeyword(ctx context.Context, query string, documentIDs, taskIDs []int64, topK int, filters *vo.DocumentRetrieveFilters) ([]*data.EmbeddingChunk, error)

	GetParentBlocks(ctx context.Context, parentBlockIDs []int64) ([]*data.SuperAgentDocumentParentBlock, error)
}
