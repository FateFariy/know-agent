package config

// StructureParsingConf 结构解析配置
type StructureParsingConf struct {
	LLMDisambiguationEnabled   bool    `json:",default=true"` // 是否启用 LLM 消歧能力，默认启用
	MaxAmbiguousSignalsPerCall int     `json:",default=8"`    // 单次 LLM 调用最多处理的歧义信号数量，默认 8
	ContextWindowLines         int     `json:",default=2"`    // 歧义判定时参考的上下文窗口行数，默认 2 行
	MaxPlainHeadingChars       int     `json:",default=32"`   // 纯文本标题判定的最大字符数阈值，默认 32
	AmbiguityConfidenceFloor   float64 `json:",default=0.45"` // 歧义置信度下限，低于该值判定为明确非标题，默认 0.45
	AmbiguityConfidenceCeil    float64 `json:",default=0.80"` // 歧义置信度上限，高于该值判定为明确标题，默认 0.80
}

// ChunkConf 文档分块配置
type ChunkConf struct {
	RecursiveMaxChars           int     `json:",default=800"`   // 递归分块的最大字符数，默认800
	RecursiveOverlapChars       int     `json:",default=120"`   // 递归分块的重叠字符数，默认120
	SemanticMaxChars            int     `json:",default=700"`   // 语义分块的最大字符数，默认700
	SemanticMinChars            int     `json:",default=240"`   // 语义分块的最小字符数，默认240
	SemanticSimilarityThreshold float64 `json:",default=0.18"`  // 语义相似度阈值，默认0.18
	LlmEnabled                  bool    `json:",default=false"` // 是否启用大模型分块，默认false
	LlmMaxChars                 int     `json:",default=3500"`  // 大模型分块的最大字符数，默认3500
	RecommendLlmWhenLowQuality  bool    `json:",default=true"`  // 低质量时是否推荐使用大模型分块，默认true
}
