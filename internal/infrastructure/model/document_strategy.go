package model

import (
	"time"

	"github.com/swiftbit/know-agent/common"
)

// DocumentStrategyPlan 策略方案实体
type DocumentStrategyPlan struct {
	common.Model
	DocumentId       int64     `gorm:"column:document_id"`       // 文档ID
	PlanVersion      int       `gorm:"column:plan_version"`      // 方案版本
	PlanSource       int       `gorm:"column:plan_source"`       // 方案来源
	PlanStatus       int       `gorm:"column:plan_status"`       // 方案状态
	StrategyCount    int       `gorm:"column:strategy_count"`    // 策略数量
	StrategySnapshot string    `gorm:"column:strategy_snapshot"` // 策略快照
	RecommendReason  string    `gorm:"column:recommend_reason"`  // 推荐理由
	AdjustNote       string    `gorm:"column:adjust_note"`       // 调整备注
	ConfirmUserId    int64     `gorm:"column:confirm_user_id"`   // 确认人ID
	ConfirmTime      time.Time `gorm:"column:confirm_time"`      // 确认时间
}

// DocumentStrategyStep 策略步骤实体
type DocumentStrategyStep struct {
	common.Model
	DocumentId      int64  `gorm:"column:document_id"`      // 文档ID
	PlanId          int64  `gorm:"column:plan_id"`          // 方案ID
	StepNo          int    `gorm:"column:step_no"`          // 步骤序号
	PipelineType    string `gorm:"column:pipeline_type"`    // 流水线类型
	StrategyType    int    `gorm:"column:strategy_type"`    // 策略类型
	StrategyRole    int    `gorm:"column:strategy_role"`    // 策略角色
	SourceType      int    `gorm:"column:source_type"`      // 来源类型
	ExecuteStatus   int    `gorm:"column:execute_status"`   // 执行状态
	RecommendReason string `gorm:"column:recommend_reason"` // 推荐理由
}
