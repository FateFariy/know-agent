package entity

// KnowledgeScopeNode 知识范围节点
type KnowledgeScopeNode struct {
	ID              int64  `gorm:"column:id"`
	KnowledgeBaseId int64  `gorm:"column:knowledge_base_id"` // 知识库ID
	ScopeName       string `gorm:"column:scope_name"`        // 范围名称
	ParentScopeId   int64  `gorm:"column:parent_scope_id"`   // 父范围ID
	Description     string `gorm:"column:description"`       // 描述
	Aliases         string `gorm:"column:aliases"`           // 别名
	Examples        string `gorm:"column:examples"`          // 示例
	SortOrder       int    `gorm:"column:sort_order"`        // 排序序号
	OperatorId      string `gorm:"-"`                        // 操作员ID
}
