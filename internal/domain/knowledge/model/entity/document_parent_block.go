package model

type DocumentParentBlock struct {
	ID                int64  `gorm:"column:id"`                  // 主键ID
	DocumentId        int64  `gorm:"column:document_id"`         // 文档ID
	TaskId            int64  `gorm:"column:task_id"`             // 任务ID
	PlanId            int64  `gorm:"column:plan_id"`             // 计划ID
	ParentNo          int    `gorm:"column:parent_no"`           // 父节点编号
	SourceType        int    `gorm:"column:source_type"`         // 来源类型
	SectionPath       string `gorm:"column:section_path"`        // 章节路径
	StructureNodeId   int64  `gorm:"column:structure_node_id"`   // 结构节点ID
	StructureNodeType int    `gorm:"column:structure_node_type"` // 结构节点类型
	CanonicalPath     string `gorm:"column:canonical_path"`      // 规范路径
	ItemIndex         int    `gorm:"column:item_index"`          // 项目索引
	ParentText        string `gorm:"column:parent_text"`         // 父节点文本
	CharCount         int    `gorm:"column:char_count"`          // 字符数
	TokenCount        int    `gorm:"column:token_count"`         // 令牌数
	ChildCount        int    `gorm:"column:child_count"`         // 子节点数
	StartChunkNo      int    `gorm:"column:start_chunk_no"`      // 起始块号
	EndChunkNo        int    `gorm:"column:end_chunk_no"`        // 结束块号
}
