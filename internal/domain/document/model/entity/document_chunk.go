package entity

import "github.com/swiftbit/know-agent/internal/domain/document/model/vo"

type DocumentChunk struct {
	ID                 int64  `gorm:"column:id"`                  // ID
	DocumentId         int64  `gorm:"column:document_id"`         // 文档ID
	TaskId             int64  `gorm:"column:task_id"`             // 任务ID
	PlanId             int64  `gorm:"column:plan_id"`             // 计划ID
	ParentBlockId      int64  `gorm:"column:parent_block_id"`     // 父块ID
	ChunkNo            int    `gorm:"column:chunk_no"`            // 块序号
	SourceType         int    `gorm:"column:source_type"`         // 来源类型
	SectionPath        string `gorm:"column:section_path"`        // 章节路径
	StructureNodeId    int64  `gorm:"column:structure_node_id"`   // 结构节点ID
	StructureNodeType  int    `gorm:"column:structure_node_type"` // 结构节点类型
	CanonicalPath      string `gorm:"column:canonical_path"`      // 规范路径
	ItemIndex          int64  `gorm:"column:item_index"`          // 项索引
	ChunkText          string `gorm:"column:chunk_text"`          // 块文本
	CharCount          int    `gorm:"column:char_count"`          // 字符数
	TokenCount         int    `gorm:"column:token_count"`         // token数
	VectorStatus       int    `gorm:"column:vector_status"`       // 向量状态
	VectorStoreType    int    `gorm:"column:vector_store_type"`   // 向量存储类型
	VectorId           string `gorm:"column:vector_id"`           // 向量ID
	ParentBlockNo      int    `gorm:"-"`                          // 父块序号
	ParentChildCount   int    `gorm:"-"`                          // 父子节点数
	ParentStartChunkNo int    `gorm:"-"`                          // 父起始块号
	ParentEndChunkNo   int    `gorm:"-"`                          // 父结束块号
	SourceTypeName     string `gorm:"-"`                          // 来源类型名称
	VectorStatusName   string `gorm:"-"`                          // 向量状态名称
}

func (d *DocumentChunk) FillEnumName() {
	d.VectorStatusName = vo.VectorStatusName(d.VectorStatus)
	d.SourceTypeName = vo.DocumentChunkSourceTypeName(d.SourceType)
}

func (d *DocumentChunk) FillParentInfo(parentBlock *DocumentParentBlock) {
	if parentBlock != nil {
		d.ParentBlockNo = parentBlock.ParentNo
		d.ParentChildCount = parentBlock.ChildCount
		d.ParentStartChunkNo = parentBlock.StartChunkNo
		d.ParentEndChunkNo = parentBlock.EndChunkNo
	}
}
