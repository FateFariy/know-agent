package adapter

import (
	"context"

	"github.com/swiftbit/know-agent/internal/domain/rag/model/vo"
)

type VectorDB interface {
	// SearchByVector 基于 Milvus 向量数据库进行相似度检索
	SearchByVector(ctx context.Context, query *vo.DocumentRetrieve) ([]*vo.DocumentChunk, error)
}

type KeywordDB interface {
	// SearchByKeyword 基于关键词索引进行检索
	SearchByKeyword(ctx context.Context, query *vo.DocumentRetrieve) ([]*vo.DocumentChunk, error)
}
