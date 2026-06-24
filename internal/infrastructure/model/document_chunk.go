package model

import "github.com/swiftbit/know-agent/common"

type DocumentChunk struct {
	common.Model
	documentId        int64  `gorm:"column:document_id"`         // 文档ID
	taskId            int64  `gorm:"column:task_id"`             // 任务ID
	planId            int64  `gorm:"column:plan_id"`             // 计划ID
	parentBlockId     int64  `gorm:"column:parent_block_id"`     // 父块ID
	chunkNo           int64  `gorm:"column:chunk_no"`            // 块序号
	sourceType        int64  `gorm:"column:source_type"`         // 来源类型
	sectionPath       string `gorm:"column:section_path"`        // 章节路径
	structureNodeId   int64  `gorm:"column:structure_node_id"`   // 结构节点ID
	structureNodeType int64  `gorm:"column:structure_node_type"` // 结构节点类型
	canonicalPath     string `gorm:"column:canonical_path"`      // 规范路径
	itemIndex         int64  `gorm:"column:item_index"`          // 项索引
	chunkText         string `gorm:"column:chunk_text"`          // 块文本
	charCount         int64  `gorm:"column:char_count"`          // 字符数
	tokenCount        int64  `gorm:"column:token_count"`         // token数
	vectorStatus      int64  `gorm:"column:vector_status"`       // 向量状态
	vectorStoreType   int64  `gorm:"column:vector_store_type"`   // 向量存储类型
	vectorId          string `gorm:"column:vector_id"`           // 向量ID
}
