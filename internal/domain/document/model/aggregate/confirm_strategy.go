package aggregate

import "github.com/swiftbit/know-agent/internal/domain/document/model/entity"

// ConfirmStrategy 确认策略聚合根
// 封装策略确认过程中的所有修改操作，由基础设施层统一事务性保存
type ConfirmStrategy struct {
	Document        *entity.Document               // 文档实体（更新CurrentPlanId和StrategyStatus）
	Task            *entity.DocumentTask           // 任务实体（更新CurrentStage）
	OldStrategyPlan *entity.DocumentStrategyPlan   // 原策略方案（可能被废弃）
	NewStrategyPlan *entity.DocumentStrategyPlan   // 新策略方案（用户调整后创建）
	Steps           []*entity.DocumentStrategyStep // 策略步骤列表（新方案的步骤）
	AdjustLog       *entity.DocumentTaskLog        // 调整日志（用户调整策略时记录）
	ConfirmLog      *entity.DocumentTaskLog        // 确认日志（用户确认策略时记录）
	IsChanged       bool                           // 是否发生了策略变更
}
