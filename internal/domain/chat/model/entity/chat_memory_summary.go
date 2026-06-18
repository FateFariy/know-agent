package entity

import (
	"time"
)

// ChatMemorySummary 会话记忆摘要
type ChatMemorySummary struct {
	ID                   int64                `gorm:"column:id"`                      // 主键ID
	ConversationId       string               `gorm:"column:conversation_id"`         // 会话ID
	CoveredExchangeId    int64                `gorm:"column:covered_exchange_id"`     // 覆盖的最后轮次ID
	CoveredExchangeCount int                  `gorm:"column:covered_exchange_count"`  // 覆盖的轮数
	CompressionCount     int                  `gorm:"column:compression_count"`       // 压缩次数
	SummaryVersion       int                  `gorm:"column:summary_version"`         // 摘要版本
	SummaryText          string               `gorm:"column:summary_text"`            // 摘要文本
	SummaryJson          string               `gorm:"column:summary_json"`            // 摘要JSON
	LastSourceUpdateTime time.Time            `gorm:"column:last_source_update_time"` // 最后源更新时间
	UpdateTime           time.Time            `gorm:"column:update_time"`             // 更新时间
	IsCompressed         bool                 `gorm:"-"`                              // 是否已应用历史压缩
	Summary              *ConversationSummary `gorm:"-"`                              // 会话摘要
}

// ConversationSummary 会话摘要
type ConversationSummary struct {
	Summary          string   // 摘要
	ConversationGoal string   // 会话目标
	StableFacts      []string // 稳定事实
	UserPreferences  []string // 用户偏好
	ResolvedPoints   []string // 解决的点
	PendingQuestions []string // 待解决的问题
	RetrievalHints   []string // 检索提示
}
