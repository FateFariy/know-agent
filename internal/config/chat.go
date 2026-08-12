package config

import "time"

type ChatConf struct {
	RewriteEnabled        bool               `json:",default=true"` // 是否启用问题改写
	RecommendationEnabled bool               `json:",default=true"` // 是否启用推荐追问
	Memory                MemoryConf         // 记忆配置
	Rewrite               RewriteConf        // 问题改写配置
	Recommendation        RecommendationConf // 推荐配置
	Rag                   RagConf            // RAG配置
	Agent                 AgentConf          // Agent配置
}

// RecommendationConf 推荐追问配置
type RecommendationConf struct {
	Enabled             bool          `json:",default=true"` // 是否启用推荐追问
	Timeout             time.Duration `json:",default=5s"`   // 推荐生成超时时间
	HistoryPreviewTurns int           `json:",default=3"`    // 预览历史轮次
}

// MemoryConf 记忆配置
type MemoryConf struct {
	StrategyType             string             `json:",default=summary_compression"` // 记忆策略类型: sliding_window 或 summary_compression
	RecentTranscriptMaxChars int                `json:",default=1024"`                // 最近对话记录最大字符数
	QuestionHistoryMaxChars  int                `json:",default=512"`                 // 问题历史最大字符数
	RewriteEnabled           bool               `json:",default=true"`                // 是否启用问题改写
	MaxSubQuestions          int                `json:",default=5"`                   // 最大子问题数量
	HistorySummary           HistorySummaryConf // 历史摘要配置
}

// HistorySummaryConf 历史摘要配置
type HistorySummaryConf struct {
	Enabled                  bool `json:",default=true"` // 是否启用摘要压缩
	HistoryTurns             int  `json:",default=6"`    // 历史轮次
	KeepRecentTurns          int  `json:",default=3"`    // 保留最近轮次
	CompressionBatchTurns    int  `json:",default=3"`    // 压缩批次轮次
	MaxChars                 int  `json:",default=1024"` // 摘要最大字符数
	RecentTranscriptMaxChars int  `json:",default=2200"` // 最近对话记录最大字符数
}

// RewriteConf 问题改写配置
type RewriteConf struct {
	Enabled         bool    `json:",default=true"`  // 是否启用问题改写
	MaxSubQuestions int     `json:",default=5"`     // 最大子问题数量
	Temperature     float32 `json:",default=0.1"`   // 温度参数
	TopP            float32 `json:",default=0.3"`   // TopP参数
	Thinking        bool    `json:",default=false"` // 是否启用思考过程
}

// RagConf RAG配置
type RagConf struct {
	Enabled                        bool                  `json:",default=true"`                           // 是否启用RAG
	NoEvidenceReply                string                `json:",default=当前没有从已接入文档中检索到足够证据，暂时不能给出可靠结论。"` // 无证据时的回复
	SystemPrompt                   string                `json:",optional"`                               // 系统提示词（可选）
	ChannelTimeout                 time.Duration         `json:",default=5s"`                             // 通道超时时间
	SubQuestionTimeout             time.Duration         `json:",default=12s"`                            // 子问题超时时间
	CandidateTopK                  int                   `json:",default=40"`                             // 候选集TopK
	FinalTopK                      int                   `json:",default=6"`                              // 最终结果TopK
	ParentEvidenceMaxChars         int                   `json:",default=2200"`                           // 父证据最大字符数
	PlanningHistoryMaxChars        int                   `json:",default=1600"`                           // 规划历史最大字符数
	QuestionHistoryMaxChars        int                   `json:",default=1000"`                           // 问题历史最大字符数
	TotalEvidenceMaxChars          int                   `json:",default=5200"`                           // 总证据最大字符数
	PerSubQuestionEvidenceMaxChars int                   `json:",default=2200"`                           // 每个子问题证据最大字符数
	Keyword                        KeywordConf           // 关键词通道
	Vector                         VectorConf            // 向量通道
	Table                          TableConf             // 表格通道
	GraphRag                       GraphRagConf          // 图RAG通道
	Raptor                         RaptorConf            // RAPTOR通道
	Rerank                         RerankConf            // 重排序通道
	Hybrid                         HybridConf            // 混合检索权重
	GraphRagQueryPlan              GraphRagQueryPlanConf // 图RAG查询计划
	AutoRoute                      AutoRouteConf         // 自动路由配置
}

type KeywordConf struct {
	Enabled            bool    `json:",default=true"` // 是否启用关键词
	TopK               int     `json:",default=10"`   // 关键词检索TopK
	RelativeScoreFloor float64 `json:",default=0.35"` // 关键词相对分数阈值
}

type VectorConf struct {
	Enabled       bool    `json:",default=true"` // 是否启用向量
	TopK          int     `json:",default=10"`   // 向量检索TopK
	MinSimilarity float64 `json:",default=0.45"` // 相似度阈值
}

type TableConf struct {
	Enabled bool `json:",default=true"` // 是否启用表格
}

// GraphRagConf 图RAG相关配置
type GraphRagConf struct {
	Enabled bool `json:",default=true"` // 是否启用图RAG
	TopK    int  `json:",default=5"`    // 图RAG检索TopK
	MaxHops int  `json:",default=2"`    // 最大跳数
}

type RaptorConf struct {
	Enabled             bool    `json:",default=true"` // 是否启用Raptor
	TopK                int     `json:",default=5"`    // 检索TopK
	SourceChunkTopK     int     `json:",default=3"`    // 源块检索TopK
	MaxClusterSize      int     `json:",default=6"`    // 最大聚类大小
	MaxLevels           int     `json:",default=3"`    // 最大层级
	LlmSummaryEnabled   bool    `json:",default=true"` // 启用LLM摘要
	SummaryQualityFloor float64 `json:",default=0.42"` // 摘要质量下限
}

// GraphRagQueryPlanConf 图RAG查询计划
type GraphRagQueryPlanConf struct {
	Enabled bool `json:",default=true"`
}

// HybridConf 混合检索权重
type HybridConf struct {
	VectorWeight        float64 `json:",default=1.0"`
	KeywordWeight       float64 `json:",default=1.0"`
	TableWeight         float64 `json:",default=1.2"`
	GraphRagWeight      float64 `json:",default=1.1"`
	RaptorWeight        float64 `json:",default=1.05"`
	RankWeight          float64 `json:",default=1.0"`
	OriginalScoreWeight float64 `json:",default=0.08"`
	MetadataBoostWeight float64 `json:",default=0.04"`
	MaxMetadataBoost    float64 `json:",default=1.0"`
}

// AutoRouteConf 自动路由
type AutoRouteConf struct {
	RecommendationThreshold float64 `json:",default=0.55"`
}

// AgentConf Agent配置
type AgentConf struct {
	RecommendationEnabled  bool          `json:",default=true"` // 是否启用推荐追问
	MaxModelCallsPerRun    int           `json:",default=8"`    // 每次最大模型调用次数
	MaxModelCallsPerThread int           `json:",default=40"`   // 每次最大模型调用线程数
	MaxToolCallsPerRun     int           `json:",default=6"`    // 每次最大工具调用次数
	MaxToolCallsPerThread  int           `json:",default=30"`   // 每次最大工具调用线程数
	HistoryPreviewTurns    int           `json:",default=4"`    // 预览历史轮次
	RecommendationTimeout  time.Duration `json:",default=5s"`   // 推荐生成超时时间
	SystemPrompt           string        `json:",optional"`     // 系统提示
	RecommendationPrompt   string        `json:",optional"`     // 推荐追问提示
}

type RerankConf struct {
	Enabled        bool          `json:",default=true"` // 是否启用重排序
	URL            string        `json:",optional"`     // 重排序API地址
	ApiKey         string        `json:",optional"`     // 重排序API密钥
	Model          string        `json:",optional"`     // 重排序模型
	TopN           int           `json:",default=24"`   // 重排序TopN
	ConnectTimeout time.Duration `json:",default=3s"`   // 连接超时时间
	ReadTimeout    time.Duration `json:",default=6s"`   // 读取超时时间
	ScoreThreshold float64       `json:",default=0.0"`  // 重排序分数阈值
}
