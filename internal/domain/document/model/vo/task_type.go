package vo

// TaskType 任务类型
type TaskType = int

const (
	TaskTypeParseRoute TaskType = iota + 1
	TaskTypeBuildIndex
)

func TaskTypeName(tt TaskType) string {
	switch tt {
	case TaskTypeParseRoute:
		return "解析路由"
	case TaskTypeBuildIndex:
		return "构建索引"
	default:
		return ""
	}
}
