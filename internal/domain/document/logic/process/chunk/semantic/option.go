package semantic

import (
	"github.com/swiftbit/know-agent/internal/domain/document/logic/process/chunk"
	"github.com/swiftbit/know-agent/internal/domain/document/logic/process/chunk/semantic/similarity"
)

const (
	defaultMaxChars            = 700  // 默认单个块最大字符数
	defaultMinChars            = 240  // 触发语义切分的最小字符数
	defaultSimilarityThreshold = 0.18 // 默认的 Jaccard 相似度阈值
)

// options 语义分块策略的参数配置
type options struct {
	maxChars            int
	minChars            int
	similarityThreshold float64
	calculator          similarity.Calculator
}

// WithMaxChars 设置单个块的最大字符数
func WithMaxChars(maxChars int) chunk.Option {
	return chunk.WrapSpecificOptFn(func(o *options) {
		if maxChars <= 0 {
			maxChars = defaultMaxChars
		}
		o.maxChars = maxChars
	})
}

// WithMinChars 设置触发语义切分的最小字符数
func WithMinChars(minChars int) chunk.Option {
	return chunk.WrapSpecificOptFn(func(o *options) {
		if minChars <= 0 {
			minChars = defaultMinChars
		}
		o.minChars = minChars
	})
}

// WithSimilarityThreshold 设置语义相似度阈值
func WithSimilarityThreshold(threshold float64) chunk.Option {
	return chunk.WrapSpecificOptFn(func(o *options) {
		if threshold <= 0 || threshold >= 1 {
			threshold = defaultSimilarityThreshold
		}
		o.similarityThreshold = threshold
	})
}

// WithSimilarity 注入自定义的相似度计算实现
func WithSimilarity(calculator similarity.Calculator) chunk.Option {
	return chunk.WrapSpecificOptFn(func(o *options) {
		if calculator == nil {
			return
		}
		o.calculator = calculator
	})
}
