package vo

// TaskStatus 任务状态
type TaskStatus = int

const (
	TaskStatusUnknown TaskStatus = iota
	TaskStatusNew
	TaskStatusRunning
	TaskStatusCompleted
	TaskStatusFailed
)
