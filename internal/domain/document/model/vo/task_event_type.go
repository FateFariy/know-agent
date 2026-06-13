package vo

// TaskEventType 任务事件类型
type TaskEventType = int

const (
	TaskEventStart TaskEventType = iota + 1
	TaskEventComplete
	TaskEventFailed
	TaskEventUserConfirm
	TaskEventUserAdjust
	TaskEventRecommendStrategy
)

func TaskEventTypeName(et TaskEventType) string {
	switch et {
	case TaskEventStart:
		return "开始"
	case TaskEventComplete:
		return "完成"
	case TaskEventFailed:
		return "失败"
	case TaskEventUserConfirm:
		return "用户确认"
	case TaskEventUserAdjust:
		return "用户调整"
	case TaskEventRecommendStrategy:
		return "推荐策略"
	default:
		return ""
	}
}
