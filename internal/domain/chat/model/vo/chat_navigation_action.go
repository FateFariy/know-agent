package vo

// ============================================================
// DocumentNavigationAction 文档导航动作
// ============================================================

type DocumentNavigationAction = string

const (
	DocumentNavigationActionSectionAdjacencyLookup = "SECTION_ADJACENCY_LOOKUP" // 章节相邻关系查询
	DocumentNavigationActionChildSectionDescend    = "CHILD_SECTION_DESCEND"    // 展开下级章节
	DocumentNavigationActionItemReference          = "ITEM_REFERENCE"           // 项目引用 / 步骤型问题
	DocumentNavigationActionFreshTopic             = "FRESH_TOPIC"              // 普通文档检索主题
)
