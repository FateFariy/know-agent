package recursive

import (
	"github.com/swiftbit/know-agent/common"
)

const (
	defaultMaxChars     = 800 // 默认最大字符数
	defaultOverlapChars = 120 // 默认重叠字符数
)

// options 递归分块策略的参数配置
type options struct {
	maxChars     int
	overlapChars int
}

// WithMaxChars 设置单个块的最大字符数
func WithMaxChars(maxChars int) common.Option {
	return common.WrapImplSpecificOptFn(func(o *options) {
		if maxChars <= 0 {
			maxChars = defaultMaxChars
		}
		o.maxChars = maxChars
	})
}

// WithOverlapChars 设置相邻块的重叠字符数
func WithOverlapChars(overlapChars int) common.Option {
	return common.WrapImplSpecificOptFn(func(o *options) {
		if overlapChars < 0 {
			overlapChars = defaultOverlapChars
		}
		o.overlapChars = overlapChars
	})
}
