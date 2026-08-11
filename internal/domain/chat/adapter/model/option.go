package model

import (
	"github.com/swiftbit/know-agent/common"
)

type Options struct {
	Model       string
	Temperature *float32
	TopP        *float32
	MaxTokens   int
	Callback    func()
}

type Option = common.Option

func WithCallback(callback func()) Option {
	return common.WrapImplSpecificOptFn(func(opt *Options) {
		if callback != nil {
			opt.Callback = callback
		}
	})
}

// WithModel 设置模型名称
func WithModel(model string) Option {
	return common.WrapImplSpecificOptFn(func(opt *Options) {
		opt.Model = model
	})
}

// WithTemperature 设置温度参数（0~1，控制随机性）
func WithTemperature(temp float32) Option {
	return common.WrapImplSpecificOptFn(func(opt *Options) {
		opt.Temperature = &temp
	})
}

// WithTopP 设置 Top-P 采样参数（0~1，控制词汇多样性）
func WithTopP(topP float32) Option {
	return common.WrapImplSpecificOptFn(func(opt *Options) {
		opt.TopP = &topP
	})
}
