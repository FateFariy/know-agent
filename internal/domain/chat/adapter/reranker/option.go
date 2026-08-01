package reranker

type Options struct {
	Model string // 重排序模型
	TopN  int    // 重排序TopN
}

type Option func(opt *Options)

func WithModel(model string) Option {
	return func(opt *Options) {
		opt.Model = model
	}
}

func WithTopN(topN int) Option {
	return func(opt *Options) {
		opt.TopN = max(1, topN)
	}
}

func GetOptions(base *Options, opts ...Option) *Options {
	if base == nil {
		base = &Options{}
	}

	for _, opt := range opts {
		opt(base)
	}

	return base
}
