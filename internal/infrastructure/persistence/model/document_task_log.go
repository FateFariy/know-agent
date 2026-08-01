package model

import "github.com/swiftbit/know-agent/common"

// DocumentTaskLog 任务日志实体
type DocumentTaskLog struct {
	common.Model
	TaskId       int64  `gorm:"column:task_id"`       // 任务ID
	DocumentId   int64  `gorm:"column:document_id"`   // 文档ID
	StageType    int    `gorm:"column:stage_type"`    // 阶段类型
	EventType    int    `gorm:"column:event_type"`    // 事件类型
	LogLevel     int    `gorm:"column:log_level"`     // 日志级别
	OperatorType int    `gorm:"column:operator_type"` // 操作人类型
	OperatorId   int64  `gorm:"column:operator_id"`   // 操作人ID
	Content      string `gorm:"column:content"`       // 日志内容
	DetailJson   string `gorm:"column:detail_json"`   // 详情JSON
}
