package window

import "github.com/swiftbit/know-agent/internal/domain/document/logic/process/chunk"

const (
	defaultMaxChars = 800 // 默认最大字符数
)

type options struct {
	maxChars int
}

// WithMaxChars 设置单个块的最大字符数
func WithMaxChars(maxChars int) chunk.Option {
	return chunk.WrapSpecificOptFn(func(o *options) {
		o.maxChars = maxChars
	})
}
