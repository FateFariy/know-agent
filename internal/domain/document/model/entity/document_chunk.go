package entity

type DocumentChunk struct {
	ID                 int64  `gorm:"column:id"`                  // ID
	DocumentId         int64  `gorm:"column:document_id"`         // 文档ID
	TaskId             int64  `gorm:"column:task_id"`             // 任务ID
	PlanId             int64  `gorm:"column:plan_id"`             // 计划ID
	ParentBlockId      int64  `gorm:"column:parent_block_id"`     // 父块ID
	ChunkNo            int64  `gorm:"column:chunk_no"`            // 块序号
	SourceType         int64  `gorm:"column:source_type"`         // 来源类型
	SectionPath        string `gorm:"column:section_path"`        // 章节路径
	StructureNodeId    int64  `gorm:"column:structure_node_id"`   // 结构节点ID
	StructureNodeType  int64  `gorm:"column:structure_node_type"` // 结构节点类型
	CanonicalPath      string `gorm:"column:canonical_path"`      // 规范路径
	ItemIndex          int64  `gorm:"column:item_index"`          // 项索引
	ChunkText          string `gorm:"column:chunk_text"`          // 块文本
	CharCount          int64  `gorm:"column:char_count"`          // 字符数
	TokenCount         int64  `gorm:"column:token_count"`         // token数
	VectorStatus       int64  `gorm:"column:vector_status"`       // 向量状态
	VectorStoreType    int64  `gorm:"column:vector_store_type"`   // 向量存储类型
	VectorId           string `gorm:"column:vector_id"`           // 向量ID
	ParentBlockNo      int    `gorm:"column:-"`                   // 父块序号
	ParentChildCount   int    `gorm:"column:-"`                   // 父子节点数
	ParentStartChunkNo int    `gorm:"column:-"`                   // 父起始块号
	ParentEndChunkNo   int    `gorm:"column:-"`                   // 父结束块号
	SourceTypeName     string `gorm:"column:-"`                   // 来源类型名称
	VectorStatusName   string `gorm:"column:-"`                   // 向量状态名称
}
