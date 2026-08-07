package model

import (
	"time"

	"github.com/swiftbit/know-agent/common"
)

// DocumentTask 文档任务实体
type DocumentTask struct {
	common.Model
	DocumentId        int64      `gorm:"column:document_id;type:bigint"`          // 文档ID
	PlanId            int64      `gorm:"column:plan_id;type:bigint"`              // 计划ID
	SourceParseTaskId int64      `gorm:"column:source_parse_task_id;type:bigint"` // 源解析任务ID
	TaskType          int        `gorm:"column:task_type;type:int"`               // 任务类型
	TaskStatus        int        `gorm:"column:task_status;type:int"`             // 任务状态
	CurrentStage      int        `gorm:"column:current_stage;type:int"`           // 当前阶段
	TriggerSource     int        `gorm:"column:trigger_source;type:int"`          // 触发来源
	StrategySnapshot  string     `gorm:"column:strategy_snapshot;type:text"`      // 策略快照
	RetryCount        int        `gorm:"column:retry_count;type:int"`             // 重试次数
	StartTime         *time.Time `gorm:"column:start_time;type:datetime"`         // 开始时间
	FinishTime        *time.Time `gorm:"column:finish_time;type:datetime"`        // 完成时间
	CostMillis        int64      `gorm:"column:cost_millis;type:bigint"`          // 耗时毫秒
	ErrorCode         *string    `gorm:"column:error_code;type:varchar(50)"`      // 错误码
	ErrorMsg          *string    `gorm:"column:error_msg;type:text"`              // 错误信息
	ExtJson           string     `gorm:"column:ext_json;type:text"`               // 扩展JSON
}
