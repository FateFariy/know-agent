package vo

// PipelineType 流水线类型
type PipelineType = string

const (
	PipelineTypeParent PipelineType = "PARENT" // 父块
	PipelineTypeChild               = "CHILD"  // 子块
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
