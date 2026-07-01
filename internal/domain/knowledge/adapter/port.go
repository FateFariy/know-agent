package adapter

import (
	"context"

	"github.com/swiftbit/know-agent/internal/domain/knowledge/model/data"
)

type VectorDB interface {
	VectorRetrieval(ctx context.Context, query string) ([]*data.EmbeddingChunk, error)
}
