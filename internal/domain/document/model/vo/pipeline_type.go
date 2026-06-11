package vo

// PipelineType 流水线类型
type PipelineType = int

const (
	PipelineTypeUnknown PipelineType = iota
	PipelineTypeParent
	PipelineTypeChild
)
