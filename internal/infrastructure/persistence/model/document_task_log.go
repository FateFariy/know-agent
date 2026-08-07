package model

import "github.com/swiftbit/know-agent/common"

// DocumentTaskLog 任务日志实体
type DocumentTaskLog struct {
	common.Model
	TaskId       int64  `gorm:"column:task_id;type:bigint"`     // 任务ID
	DocumentId   int64  `gorm:"column:document_id;type:bigint"` // 文档ID
	StageType    int    `gorm:"column:stage_type;type:int"`     // 阶段类型
	EventType    int    `gorm:"column:event_type;type:int"`     // 事件类型
	LogLevel     int    `gorm:"column:log_level;type:int"`      // 日志级别
	OperatorType int    `gorm:"column:operator_type;type:int"`  // 操作人类型
	OperatorId   int64  `gorm:"column:operator_id;type:bigint"` // 操作人ID
	Content      string `gorm:"column:content;type:text"`       // 内容
	DetailJson   string `gorm:"column:detail_json;type:text"`   // 详情JSON
}
