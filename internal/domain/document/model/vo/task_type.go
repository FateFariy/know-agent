package vo

// TaskType 任务类型
type TaskType = int

const (
	TaskTypeUnknown TaskType = iota
	TaskTypeParseRoute
	TaskTypeBuildIndex
)
