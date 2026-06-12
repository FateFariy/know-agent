package entity

import "github.com/swiftbit/know-agent/common"

// DocumentChunk 文档块实体
type DocumentChunk struct {
	common.Model
	DocumentId        int64  `gorm:"column:document_id"`         // 文档ID
	TaskId            int64  `gorm:"column:task_id"`             // 任务ID
	PlanId            int64  `gorm:"column:plan_id"`             // 方案ID
	ParentBlockId     int64  `gorm:"column:parent_block_id"`     // 父块ID
	ChunkNo           int    `gorm:"column:chunk_no"`            // 块序号
	SourceType        int    `gorm:"column:source_type"`         // 来源类型
	SectionPath       string `gorm:"column:section_path"`        // 章节路径
	StructureNodeId   int64  `gorm:"column:structure_node_id"`   // 结构节点ID
	StructureNodeType int    `gorm:"column:structure_node_type"` // 结构节点类型
	CanonicalPath     string `gorm:"column:canonical_path"`      // 规范路径
	ItemIndex         int    `gorm:"column:item_index"`          // 条目索引
	ChunkText         string `gorm:"column:chunk_text"`          // 块文本内容
	CharCount         int    `gorm:"column:char_count"`          // 字符数
	TokenCount        int    `gorm:"column:token_count"`         // Token数量
	VectorStatus      int    `gorm:"column:vector_status"`       // 向量状态
	VectorStoreType   int    `gorm:"column:vector_store_type"`   // 向量存储类型
	VectorId          string `gorm:"column:vector_id"`           // 向量ID
}

// DocumentParentBlock 文档父块实体
type DocumentParentBlock struct {
	common.Model
	DocumentId        int64  `gorm:"column:document_id"`         // 文档ID
	TaskId            int64  `gorm:"column:task_id"`             // 任务ID
	PlanId            int64  `gorm:"column:plan_id"`             // 方案ID
	ParentNo          int    `gorm:"column:parent_no"`           // 父块序号
	SourceType        int    `gorm:"column:source_type"`         // 来源类型
	SectionPath       string `gorm:"column:section_path"`        // 章节路径
	StructureNodeId   int64  `gorm:"column:structure_node_id"`   // 结构节点ID
	StructureNodeType int    `gorm:"column:structure_node_type"` // 结构节点类型
	CanonicalPath     string `gorm:"column:canonical_path"`      // 规范路径
	ItemIndex         int    `gorm:"column:item_index"`          // 条目索引
	ParentText        string `gorm:"column:parent_text"`         // 父块文本内容
	CharCount         int    `gorm:"column:char_count"`          // 字符数
	TokenCount        int    `gorm:"column:token_count"`         // Token数量
	ChildCount        int    `gorm:"column:child_count"`         // 子块数量
	StartChunkNo      int    `gorm:"column:start_chunk_no"`      // 起始块序号
	EndChunkNo        int    `gorm:"column:end_chunk_no"`        // 结束块序号
}

// DocumentProfile 文档属性实体
type DocumentProfile struct {
	common.Model
	DocumentId           int64  `gorm:"column:document_id"`            // 文档ID
	ProfileVersion       int    `gorm:"column:profile_version"`        // 属性版本
	DocumentSummary      string `gorm:"column:document_summary"`       // 文档摘要
	DocumentType         string `gorm:"column:document_type"`          // 文档类型
	CoreTopics           string `gorm:"column:core_topics"`            // 核心话题
	ExampleQuestions     string `gorm:"column:example_questions"`      // 示例问题
	GraphFriendly        int    `gorm:"column:graph_friendly"`         // 图谱友好度
	SupportsGraphOutline int    `gorm:"column:supports_graph_outline"` // 支持图谱大纲
	SupportsItemLookup   int    `gorm:"column:supports_item_lookup"`   // 支持条目检索
	SupportsGraphAssist  int    `gorm:"column:supports_graph_assist"`  // 支持图谱辅助
	ProfileSource        string `gorm:"column:profile_source"`         // 属性来源
	ProfileStatus        int    `gorm:"column:profile_status"`         // 属性状态
	ErrorMsg             string `gorm:"column:error_msg"`              // 错误信息
}

// TopicDocumentRelation 话题文档关联实体
type TopicDocumentRelation struct {
	common.Model
	TopicCode      string `gorm:"column:topic_code"`      // 话题编码
	DocumentId     int64  `gorm:"column:document_id"`     // 文档ID
	RelationScore  string `gorm:"column:relation_score"`  // 关联分数
	RelationSource string `gorm:"column:relation_source"` // 关联来源
	Reason         string `gorm:"column:reason"`          // 关联原因
}
