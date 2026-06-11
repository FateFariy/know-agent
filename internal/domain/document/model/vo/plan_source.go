package vo

// PlanSource 方案来源
type PlanSource = int

const (
	PlanSourceUnknown PlanSource = iota
	PlanSourceSystemRecommend
	PlanSourceUserAdjust
)
