package model

type Options struct {
	Model       string
	Temperature *float32
	TopP        *float32
	MaxTokens   int
	Callback    func()
}

type Option func(opt *Options)

func WithCallback(callback func()) Option {
	return func(opt *Options) {
		if callback != nil {
			opt.Callback = callback
		}
	}
}

func WithModel(model string) Option {
	return func(opt *Options) {
		opt.Model = model
	}
}

func WithTemperature(temp float32) Option {
	return func(opt *Options) {
		opt.Temperature = &temp
	}
}

func WithTopP(topP float32) Option {
	return func(opt *Options) {
		opt.TopP = &topP
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
