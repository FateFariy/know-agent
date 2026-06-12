package vo

// TaskStatus 任务状态
type TaskStatus = int

const (
	TaskStatusNew TaskStatus = iota + 1
	TaskStatusRunning
	TaskStatusCompleted
	TaskStatusFailed
)

func TaskStatusName(ts TaskStatus) string {
	switch ts {
	case TaskStatusNew:
		return "新建"
	case TaskStatusRunning:
		return "运行中"
	case TaskStatusCompleted:
		return "已完成"
	case TaskStatusFailed:
		return "失败"
	default:
		return ""
	}
}
