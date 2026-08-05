package config

type LLMConf struct {
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
	APIKey     string // API密钥
	APIType    string `json:",default=text,options=text|multi_modal"` // API类型
	Dimensions int    // 嵌入维度
}
