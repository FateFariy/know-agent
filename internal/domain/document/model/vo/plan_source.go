package vo

// PlanSource 方案来源
type PlanSource = int

const (
	PlanSourceUnknown PlanSource = iota
	PlanSourceSystemRecommend
	PlanSourceUserAdjust
)

func PlanSourceName(source PlanSource) string {
	switch source {
	case PlanSourceSystemRecommend:
		return "系统推荐"
	case PlanSourceUserAdjust:
		return "用户调整"
	default:
		return "未知"
	}
}
