package model

import (
	"github.com/swiftbit/know-agent/common"
)

type Options struct {
	Function    string
	Model       string
	Temperature *float32
	TopP        *float32
	MaxTokens   int

	Think string // 思考过程, true/false,"high"、"medium"、"low"
}

type Option = common.Option

func WithFunction(function string) Option {
	return common.WrapImplSpecificOptFn(func(opt *Options) {
		opt.Function = function
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

// WithMaxTokens 设置单次调用的最大输出 token 数
func WithMaxTokens(maxTokens int) Option {
	return common.WrapImplSpecificOptFn(func(opt *Options) {
		opt.MaxTokens = maxTokens
	})
}

// WithThink 设置思考过程
func WithThink(think string) Option {
	return common.WrapImplSpecificOptFn(func(opt *Options) {
		opt.Think = think
	})
}
