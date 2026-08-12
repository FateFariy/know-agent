package enum

// StructureNavigationOperation 结构导航操作枚举
type StructureNavigationOperation = string

const (
	CurrentSection      StructureNavigationOperation = "CURRENT_SECTION"
	ParentSection       StructureNavigationOperation = "PARENT_SECTION"
	PreviousSibling     StructureNavigationOperation = "PREVIOUS_SIBLING"
	NextSibling         StructureNavigationOperation = "NEXT_SIBLING"
	DirectChildren      StructureNavigationOperation = "DIRECT_CHILDREN"
	SectionWithSiblings StructureNavigationOperation = "SECTION_WITH_SIBLINGS"
	SectionWithChildren StructureNavigationOperation = "SECTION_WITH_CHILDREN"
)
