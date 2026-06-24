package config

import (
	"time"

	"github.com/swiftbit/know-agent/common"
)

type Config struct {
	common.BaseConfig
	Minio     MinioConf
	Neo4j     Neo4jConf
	MQ        MQConf
	Embedding EmbeddingConf
	Milvus    MilvusConf
	Chat      ChatConf
	ChatModel map[string]*LLMConf
}
type MinioConf struct {
	Endpoint         string `json:",omitempty,default=http://127.0.0.1:9000"`
	AccessKeyID      string `json:",omitempty,default=minioadmin"`
	SecretAccessKey  string `json:",omitempty,default=minioadmin"`
	BucketName       string `json:",omitempty,default=super-agent-document"`
	ObjectPrefix     string `json:",omitempty,default=rag/document"`
	ParsedTextPrefix string `json:",omitempty,default=rag/parsed-text"`
	UseSSL           bool   `json:",omitempty,default=false"`
}

type Neo4jConf struct {
	Enabled             bool   `json:",omitempty,default=false"`
	Uri                 string `json:",omitempty,default=bolt://127.0.0.1:7687"`
	Username            string `json:",omitempty,default=neo4j"`
	Password            string `json:",omitempty,default=neo4j"`
	Database            string `json:",omitempty,default=neo4j"`
	QueryTimeoutSeconds int    `json:",omitempty,default=5"`
}

type MilvusConf struct {
	Addr     string `json:",omitempty,default=127.0.0.1:19530"`
	Username string `json:",omitempty,optional"`
	Password string `json:",omitempty,optional"`
}

type MQConf struct {
	ParseTopic string `json:",omitempty,default=know-agent-document"`
	IndexTopic string `json:",omitempty,default=know-agent-index"`
	Enabled    bool   `json:",omitempty,default=false"`
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

type ChatConf struct {
	RewriteEnabled        bool               `json:",optional,default=true"` // 是否启用问题改写
	RecommendationEnabled bool               `json:",optional,default=true"` // 是否启用推荐追问
	Memory                MemoryConf         // 记忆配置
	Rewrite               RewriteConf        // 问题改写配置
	Recommendation        RecommendationConf // 推荐配置
	Rag                   RagConf            // RAG配置
}

// RecommendationConf 推荐追问配置
type RecommendationConf struct {
	Enabled             bool          `json:",optional,default=true"` // 是否启用推荐追问
	Timeout             time.Duration `json:",optional,default=5s"`   // 推荐生成超时时间
	HistoryPreviewTurns int           `json:",optional,default=3"`    // 预览历史轮次
}

// MemoryConf 记忆配置
type MemoryConf struct {
	StrategyType             string             `json:",optional,default=summary_compression"` // 记忆策略类型: sliding_window 或 summary_compression
	HistorySummary           HistorySummaryConf `json:",optional"`                             // 历史摘要配置
	RewriteHistoryTurns      int                `json:",optional,default=4"`                   // 重写历史轮次
	RecentTranscriptMaxChars int                `json:",optional,default=1024"`                // 最近对话记录最大字符数
	QuestionHistoryMaxChars  int                `json:",optional,default=512"`                 // 问题历史最大字符数
	RewriteEnabled           bool               `json:",optional,default=true"`                // 是否启用问题改写
	MaxSubQuestions          int                `json:",optional,default=5"`                   // 最大子问题数量
}

// HistorySummaryConf 历史摘要配置
type HistorySummaryConf struct {
	Enabled               bool `json:",optional,default=true"` // 是否启用摘要压缩
	KeepRecentTurns       int  `json:",optional,default=3"`    // 保留最近轮次
	CompressionBatchTurns int  `json:",optional,default=3"`    // 压缩批次轮次
	SummaryMaxChars       int  `json:",optional,default=1024"` // 摘要最大字符数
}

// RewriteConf 问题改写配置
type RewriteConf struct {
	Enabled         bool    `json:",optional,default=true"`  // 是否启用问题改写
	MaxSubQuestions int     `json:",optional,default=5"`     // 最大子问题数量
	Temperature     float32 `json:",optional,default=0.1"`   // 温度参数
	TopP            float32 `json:",optional,default=0.3"`   // TopP参数
	Thinking        bool    `json:",optional,default=false"` // 是否启用思考过程
}

// RagConf RAG配置
type RagConf struct {
	Enabled                   bool          `json:",optional,default=true"` // 是否启用RAG
	RerankEnabled             bool          `json:",optional,default=true"` // 是否启用重排序
	NoEvidenceReply           string        `json:",optional"`              // 无证据时的回复
	ChannelTimeout            time.Duration `json:",optional,default=5s"`   // 通道超时时间
	SubQuestionTimeout        time.Duration `json:",optional,default=12s"`  // 子问题超时时间
	KeywordTopK               int           `json:",optional,default=8"`    // 关键词检索TopK
	VectorTopK                int           `json:",optional,default=8"`    // 向量检索TopK
	CandidateTopK             int           `json:",optional,default=10"`   // 候选项TopK
	FinalTopK                 int           `json:",optional,default=5"`    // 最终选项TopK
	ParentEvidenceMaxChars    int           `json:",optional,default=1024"` // 父证据最大字符数
	MinVectorSimilarity       float64       `json:",optional,default=0.5"`  // 向量相似度阈值
	KeywordRelativeScoreFloor float64       `json:",optional,default=0.35"` // 关键词相对分数阈值
	PlanningHistoryMaxChars   int           `json:",optional,default=2000"` // 规划历史最大字符数
	QuestionHistoryMaxChars   int           `json:",optional,default=1000"` // 问题历史最大字符数
}

// EmbeddingConf 嵌入配置
type EmbeddingConf struct {
	Model      string // 模型名称
	APIKey     string // API密钥
	APIType    string `json:",omitempty,default=text,options=text|multi_model"` // API类型
	Dimensions int    // 嵌入维度
}

func (c Config) GetBaseConfig() *common.BaseConfig {
	return &c.BaseConfig
}
