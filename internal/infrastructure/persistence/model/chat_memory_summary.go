package model

import (
	"time"

	"github.com/swiftbit/know-agent/common"
)

// ChatMemorySummary 会话记忆摘要
type ChatMemorySummary struct {
	common.Model
	ConversationId       string    `gorm:"column:conversation_id;type:varchar(64)"`      // 对话ID
	CoveredExchangeId    int64     `gorm:"column:covered_exchange_id;type:bigint"`       // 覆盖的交互ID
	CoveredExchangeCount int       `gorm:"column:covered_exchange_count;type:int"`       // 覆盖交互数量
	CompressionCount     int       `gorm:"column:compression_count;type:int"`            // 压缩次数
	SummaryVersion       int       `gorm:"column:summary_version;type:int"`              // 摘要版本
	SummaryText          string    `gorm:"column:summary_text;type:text"`                // 摘要文本
	SummaryJson          string    `gorm:"column:summary_json;type:text"`                // 摘要JSON
	LastSourceUpdateTime time.Time `gorm:"column:last_source_update_time;type:datetime"` // 源数据最后编辑时间
}
