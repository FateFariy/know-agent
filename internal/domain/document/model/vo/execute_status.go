package vo

// ExecuteStatus 执行状态
type ExecuteStatus = int

const (
	ExecuteStatusUnknown ExecuteStatus = iota
	ExecuteStatusPending
	ExecuteStatusRunning
	ExecuteStatusCompleted
	ExecuteStatusFailed
)
