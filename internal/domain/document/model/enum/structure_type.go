package enum

type StructureNodeType = int

const (
	NodeTypeDocument StructureNodeType = iota + 1
	NodeTypeSection
	NodeTypeListItem
	NodeTypeStep
)
