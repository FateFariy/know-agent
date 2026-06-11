package vo

// PlanStatus 方案状态
type PlanStatus = int

const (
	PlanStatusUnknown PlanStatus = iota
	PlanStatusRecommended
	PlanStatusConfirmed
	PlanStatusDiscarded
)
