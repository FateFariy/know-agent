package model

import "github.com/swiftbit/know-agent/common"

type DocumentChunk struct {
	common.Model
	DocumentId        int64  `gorm:"column:document_id;type:bigint"`       // 文档ID
	TaskId            int64  `gorm:"column:task_id;type:bigint"`           // 任务ID
	PlanId            int64  `gorm:"column:plan_id;type:bigint"`           // 计划ID
	ParentBlockId     int64  `gorm:"column:parent_block_id;type:bigint"`   // 父块ID
	ChunkNo           int    `gorm:"column:chunk_no;type:int"`             // 块序号
	SourceType        int    `gorm:"column:source_type;type:int"`          // 来源类型
	SectionPath       string `gorm:"column:section_path;type:text"`        // 章节路径
	StructureNodeId   int64  `gorm:"column:structure_node_id;type:bigint"` // 结构节点ID
	StructureNodeType int    `gorm:"column:structure_node_type;type:int"`  // 结构节点类型
	CanonicalPath     string `gorm:"column:canonical_path;type:text"`      // 规范路径
	ItemIndex         int    `gorm:"column:item_index;type:int"`           // 项索引
	ChunkText         string `gorm:"column:chunk_text;type:text"`          // 块文本
	CharCount         int    `gorm:"column:char_count;type:int"`           // 字符数
	TokenCount        int    `gorm:"column:token_count;type:int"`          // Token数
	VectorStatus      int    `gorm:"column:vector_status;type:int"`        // 向量状态
	VectorStoreType   int    `gorm:"column:vector_store_type;type:int"`    // 向量存储类型
	VectorId          string `gorm:"column:vector_id;type:varchar(255)"`   // 向量ID
	ContentWithWeight string `gorm:"column:content_with_weight;type:text"` // 加权内容
	ChunkType         string `gorm:"column:chunk_type;type:varchar(50)"`   // 块类型
	Title             string `gorm:"column:title;type:varchar(500)"`       // 标题
	Keywords          string `gorm:"column:keywords;type:text"`            // 关键词
	Questions         string `gorm:"column:questions;type:text"`           // 预设问题
	PageNo            int    `gorm:"column:page_no;type:int"`              // 页码
	PageRange         string `gorm:"column:page_range;type:varchar(50)"`   // 页码范围
	BboxJson          string `gorm:"column:bbox_json;type:text"`           // 边界框JSON
	SourceBlockIds    string `gorm:"column:source_block_ids;type:text"`    // 源块ID列表
}
