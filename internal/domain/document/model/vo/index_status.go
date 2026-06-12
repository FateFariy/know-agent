package vo

// IndexStatus 索引状态
type IndexStatus = int

const (
	IndexStatusWaitBuild    IndexStatus = iota + 1 // 待构建
	IndexStatusBuilding                            // 构建中
	IndexStatusBuildSuccess                        // 构建成功
	IndexStatusBuildFailed                         // 构建失败
)

func IndexStatusName(status IndexStatus) string {
	switch status {
	case IndexStatusWaitBuild:
		return "待构建"
	case IndexStatusBuilding:
		return "构建中"
	case IndexStatusBuildSuccess:
		return "构建成功"
	case IndexStatusBuildFailed:
		return "构建失败"
	default:
		return ""
	}
}
