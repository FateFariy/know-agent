package model

import "github.com/swiftbit/know-agent/common"

// DocumentStructureNode 文档结构节点实体
type DocumentStructureNode struct {
	common.Model
	DocumentId          int64  `gorm:"column:document_id;type:bigint"`                // 文档ID
	ParseTaskId         int64  `gorm:"column:parse_task_id;type:bigint"`              // 解析任务ID
	NodeNo              int    `gorm:"column:node_no;type:int"`                       // 节点编号
	NodeType            int    `gorm:"column:node_type;type:int"`                     // 节点类型
	ParentNodeId        int64  `gorm:"column:parent_node_id;type:bigint"`             // 父节点ID
	PrevSiblingNodeId   int64  `gorm:"column:prev_sibling_node_id;type:bigint"`       // 前兄弟节点ID
	NextSiblingNodeId   int64  `gorm:"column:next_sibling_node_id;type:bigint"`       // 后兄弟节点ID
	Depth               int    `gorm:"column:depth;type:int"`                         // 深度
	NodeCode            string `gorm:"column:node_code;type:varchar(255)"`            // 节点编码
	Title               string `gorm:"column:title;type:varchar(500)"`                // 标题
	AnchorText          string `gorm:"column:anchor_text;type:text"`                  // 锚点文本
	CanonicalPath       string `gorm:"column:canonical_path;type:text"`               // 规范路径
	SectionPath         string `gorm:"column:section_path;type:text"`                 // 章节路径
	ContentText         string `gorm:"column:content_text;type:text"`                 // 内容文本
	ItemIndex           int    `gorm:"column:item_index;type:int"`                    // 条目索引
	SyntaxSchemaVersion string `gorm:"column:syntax_schema_version;type:varchar(50)"` // 语法结构版本
	SyntaxSourceSha256  string `gorm:"column:syntax_source_sha256;type:varchar(64)"`  // 语法源SHA256
	SyntaxNodeId        string `gorm:"column:syntax_node_id;type:varchar(255)"`       // 语法节点ID
	SyntaxNodeType      string `gorm:"column:syntax_node_type;type:varchar(255)"`     // 语法节点类型
	SyntaxSourceOrigin  string `gorm:"column:syntax_source_origin;type:varchar(255)"` // 语法源来源
	SourceStartByte     int    `gorm:"column:source_start_byte;type:int"`             // 源起始字节
	SourceEndByte       int    `gorm:"column:source_end_byte;type:int"`               // 源结束字节
	SourceStartLine     int    `gorm:"column:source_start_line;type:int"`             // 源起始行
	SourceStartColumn   int    `gorm:"column:source_start_column;type:int"`           // 源起始列
	SourceEndLine       int    `gorm:"column:source_end_line;type:int"`               // 源结束行
	SourceEndColumn     int    `gorm:"column:source_end_column;type:int"`             // 源结束列
}
