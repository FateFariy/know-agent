package reranker

import (
	"context"

	"github.com/swiftbit/know-agent/common"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
)

// Reranker 重排序器
type Reranker interface {
	// Process 重排序
	Process(ctx context.Context, question string, chunks []*vo.DocumentChunk, opts ...common.Option) ([]*vo.DocumentChunk, error)
}
