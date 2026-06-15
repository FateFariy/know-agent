package entity

import "time"

// ChatMemorySummary 会话记忆摘要实体
type ChatMemorySummary struct {
	ID                   int64     // ID
	ConversationId       string    // 会话ID
	CoveredExchangeId    int64     // 覆盖的最后交换ID
	CoveredExchangeCount int       // 覆盖的交换数量
	CompressionCount     int       // 压缩次数
	SummaryVersion       int       // 摘要版本
	SummaryText          string    // 摘要文本
	SummaryJson          string    // 摘要JSON
	LastSourceEditTime   time.Time // 最后源编辑时间
	Status               int       // 状态
	CreateTime           time.Time // 创建时间
	UpdateTime           time.Time // 更新时间
}
