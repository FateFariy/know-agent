package entity

import (
	"time"
)

// ChatExchangeEval 对话级 RAG 质量评估结果
// 每次问答自动计算忠实度/相关性/上下文精度等指标并落库，支持按会话回溯质量趋势。
type ChatExchangeEval struct {
	ID             int64         `gorm:"column:id"`              // 主键ID
	ConversationId string        `gorm:"column:conversation_id"` // 对话ID
	ExchangeId     int64         `gorm:"column:exchange_id"`     // 交互ID
	MetricName     string        `gorm:"column:metric_name"`     // 指标编码（如 AnswerFaithfulness）
	MetricLabel    string        `gorm:"column:metric_label"`    // 指标展示名
	Score          float64       `gorm:"column:score"`           // 评估得分（0~1）
	LatencyMs      time.Duration `gorm:"column:latency_ms"`      // 评估耗时（ms）
	Status         int8          `gorm:"column:status"`          // 评估状态（0成功 1失败）
	ErrorMsg       *string       `gorm:"column:error_msg"`       // 错误信息
}
