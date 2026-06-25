package entity

import (
	"time"

	"github.com/duke-git/lancet/v2/slice"

	"github.com/swiftbit/know-agent/internal/domain/document/model/vo"
)

// DocumentTask 文档任务实体
type DocumentTask struct {
	ID               int64              `gorm:"column:id"`                // 主键ID
	DocumentId       int64              `gorm:"column:document_id"`       // 文档ID
	PlanId           int64              `gorm:"column:plan_id"`           // 方案ID
	TaskType         int                `gorm:"column:task_type"`         // 任务类型
	TaskStatus       int                `gorm:"column:task_status"`       // 任务状态
	CurrentStage     int                `gorm:"column:current_stage"`     // 当前阶段
	TriggerSource    int                `gorm:"column:trigger_source"`    // 触发来源
	StrategySnapshot string             `gorm:"column:strategy_snapshot"` // 策略快照
	RetryCount       int                `gorm:"column:retry_count"`       // 重试次数
	StartTime        time.Time          `gorm:"column:start_time"`        // 开始时间
	FinishTime       time.Time          `gorm:"column:finish_time"`       // 完成时间
	CostMillis       int64              `gorm:"column:cost_millis"`       // 耗时(毫秒)
	ErrorCode        string             `gorm:"column:error_code"`        // 错误码
	ErrorMsg         string             `gorm:"column:error_msg"`         // 错误信息
	ExtJson          string             `gorm:"column:ext_json"`          // 扩展JSON
	TaskTypeName     string             `gorm:"column:-"`                 // 任务类型名称
	TaskStatusName   string             `gorm:"column:-"`                 // 任务状态名称
	CurrentStageName string             `gorm:"column:-"`                 // 当前阶段名称
	Logs             []*DocumentTaskLog `gorm:"column:-"`                 // 日志
}

func (d *DocumentTask) FillEnumNames() {
	d.TaskTypeName = vo.TaskTypeName(d.TaskType)
	d.TaskStatusName = vo.TaskStatusName(d.TaskStatus)
	d.CurrentStageName = vo.TaskStageName(d.CurrentStage)
	slice.ForEach(d.Logs, func(index int, log *DocumentTaskLog) {
		log.FillEnumNames()
	})
}
