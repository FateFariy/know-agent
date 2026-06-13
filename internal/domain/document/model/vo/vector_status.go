package vo

// VectorStatus 向量状态
type VectorStatus = int

const (
	VectorStatusUnknown VectorStatus = iota
	VectorStatusPending
	VectorStatusBuilding
	VectorStatusBuilt
	VectorStatusFailed
	VectorStatusBuildSuccess // 构建成功
)

func VectorStatusName(vs VectorStatus) string {
	switch vs {
	case VectorStatusPending:
		return "待构建"
	case VectorStatusBuilding:
		return "构建中"
	case VectorStatusBuilt:
		return "已构建"
	case VectorStatusFailed:
		return "构建失败"
	case VectorStatusBuildSuccess:
		return "构建成功"
	default:
		return "未知"
	}
}
