package vo

// PlanStatus 方案状态
type PlanStatus = int

const (
	PlanStatusUnknown PlanStatus = iota
	PlanStatusRecommended
	PlanStatusConfirmed
	PlanStatusDiscarded
)

func PlanStatusName(status PlanStatus) string {
	switch status {
	case PlanStatusRecommended:
		return "已推荐"
	case PlanStatusConfirmed:
		return "已确认"
	case PlanStatusDiscarded:
		return "已废弃"
	default:
		return "未知"
	}
}
