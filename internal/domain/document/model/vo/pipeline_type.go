package vo

import "strings"

// PipelineType 流水线类型
type PipelineType = int

const (
	PipelineTypeParent PipelineType = iota + 1 // 父块
	PipelineTypeChild                          // 子块
)

func PipelineTypeName(pt PipelineType) string {
	switch pt {
	case PipelineTypeParent:
		return "父块"
	case PipelineTypeChild:
		return "子块"
	default:
		return ""
	}
}

// PipelineTypeNameStr 根据字符串类型获取流水线类型名称
func PipelineTypeNameStr(pt string) string {
	switch strings.ToUpper(pt) {
	case "PARENT":
		return "父块"
	case "CHILD":
		return "子块"
	default:
		return "子块"
	}
}
