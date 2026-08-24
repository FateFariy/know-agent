package model

import (
	"github.com/swiftbit/know-agent/common"
)

type ChatExchangeEval struct {
	common.Model
	ConversationId string  `gorm:"column:conversation_id;type:varchar(255)"` // 会话ID
	ExchangeId     int64   `gorm:"column:exchange_id;type:bigint"`           // 对话轮次ID
	MetricName     string  `gorm:"column:metric_name;type:varchar(100)"`     // 评估指标名称
	MetricLabel    string  `gorm:"column:metric_label;type:varchar(100)"`    // 评估指标标签
	Score          float64 `gorm:"column:score;type:decimal(10,4)"`          // 评估得分
	LatencyMs      int64   `gorm:"column:latency_ms;type:bigint"`            // 评估耗时（毫秒）
	Status         int8    `gorm:"column:status;type:tinyint"`               // 评估状态（1成功，0失败）
	ErrorMsg       *string `gorm:"column:error_msg;type:varchar(512)"`       // 错误信息，空表示无异常
}
