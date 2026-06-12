package vo

// ExecuteStatus 执行状态
type ExecuteStatus = int

const (
	ExecuteStatusPending ExecuteStatus = iota + 1 // 待执行
	ExecuteStatusRunning
	ExecuteStatusCompleted
	ExecuteStatusFailed
)

func ExecuteStatusName(status ExecuteStatus) string {
	switch status {
	case ExecuteStatusPending:
		return "待执行"
	case ExecuteStatusRunning:
		return "执行中"
	case ExecuteStatusCompleted:
		return "已完成"
	case ExecuteStatusFailed:
		return "失败"
	default:
		return ""
	}
}
