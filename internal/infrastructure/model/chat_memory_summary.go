package model

import "time"

// ChatMemorySummary 会话记忆摘要模型
type ChatMemorySummary struct {
	ID                   int64     `gorm:"column:id;primaryKey"`        // ID
	ConversationId       string    `gorm:"column:conversation_id"`     // 会话ID
	CoveredExchangeId    int64     `gorm:"column:covered_exchange_id"` // 覆盖的最后交换ID
	CoveredExchangeCount int       `gorm:"column:covered_exchange_count"` // 覆盖的交换数量
	CompressionCount     int       `gorm:"column:compression_count"`   // 压缩次数
	SummaryVersion       int       `gorm:"column:summary_version"`     // 摘要版本
	SummaryText          string    `gorm:"column:summary_text"`        // 摘要文本
	SummaryJson          string    `gorm:"column:summary_json"`        // 摘要JSON
	LastSourceEditTime   time.Time `gorm:"column:last_source_edit_time"` // 最后源编辑时间
	Status               int       `gorm:"column:status"`              // 状态
	CreateTime           time.Time `gorm:"column:create_time"`         // 创建时间
	UpdateTime           time.Time `gorm:"column:update_time"`         // 更新时间
}