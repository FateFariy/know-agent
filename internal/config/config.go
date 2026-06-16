package config

import (
	"github.com/swiftbit/know-agent/common"
)

type Config struct {
	common.BaseConfig
	Minio     MinioConf
	Neo4j     Neo4jConf
	MQ        MQConf
	Memory    MemoryConf
	ChatModel []LLMConf
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

type MQConf struct {
	ParseTopic string `json:",omitempty,default=know-agent-document"`
	IndexTopic string `json:",omitempty,default=know-agent-index"`
	Enabled    bool   `json:",omitempty,default=false"`
}

type LLMConf struct {
	Model   string
	ApiKey  string
	BaseURL string
}

// HistorySummaryConf 历史摘要配置
type HistorySummaryConf struct {
	Enabled                  bool `json:",optional,default=true"` // 是否启用摘要压缩
	KeepRecentTurns          int  `json:",optional,default=3"`    // 保留最近轮次
	CompressionBatchTurns    int  `json:",optional,default=3"`    // 压缩批次轮次
	SummaryMaxChars          int  `json:",optional,default=1024"` // 摘要最大字符数
	RecentTranscriptMaxChars int  `json:",optional,default=1024"` // 最近对话记录最大字符数
}

// MemoryConf 记忆配置
type MemoryConf struct {
	StrategyType            string             `json:",optional,default=summary_compression"` // 记忆策略类型: sliding_window 或 summary_compression
	HistorySummary          HistorySummaryConf `json:",optional"`                             // 历史摘要配置
	RewriteHistoryTurns     int                `json:",optional,default=4"`                   // 重写历史轮次
	QuestionHistoryMaxChars int                `json:",optional,default=512"`                 // 问题历史最大字符数
}

func (c Config) GetBaseConfig() *common.BaseConfig {
	return &c.BaseConfig
}
