package model

import "github.com/swiftbit/know-agent/common"

// DocumentStructureNode 文档结构节点实体
type DocumentStructureNode struct {
	common.Model
	DocumentId        int64  `gorm:"column:document_id"`          // 文档ID
	ParseTaskId       int64  `gorm:"column:parse_task_id"`        // 解析任务ID
	NodeNo            int    `gorm:"column:node_no"`              // 节点序号
	NodeType          int    `gorm:"column:node_type"`            // 节点类型
	ParentNodeId      int64  `gorm:"column:parent_node_id"`       // 父节点ID
	PrevSiblingNodeId int64  `gorm:"column:prev_sibling_node_id"` // 前一个兄弟节点ID
	NextSiblingNodeId int64  `gorm:"column:next_sibling_node_id"` // 后一个兄弟节点ID
	Depth             int    `gorm:"column:depth"`                // 深度
	NodeCode          string `gorm:"column:node_code"`            // 节点编码
	Title             string `gorm:"column:title"`                // 标题
	AnchorText        string `gorm:"column:anchor_text"`          // 锚文本
	CanonicalPath     string `gorm:"column:canonical_path"`       // 规范路径
	SectionPath       string `gorm:"column:section_path"`         // 章节路径
	ContentText       string `gorm:"column:content_text"`         // 内容文本
	ItemIndex         int    `gorm:"column:item_index"`           // 条目索引
}
