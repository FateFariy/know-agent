package llm

import "github.com/swiftbit/know-agent/internal/domain/document/logic/process/chunk"

type options struct {
	llmSplitPrompt string
	llmMaxChars    int
	enabled        bool
}

// WithLlmSplitPrompt 设置LLM分块提示词
func WithLlmSplitPrompt(llmSplitPrompt string) chunk.Option {
	return chunk.WrapSpecificOptFn(func(o *options) {
		if llmSplitPrompt == "" {
			llmSplitPrompt = documentLlmSplit
		}
		o.llmSplitPrompt = llmSplitPrompt
	})
}

func WithLlmMaxChars(llmMaxChars int) chunk.Option {
	return chunk.WrapSpecificOptFn(func(o *options) {
		if llmMaxChars <= 50 {
			llmMaxChars = defaultLLMMaxChars
		}
		o.llmMaxChars = llmMaxChars
	})
}

// WithEnabled 设置是否启用LLM分块
func WithEnabled(enabled bool) chunk.Option {
	return chunk.WrapSpecificOptFn(func(o *options) {
		o.enabled = enabled
	})
}
