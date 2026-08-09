package entity

import (
	"slices"
	"strings"
	"time"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/document/model/enum"
)

// DocumentStrategyPlan 策略方案实体
type DocumentStrategyPlan struct {
	ID               int64                     `gorm:"column:id;primaryKey"`     // 主键ID
	DocumentId       int64                     `gorm:"column:document_id"`       // 文档ID
	PlanVersion      int                       `gorm:"column:plan_version"`      // 方案版本
	PlanSource       int                       `gorm:"column:plan_source"`       // 方案来源
	PlanStatus       int                       `gorm:"column:plan_status"`       // 方案状态
	StrategyCount    int                       `gorm:"column:strategy_count"`    // 策略步骤数量
	StrategySnapshot string                    `gorm:"column:strategy_snapshot"` // 策略快照
	RecommendReason  string                    `gorm:"column:recommend_reason"`  // 推荐理由
	AdjustNote       string                    `gorm:"column:adjust_note"`       // 调整备注
	ConfirmUserId    int64                     `gorm:"column:confirm_user_id"`   // 确认人ID
	ConfirmTime      *time.Time                `gorm:"column:confirm_time"`      // 确认时间
	PlanSourceName   string                    `gorm:"-"`                        // 方案来源名称
	PlanStatusName   string                    `gorm:"-"`                        // 方案状态名称
	Normalized       bool                      `gorm:"-"`                        // 是否归一化
	ParentPipeline   *DocumentStrategyPipeline `gorm:"-"`                        // 父级流水线
	ChildPipeline    *DocumentStrategyPipeline `gorm:"-"`                        // 子级流水线
}

func (d *DocumentStrategyPlan) FillEnumNames() {
	if d == nil {
		return
	}
	d.PlanSourceName = enum.PlanSourceName(d.PlanSource)
	d.PlanStatusName = enum.ParseStatusName(d.PlanStatus)
}

func (d *DocumentStrategyPlan) FillAndProcessPipeline(stepList []*DocumentStrategyStep) {
	if d == nil {
		return
	}
	d.ParentPipeline = NewDocumentStrategyPipeline(enum.PipelineTypeParent, stepList)
	d.ChildPipeline = NewDocumentStrategyPipeline(enum.PipelineTypeChild, stepList)
	d.StrategySnapshot = "PARENT:" + d.ParentPipeline.StrategySnapshot + ";CHILD:" + d.ChildPipeline.StrategySnapshot
}

// DocumentStrategyStep 策略步骤实体
type DocumentStrategyStep struct {
	ID                int64  `gorm:"column:id;primaryKey"`    // 主键ID
	DocumentId        int64  `gorm:"column:document_id"`      // 文档ID
	PlanId            int64  `gorm:"column:plan_id"`          // 方案ID
	StepNo            int    `gorm:"column:step_no"`          // 步骤序号
	PipelineType      string `gorm:"column:pipeline_type"`    // 流水线类型
	StrategyType      int    `gorm:"column:strategy_type"`    // 策略类型
	StrategyRole      int    `gorm:"column:strategy_role"`    // 策略角色
	SourceType        int    `gorm:"column:source_type"`      // 来源类型
	ExecuteStatus     int    `gorm:"column:execute_status"`   // 执行状态
	RecommendReason   string `gorm:"column:recommend_reason"` // 推荐理由
	PipelineTypeName  string `gorm:"-"`                       // 流水线类型名称
	StrategyTypeName  string `gorm:"-"`                       // 策略类型名称
	StrategyRoleName  string `gorm:"-"`                       // 策略角色名称
	SourceTypeName    string `gorm:"-"`                       // 来源类型名称
	ExecuteStatusName string `gorm:"-"`                       // 执行状态名称
}

func (d *DocumentStrategyStep) FillEnumNames() {
	if d == nil {
		return
	}
	d.PipelineTypeName = enum.PipelineTypeName(d.PipelineType)
	d.StrategyTypeName = enum.StrategyTypeName(d.StrategyType)
	d.StrategyRoleName = enum.StrategyRoleName(d.StrategyRole)
	d.SourceTypeName = enum.StrategySourceTypeName(d.SourceType)
	d.ExecuteStatusName = enum.StrategyStatusName(d.ExecuteStatus)
}

type DocumentStrategyPipeline struct {
	PipelineType     string
	PipelineTypeName string
	StrategySnapshot string
	Steps            []*DocumentStrategyStep
}

func NewDocumentStrategyPipeline(pipelineType string, stepList []*DocumentStrategyStep) *DocumentStrategyPipeline {
	if stepList == nil {
		return nil
	}
	pipeline := &DocumentStrategyPipeline{PipelineType: pipelineType}
	pipeline.FillAndProcessSteps(stepList)
	return pipeline
}

func (d *DocumentStrategyPipeline) FillAndProcessSteps(stepList []*DocumentStrategyStep) {
	if d == nil {
		return
	}
	d.PipelineTypeName = enum.PipelineTypeName(d.PipelineType)
	steps := make([]*DocumentStrategyStep, 0, len(stepList))
	strategyTypes := make([]string, 0, len(stepList))
	for _, step := range stepList {
		if step == nil {
			continue
		}
		step.PipelineType = utils.BlankToDefault(step.PipelineType, enum.PipelineTypeChild)
		if step.PipelineType == d.PipelineType {
			step.FillEnumNames()
			steps = append(steps, step)
			strategyTypes = append(strategyTypes, step.StrategyTypeName)
		}
	}
	slices.SortFunc(steps, func(a, b *DocumentStrategyStep) int { return a.StepNo - b.StepNo })
	d.StrategySnapshot = strings.Join(strategyTypes, ",")
	d.Steps = steps
}

type DocumentStrategySteps []*DocumentStrategyStep

// SortPipelineSteps 过滤属于指定流水线的步骤并按 StepNo 升序排列
func (s DocumentStrategySteps) SortPipelineSteps(pipelineType string) DocumentStrategySteps {
	filtered := make(DocumentStrategySteps, 0, len(s))
	for _, step := range s {
		if step == nil {
			continue
		}
		toDefault := utils.BlankToDefault(step.PipelineType, enum.PipelineTypeChild)
		if utils.EqualsIgnoreCase(pipelineType, toDefault) {
			filtered = append(filtered, step)
		}
	}
	less := func(a, b *DocumentStrategyStep) int { return a.StepNo - b.StepNo }
	slices.SortFunc(filtered, less)
	return filtered
}

// DeleteStep 删除指定策略类型的步骤
func (s DocumentStrategySteps) DeleteStep(strategyType int) DocumentStrategySteps {
	if len(s) == 0 {
		return nil
	}
	filtered := make(DocumentStrategySteps, 0, len(s))
	for _, step := range s {
		if step.StrategyType != strategyType {
			filtered = append(filtered, step)
		}
	}
	return filtered
}

// Contains 检查步骤列表中是否包含指定策略类型的步骤
func (s DocumentStrategySteps) Contains(strategyType int) bool {
	if len(s) == 0 {
		return false
	}
	for _, step := range s {
		if step.StrategyType == strategyType {
			return true
		}
	}
	return false
}
