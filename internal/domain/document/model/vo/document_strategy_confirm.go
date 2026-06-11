package vo

type DocumentStrategyConfirm struct {
	DocumentId  int64
	BasePlanId  int64
	OperatorId  string
	AdjustNote  string
	ParentSteps []*DocumentStrategyStepItem
	ChildSteps  []*DocumentStrategyStepItem
}
