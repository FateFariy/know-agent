package chunk

import (
	"context"

	"github.com/swiftbit/know-agent/common"
)

// Chunker 文本分块器
type Chunker interface {
	// Name 名称，唯一标识
	Name() string

	// Chunk 将一段输入文本切分为多个文本块
	Chunk(ctx context.Context, input string, opts ...common.Option) ([]string, error)
}
