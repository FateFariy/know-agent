package config

type LLMConf struct {
	Function          string
	Model             string
	ApiKey            string
	BaseURL           string
	Temperature       float32 `json:",optional"`
	MaxTokens         int     `json:",optional"`
	TopP              float32 `json:",optional"`
	InputTokenCost1k  float64 `json:",optional"`
	OutputTokenCost1k float64 `json:",optional"`
}

// EmbeddingConf 嵌入配置
type EmbeddingConf struct {
	Model      string // 模型名称
	Dimensions int    // 嵌入维度
	BaseURL    string `json:",default=http://localhost:11434"` // Ollama 服务地址
}
