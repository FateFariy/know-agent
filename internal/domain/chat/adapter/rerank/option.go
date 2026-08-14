package rerank

import "github.com/swiftbit/know-agent/common"

type Options struct {
	Model string // 重排序模型
	TopN  int    // 重排序TopN
}

type Option = common.Option

func WithModel(model string) Option {
	return common.WrapImplSpecificOptFn(func(opt *Options) {
		opt.Model = model
	})
}

func WithTopN(topN int) Option {
	return common.WrapImplSpecificOptFn(func(opt *Options) {
		opt.TopN = max(1, topN)
	})
}
