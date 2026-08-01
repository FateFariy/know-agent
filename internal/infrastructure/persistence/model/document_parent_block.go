package model

import "github.com/swiftbit/know-agent/common"

type DocumentParentBlock struct {
	common.Model
	DocumentId        int64  `gorm:"column:document_id;type:bigint"`          // 文档ID
	TaskId            int64  `gorm:"column:task_id;type:bigint"`              // 任务ID
	PlanId            int64  `gorm:"column:plan_id;type:bigint"`              // 计划ID
	ParentNo          int    `gorm:"column:parent_no;type:int"`               // 父节点编号
	SourceType        int    `gorm:"column:source_type;type:int"`             // 来源类型
	SectionPath       string `gorm:"column:section_path;type:varchar(255)"`   // 章节路径
	StructureNodeId   int64  `gorm:"column:structure_node_id;type:bigint"`    // 结构节点ID
	StructureNodeType int    `gorm:"column:structure_node_type;type:int"`     // 结构节点类型
	CanonicalPath     string `gorm:"column:canonical_path;type:varchar(255)"` // 规范路径
	ItemIndex         int    `gorm:"column:item_index;type:int"`              // 项目索引
	ParentText        string `gorm:"column:parent_text;type:varchar(255)"`    // 父节点文本
	CharCount         int    `gorm:"column:char_count;type:int"`              // 字符数
	TokenCount        int    `gorm:"column:token_count;type:int"`             // 令牌数
	ChildCount        int    `gorm:"column:child_count;type:int"`             // 子节点数
	StartChunkNo      int    `gorm:"column:start_chunk_no;type:int"`          // 起始块号
	EndChunkNo        int    `gorm:"column:end_chunk_no;type:int"`            // 结束块号
}
