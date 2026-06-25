package vo

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
