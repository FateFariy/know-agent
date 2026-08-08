package chunk

import (
	"context"

	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/document/model/vo"
)

type Input struct {
	Blocks entity.DocumentBlocks
	Nodes  []*entity.StructureNode
}

// Chunker 文本分块器
type Chunker interface {
	// Name 名称，唯一标识
	Name() string

	// Chunk 将一段输入文本切分为多个文本块
	Chunk(ctx context.Context, input *Input, opts ...Option) ([]*vo.ChunkCandidate, error)
}

// Option 配置策略的函数选项
type Option struct {
	implSpecificOptFn any
}

// WrapSpecificOptFn 将策略专属的 option 函数封装为通用 Option
func WrapSpecificOptFn[T any](optFn func(*T)) Option {
	return Option{
		implSpecificOptFn: optFn,
	}
}

// GetSpecificOptions 从 Option 列表中获取策略实现专有选项
func GetSpecificOptions[T any](base *T, opts ...Option) *T {
	if base == nil {
		base = new(T)
	}

	for i := range opts {
		opt := opts[i]
		if opt.implSpecificOptFn != nil {
			s, ok := opt.implSpecificOptFn.(func(*T))
			if ok {
				s(base)
			}
		}
	}

	return base
}
