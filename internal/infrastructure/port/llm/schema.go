package llm

// ModelCallMeta 模型调用通用元信息
type ModelCallMeta struct {
	Stage     string
	Provider  string
	ModelName string
}

// ModelCallInput OnStart 专属字段，仅在输入阶段使用
type ModelCallInput struct {
	Temperature float32
	TopP        float32
}

// ModelCallOutput OnEnd 专属字段，包含响应和 token 估算所需的 prompt
type ModelCallOutput struct {
	SystemPrompt string
	UserPrompt   string
	Response     any
	ResponseText string
}
