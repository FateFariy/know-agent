package vo

import (
	"slices"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/document/model/enum"
)

type DocumentStrategyConfirmCmd struct {
	DocumentId  int64                       // 文档ID
	BasePlanId  int64                       // 基础方案ID
	OperatorId  int64                       // 操作员ID
	AdjustNote  string                      // 调整说明
	ParentSteps []*DocumentStrategyStepItem // 父步骤
	ChildSteps  []*DocumentStrategyStepItem // 子步骤
}

type DocumentStrategyStepItem struct {
	StepNo       int // 步骤编号
	StrategyType int // 策略类型
}

// NormalizeSteps 归一化步骤类型
func (c *DocumentStrategyConfirmCmd) NormalizeSteps() bool {
	parentSteps := utils.DistinctFilterLimit(c.ParentSteps, -1, func(step *DocumentStrategyStepItem) (int, bool) {
		return step.StrategyType, enum.StrategyTypeName(step.StrategyType) != ""
	})
	childSteps := utils.DistinctFilterLimit(c.ChildSteps, -1, func(step *DocumentStrategyStepItem) (int, bool) {
		return step.StrategyType, enum.StrategyTypeName(step.StrategyType) != ""
	})
	normalize := len(parentSteps) != len(c.ParentSteps) || len(childSteps) != len(c.ChildSteps)
	c.ParentSteps = parentSteps
	c.ChildSteps = childSteps
	return normalize
}

// GetSortedParentStrategyTypes 返回父步骤排序后的策略类型列表
func (c *DocumentStrategyConfirmCmd) GetSortedParentStrategyTypes() []int {
	return extractSortedStrategyTypes(c.ParentSteps)
}

// GetSortedChildStrategyTypes 返回子步骤排序后的策略类型列表
func (c *DocumentStrategyConfirmCmd) GetSortedChildStrategyTypes() []int {
	return extractSortedStrategyTypes(c.ChildSteps)
}

// extractSortedStrategyTypes 提取步骤列表并按 StepNo 升序排序后，返回策略类型切片
func extractSortedStrategyTypes(steps []*DocumentStrategyStepItem) []int {
	slices.SortFunc(steps, func(a, b *DocumentStrategyStepItem) int {
		return a.StepNo - b.StepNo
	})
	strategyTypes := make([]int, 0, len(steps))
	for _, step := range steps {
		strategyTypes = append(strategyTypes, step.StrategyType)
	}
	return strategyTypes
}
