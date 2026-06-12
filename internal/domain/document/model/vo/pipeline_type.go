package vo

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
