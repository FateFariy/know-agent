package config

// ChunkConf 文档分块配置
type ChunkConf struct {
	RecursiveMaxChars           int     `json:",default=800"`   // 递归分块的最大字符数，默认800
	RecursiveOverlapChars       int     `json:",default=120"`   // 递归分块的重叠字符数，默认120
	SemanticMaxChars            int     `json:",default=700"`   // 语义分块的最大字符数，默认700
	SemanticMinChars            int     `json:",default=240"`   // 语义分块的最小字符数，默认240
	ParentChunkMaxChars         int     `json:",default=2200"`  // 父块的最大字符数，默认2200
	ParentChunkOverlapChars     int     `json:",default=180"`   // 父块的重叠字符数，默认180
	ParentSemanticMaxChars      int     `json:",default=1600"`  // 父块的语义最大字符数，默认1600
	ParentSemanticMinChars      int     `json:",default=480"`   // 父块的语义最小字符数，默认480
	SemanticSimilarityThreshold float64 `json:",default=0.18"`  // 语义相似度阈值，默认0.18
	LlmEnabled                  bool    `json:",default=false"` // 是否启用大模型分块，默认false
	LlmMaxChars                 int     `json:",default=3500"`  // 大模型分块的最大字符数，默认3500
	RecommendLlmWhenLowQuality  bool    `json:",default=true"`  // 低质量时是否推荐使用大模型分块，默认true
}

type GseConfig struct {
	DictPath string `json:",optional"` // 词典文件路径
	StopPath string `json:",optional"` // 停用词文件路径
	UseHMM   bool   `json:",optional"` // 是否默认启用 HMM
	AlphaNum bool   `json:",optional"` // 是否保留字母数字组合
}
