package model

import "github.com/swiftbit/know-agent/common"

// KnowledgeScopeNode 知识范围节点实体
type KnowledgeScopeNode struct {
	common.Model
	KnowledgeBaseId int64  `gorm:"column:knowledge_base_id;type:bigint"` // 知识库ID
	ScopeName       string `gorm:"column:scope_name;type:varchar(255)"`  // 范围名称
	ParentScopeId   int64  `gorm:"column:parent_scope_id;type:bigint"`   // 父范围ID
	Description     string `gorm:"column:description;type:text"`         // 描述
	Aliases         string `gorm:"column:aliases;type:text"`             // 别名
	Examples        string `gorm:"column:examples;type:text"`            // 示例
	SortOrder       int    `gorm:"column:sort_order;type:int"`           // 排序序号
}
