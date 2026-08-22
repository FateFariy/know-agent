package route

import (
	"github.com/swiftbit/know-agent/common"
	"github.com/swiftbit/know-agent/internal/domain/knowledge/adapter"
)

type options struct {
	embedder adapter.Embedder
}

type Option = common.Option

// WithEmbedding 注册嵌入模型（可选）
func WithEmbedding(emb adapter.Embedder) Option {
	return common.WrapImplSpecificOptFn(func(o *options) {
		o.embedder = emb
	})
}
