package rank

import (
	"github.com/swiftbit/know-agent/common"
	"github.com/swiftbit/know-agent/internal/domain/knowledge/adapter"
	"github.com/swiftbit/know-agent/internal/domain/knowledge/logic/route/score"
)

type Option = common.Option

type options struct {
	embedder     adapter.Embedder
	lexicalIndex RouteLexicalIndex
	scorer       score.Scorer
}

// WithEmbedding 注册嵌入模型（可选）
func WithEmbedding(emb adapter.Embedder) Option {
	return common.WrapImplSpecificOptFn(func(o *options) {
		o.embedder = emb
	})
}

// WithLexicalIndex 注册词面索引（可选）
func WithLexicalIndex(index RouteLexicalIndex) Option {
	return common.WrapImplSpecificOptFn(func(o *options) {
		o.lexicalIndex = index
	})
}

// WithScorer 注册评分器（可选）
func WithScorer(scorer score.Scorer) Option {
	return common.WrapImplSpecificOptFn(func(o *options) {
		if scorer != nil {
			o.scorer = scorer
		}
	})
}
