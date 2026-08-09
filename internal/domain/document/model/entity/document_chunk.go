package entity

import (
	"github.com/swiftbit/know-agent/internal/domain/document/model/enum"
)

type DocumentChunk struct {
	ID                 int64  `gorm:"column:id"`                  // ID
	DocumentId         int64  `gorm:"column:document_id"`         // 文档ID
	TaskId             int64  `gorm:"column:task_id"`             // 任务ID
	PlanId             int64  `gorm:"column:plan_id"`             // 计划ID
	ParentChunkId      int64  `gorm:"column:parent_block_id"`     // 父块ID
	ChunkNo            int    `gorm:"column:chunk_no"`            // 块序号
	SourceType         int    `gorm:"column:source_type"`         // 来源类型
	SectionPath        string `gorm:"column:section_path"`        // 章节路径
	StructureNodeId    int64  `gorm:"column:structure_node_id"`   // 结构节点ID
	StructureNodeType  int    `gorm:"column:structure_node_type"` // 结构节点类型
	CanonicalPath      string `gorm:"column:canonical_path"`      // 规范路径
	ItemIndex          int    `gorm:"column:item_index"`          // 项索引
	ChunkText          string `gorm:"column:chunk_text"`          // 块文本
	CharCount          int    `gorm:"column:char_count"`          // 字符数
	TokenCount         int    `gorm:"column:token_count"`         // token数
	VectorStatus       int    `gorm:"column:vector_status"`       // 向量状态
	VectorStoreType    int    `gorm:"column:vector_store_type"`   // 向量存储类型
	VectorId           string `gorm:"column:vector_id"`           // 向量ID
	ContentWithWeight  string `gorm:"column:content_with_weight"` // 加权内容
	ChunkType          string `gorm:"column:chunk_type"`          // 块类型
	Title              string `gorm:"column:title"`               // 标题
	Keywords           string `gorm:"column:keywords"`            // 关键词
	Questions          string `gorm:"column:questions"`           // 预设问题
	PageNo             int    `gorm:"column:page_no"`             // 页码
	PageRange          string `gorm:"column:page_range"`          // 页码范围
	BboxJson           string `gorm:"column:bbox_json"`           // 边界框JSON
	SourceBlockIds     string `gorm:"column:source_block_ids"`    // 源块ID列表
	ParentBlockNo      int    `gorm:"-"`                          // 父块序号
	ParentChildCount   int    `gorm:"-"`                          // 父子节点数
	ParentStartChunkNo int    `gorm:"-"`                          // 父起始块号
	ParentEndChunkNo   int    `gorm:"-"`                          // 父结束块号
	SourceTypeName     string `gorm:"-"`                          // 来源类型名称
	VectorStatusName   string `gorm:"-"`                          // 向量状态名称

}

func (d *DocumentChunk) FillEnumName() {
	if d == nil {
		return
	}
	d.VectorStatusName = enum.VectorStatusName(d.VectorStatus)
	d.SourceTypeName = enum.DocumentChunkSourceTypeName(d.SourceType)
}

func (d *DocumentChunk) FillParentInfo(parentBlock *DocumentParentChunk) {
	if d != nil && parentBlock == nil {
		d.ParentBlockNo = parentBlock.ParentNo
		d.ParentChildCount = parentBlock.ChildCount
		d.ParentStartChunkNo = parentBlock.StartChunkNo
		d.ParentEndChunkNo = parentBlock.EndChunkNo
	}
}
