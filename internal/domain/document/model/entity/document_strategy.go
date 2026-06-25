package entity

import (
	"slices"
	"strings"
	"time"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/document/model/vo"
)

// DocumentStrategyPlan 策略方案实体
type DocumentStrategyPlan struct {
	ID               int64                     `gorm:"column:id;primaryKey"`     // 主键ID
	DocumentId       int64                     `gorm:"column:document_id"`       // 文档ID
	PlanVersion      int                       `gorm:"column:plan_version"`      // 方案版本
	PlanSource       int                       `gorm:"column:plan_source"`       // 方案来源
	PlanStatus       int                       `gorm:"column:plan_status"`       // 方案状态
	StrategyCount    int                       `gorm:"column:strategy_count"`    // 策略数量
	StrategySnapshot string                    `gorm:"column:strategy_snapshot"` // 策略快照
	RecommendReason  string                    `gorm:"column:recommend_reason"`  // 推荐理由
	AdjustNote       string                    `gorm:"column:adjust_note"`       // 调整备注
	ConfirmUserId    int64                     `gorm:"column:confirm_user_id"`   // 确认人ID
	ConfirmTime      time.Time                 `gorm:"column:confirm_time"`      // 确认时间
	PlanSourceName   string                    `gorm:"-"`                        // 方案来源名称
	PlanStatusName   string                    `gorm:"-"`                        // 方案状态名称
	Normalized       bool                      `gorm:"-"`                        // 是否归一化
	ParentPipeline   *DocumentStrategyPipeline `gorm:"-"`                        // 父级流水线
	ChildPipeline    *DocumentStrategyPipeline `gorm:"-"`                        // 子级流水线
}

func (d *DocumentStrategyPlan) FillEnumNames() {
	d.PlanSourceName = vo.PlanSourceName(d.PlanSource)
	d.PlanStatusName = vo.ParseStatusName(d.PlanStatus)
}

func (d *DocumentStrategyPlan) FillAndProcessPipeline(stepList []*DocumentStrategyStep) {
	d.ParentPipeline = NewDocumentStrategyPipeline(vo.PipelineTypeParent, stepList)
	d.ChildPipeline = NewDocumentStrategyPipeline(vo.PipelineTypeChild, stepList)
}

// DocumentStrategyStep 策略步骤实体
type DocumentStrategyStep struct {
	ID                int64  `gorm:"column:id;primaryKey"`    // 主键ID
	DocumentId        int64  `gorm:"column:document_id"`      // 文档ID
	PlanId            int64  `gorm:"column:plan_id"`          // 方案ID
	StepNo            int    `gorm:"column:step_no"`          // 步骤序号
	PipelineType      int    `gorm:"column:pipeline_type"`    // 流水线类型
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
	d.PipelineTypeName = vo.PipelineTypeName(d.PipelineType)
	d.StrategyTypeName = vo.StrategyTypeName(d.StrategyType)
	d.StrategyRoleName = vo.StrategyRoleName(d.StrategyRole)
	d.SourceTypeName = vo.StrategySourceTypeName(d.SourceType)
	d.ExecuteStatusName = vo.ExecuteStatusName(d.ExecuteStatus)
}

type DocumentStrategyPipeline struct {
	PipelineType     int
	PipelineTypeName string
	StrategySnapshot string
	Steps            []*DocumentStrategyStep
}

func NewDocumentStrategyPipeline(pipelineType int, stepList []*DocumentStrategyStep) *DocumentStrategyPipeline {
	pipeline := &DocumentStrategyPipeline{PipelineType: pipelineType}
	pipeline.FillAndProcessSteps(stepList)
	return pipeline
}

func (d *DocumentStrategyPipeline) FillAndProcessSteps(stepList []*DocumentStrategyStep) {
	d.PipelineTypeName = vo.PipelineTypeName(d.PipelineType)
	steps := make([]*DocumentStrategyStep, 0, len(stepList))
	strategyTypes := make([]string, 0, len(stepList))
	for i := range stepList {
		stepList[i].PipelineType = utils.Ternary(stepList[i].PipelineType == 0, vo.PipelineTypeChild, stepList[i].PipelineType)
		if stepList[i].PipelineType == d.PipelineType {
			stepList[i].FillEnumNames()
			steps = append(steps, stepList[i])
			strategyTypes = append(strategyTypes, stepList[i].StrategyTypeName)
		}
	}
	slices.SortFunc(steps, func(a, b *DocumentStrategyStep) int { return a.StepNo - b.StepNo })
	d.StrategySnapshot = strings.Join(strategyTypes, ",")
	d.Steps = steps
}
