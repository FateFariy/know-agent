package channel

import (
	"context"

	"github.com/swiftbit/know-agent/internal/domain/chat/logic/rag"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
)

type KeywordRetriever interface {
	// SearchByKeyword  按关键词检索
	SearchByKeyword(ctx context.Context, query *rag.DocumentRetrieve) ([]*vo.DocumentChunk, error)
}

type VectorRetriever interface {
	// SearchByVector 按向量检索
	SearchByVector(ctx context.Context, query *rag.DocumentRetrieve) ([]*vo.DocumentChunk, error)
}
