package model

type Options struct {
	Temperature float32
	Model       string
	TopP        float32
}

type Option func(opt *Options)

func WithModel(model string) Option {
	return func(opt *Options) {
		opt.Model = model
	}
}

func WithTemperature(temp float32) Option {
	return func(opt *Options) {
		opt.Temperature = temp
	}
}

func WithTopP(topP float32) Option {
	return func(opt *Options) {
		opt.TopP = topP
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
