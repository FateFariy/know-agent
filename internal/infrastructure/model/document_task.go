package model

import (
	"time"

	"github.com/swiftbit/know-agent/common"
)

// DocumentTask 文档任务实体
type DocumentTask struct {
	common.Model
	DocumentId       int64     `gorm:"column:document_id"`       // 文档ID
	PlanId           int64     `gorm:"column:plan_id"`           // 方案ID
	TaskType         int       `gorm:"column:task_type"`         // 任务类型
	TaskStatus       int       `gorm:"column:task_status"`       // 任务状态
	CurrentStage     int       `gorm:"column:current_stage"`     // 当前阶段
	TriggerSource    int       `gorm:"column:trigger_source"`    // 触发来源
	StrategySnapshot string    `gorm:"column:strategy_snapshot"` // 策略快照
	RetryCount       int       `gorm:"column:retry_count"`       // 重试次数
	StartTime        time.Time `gorm:"column:start_time"`        // 开始时间
	FinishTime       time.Time `gorm:"column:finish_time"`       // 完成时间
	CostMillis       int64     `gorm:"column:cost_millis"`       // 耗时(毫秒)
	ErrorCode        string    `gorm:"column:error_code"`        // 错误码
	ErrorMsg         string    `gorm:"column:error_msg"`         // 错误信息
	ExtJson          string    `gorm:"column:ext_json"`          // 扩展JSON
}
