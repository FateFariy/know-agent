package model

import (
	"time"

	"github.com/swiftbit/know-agent/common"
)

// ChatChannelExecution 通道执行记录
type ChatChannelExecution struct {
	common.Model
	ConversationId     string    `gorm:"column:conversation_id"`      // 对话ID
	ExchangeId         int64     `gorm:"column:exchange_id"`          // 交换ID
	TraceId            string    `gorm:"column:trace_id"`             // 跟踪ID
	SubQuestionIndex   int       `gorm:"column:sub_question_index"`   // 子问题索引
	SubQuestion        string    `gorm:"column:sub_question"`         // 子问题
	ChannelType        string    `gorm:"column:channel_type"`         // 渠道类型
	ExecutionState     int       `gorm:"column:execution_state"`      // 执行状态
	StartTime          time.Time `gorm:"column:start_time"`           // 开始时间
	EndTime            time.Time `gorm:"column:end_time"`             // 结束时间
	DurationMs         int64     `gorm:"column:duration_ms"`          // 执行时长（毫秒）
	RecalledCount      int       `gorm:"column:recalled_count"`       // 召回数量
	AcceptedCount      int       `gorm:"column:accepted_count"`       // 接受数量
	FinalSelectedCount int       `gorm:"column:final_selected_count"` // 最终选中数量
	AvgScore           float64   `gorm:"column:avg_score"`            // 平均分数
	MaxScore           float64   `gorm:"column:max_score"`            // 最大分数
	MinScore           float64   `gorm:"column:min_score"`            // 最小分数
	ConfigSnapshot     string    `gorm:"column:config_snapshot"`      // 配置快照
	ErrorMessage       string    `gorm:"column:error_message"`        // 错误信息
}
