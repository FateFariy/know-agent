package model

import (
	"time"

	"github.com/swiftbit/know-agent/common"
)

// DocumentStrategyPlan 策略方案
type DocumentStrategyPlan struct {
	common.Model
	DocumentId       int64      `gorm:"column:document_id;type:bigint"`     // 文档ID
	PlanVersion      int        `gorm:"column:plan_version;type:int"`       // 计划版本
	PlanSource       int        `gorm:"column:plan_source;type:int"`        // 计划来源
	PlanStatus       int        `gorm:"column:plan_status;type:int"`        // 计划状态
	StrategyCount    int        `gorm:"column:strategy_count;type:int"`     // 策略数量
	StrategySnapshot string     `gorm:"column:strategy_snapshot;type:text"` // 策略快照
	RecommendReason  string     `gorm:"column:recommend_reason;type:text"`  // 推荐理由
	AdjustNote       string     `gorm:"column:adjust_note;type:text"`       // 调整备注
	ConfirmUserId    int64      `gorm:"column:confirm_user_id;type:bigint"` // 确认用户ID
	ConfirmTime      *time.Time `gorm:"column:confirm_time;type:datetime"`  // 确认时间
}

// DocumentStrategyStep 策略步骤
type DocumentStrategyStep struct {
	common.Model
	PlanId          int64  `gorm:"column:plan_id;type:bigint"`             // 计划ID
	DocumentId      int64  `gorm:"column:document_id;type:bigint"`         // 文档ID
	StepNo          int    `gorm:"column:step_no;type:int"`                // 步骤序号
	PipelineType    string `gorm:"column:pipeline_type;type:varchar(255)"` // 管道类型
	StrategyType    int    `gorm:"column:strategy_type;type:int"`          // 策略类型
	StrategyRole    int    `gorm:"column:strategy_role;type:int"`          // 策略角色
	SourceType      int    `gorm:"column:source_type;type:int"`            // 来源类型
	ExecuteStatus   int    `gorm:"column:execute_status;type:int"`         // 执行状态
	RecommendReason string `gorm:"column:recommend_reason;type:text"`      // 推荐理由
}
