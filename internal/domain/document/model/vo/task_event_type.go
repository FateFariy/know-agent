package vo

// TaskEventType 任务事件类型
type TaskEventType = int

const (
	TaskEventUnknown TaskEventType = iota
	TaskEventStart
	TaskEventComplete
	TaskEventFailed
	TaskEventUserConfirm
	TaskEventUserAdjust
)
