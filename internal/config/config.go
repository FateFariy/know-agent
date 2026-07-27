package config

import (
	"time"

	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/rest"
)

type Config struct {
	Http             rest.RestConf
	Auth             AuthConf
	Mysql            MysqlConf
	Redis            cache.CacheConf
	Minio            MinioConf
	Neo4j            Neo4jConf
	MQ               MQConf
	StructureParsing StructureParsingConf
	Embedding        EmbeddingConf
	Chunk            ChunkConf
	Milvus           MilvusConf
	Chat             ChatConf
	ChatModel        map[string]*LLMConf
}

type AuthConf struct {
	AccessSecret string
	AccessExpire time.Duration
}

type MysqlConf struct {
	Endpoint string `json:",default=127.0.0.1:3306"`
	Username string
	Password string
	DbName   string
}

type MinioConf struct {
	Addr             string `json:",default=127.0.0.1:9000"`
	AccessKeyID      string `json:",default=minioadmin"`
	SecretAccessKey  string `json:",default=minioadmin"`
	BucketName       string `json:",default=super-agent-document"`
	ObjectPrefix     string `json:",default=rag/document"`
	ParsedTextPrefix string `json:",default=rag/parsed-text"`
	UseSSL           bool   `json:",default=false"`
}

type Neo4jConf struct {
	Enabled             bool   `json:",default=false"`
	Uri                 string `json:",default=bolt://127.0.0.1:7687"`
	Username            string `json:",default=neo4j"`
	Password            string `json:",default=neo4j"`
	Database            string `json:",default=neo4j"`
	QueryTimeoutSeconds int    `json:",default=5"`
}

type MilvusConf struct {
	Addr       string `json:",default=127.0.0.1:19530"`
	Username   string `json:",optional"`
	Password   string `json:",optional"`
	Collection string
	MetricType string `json:",default=COSINE"`
}

type MQConf struct {
	Addr       string `json:",default=127.0.0.1"`
	ParseTopic string `json:",default=know-agent-document"`
	IndexTopic string `json:",default=know-agent-index"`
	Retry      int    `json:",default=3"`
	Enabled    bool   `json:",default=false"`
}

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

// EmbeddingConf 嵌入配置
type EmbeddingConf struct {
	Model      string // 模型名称
	APIKey     string // API密钥
	APIType    string `json:",default=text,options=text|multi_modal"` // API类型
	Dimensions int    // 嵌入维度
}

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
	HistorySummary           HistorySummaryConf `json:",optional"`                    // 历史摘要配置
	RewriteHistoryTurns      int                `json:",default=4"`                   // 重写历史轮次
	RecentTranscriptMaxChars int                `json:",default=1024"`                // 最近对话记录最大字符数
	QuestionHistoryMaxChars  int                `json:",default=512"`                 // 问题历史最大字符数
	RewriteEnabled           bool               `json:",default=true"`                // 是否启用问题改写
	MaxSubQuestions          int                `json:",default=5"`                   // 最大子问题数量
}

// HistorySummaryConf 历史摘要配置
type HistorySummaryConf struct {
	Enabled               bool `json:",default=true"` // 是否启用摘要压缩
	KeepRecentTurns       int  `json:",default=3"`    // 保留最近轮次
	CompressionBatchTurns int  `json:",default=3"`    // 压缩批次轮次
	SummaryMaxChars       int  `json:",default=1024"` // 摘要最大字符数
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
	Enabled                        bool          `json:",default=true"` // 是否启用RAG
	RerankEnabled                  bool          `json:",default=true"` // 是否启用重排序
	NoEvidenceReply                string        `json:",optional"`     // 无证据时的回复
	SystemPrompt                   string        `json:",optional"`     // 系统提示词
	ChannelTimeout                 time.Duration `json:",default=5s"`   // 通道超时时间
	SubQuestionTimeout             time.Duration `json:",default=12s"`  // 子问题超时时间
	KeywordTopK                    int           `json:",default=8"`    // 关键词检索TopK
	VectorTopK                     int           `json:",default=8"`    // 向量检索TopK
	CandidateTopK                  int           `json:",default=10"`   // 候选项TopK
	FinalTopK                      int           `json:",default=5"`    // 最终选项TopK
	ParentEvidenceMaxChars         int           `json:",default=1024"` // 父证据最大字符数
	MinVectorSimilarity            float64       `json:",default=0.5"`  // 向量相似度阈值
	KeywordRelativeScoreFloor      float64       `json:",default=0.35"` // 关键词相对分数阈值
	RerankScoreThreshold           float64       `json:",default=0.7"`  // 重排序分数阈值
	PlanningHistoryMaxChars        int           `json:",default=2000"` // 规划历史最大字符数
	QuestionHistoryMaxChars        int           `json:",default=1000"` // 问题历史最大字符数
	TotalEvidenceMaxChars          int           `json:",default=5200"` // 总证据最大字符数
	PerSubQuestionEvidenceMaxChars int           `json:",default=2200"` // 每个子问题最大字符数
	Rerank                         RerankConf
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
	TopN           int           `json:",default=5"`    // 重排序TopN
	ConnectTimeout time.Duration `json:",default=3s"`   // 连接超时时间
	ReadTimeout    time.Duration `json:",default=6s"`   // 读取超时时间
}
