package model

import "github.com/swiftbit/know-agent/common"

// KnowledgeScopeNode 知识范围节点实体
type KnowledgeScopeNode struct {
	common.Model
	ScopeCode       string `gorm:"column:scope_code"`        // 范围编码
	ScopeName       string `gorm:"column:scope_name"`        // 范围名称
	ParentScopeCode string `gorm:"column:parent_scope_code"` // 父范围编码
	Description     string `gorm:"column:description"`       // 描述
	Aliases         string `gorm:"column:aliases"`           // 别名
	Examples        string `gorm:"column:examples"`          // 示例
	SortOrder       int    `gorm:"column:sort_order"`        // 排序顺序
}
