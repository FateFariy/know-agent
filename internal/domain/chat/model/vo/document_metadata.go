package vo

type DocumentMetadata struct {
	DocumentId        int64  `gorm:"column:id"`                  // 文档ID
	DocumentName      string `gorm:"column:document_name"`       // 文档名称
	KnowledgeBaseId   int64  `gorm:"column:knowledge_base_id"`   // 知识库ID
	KnowledgeBaseName string `gorm:"column:knowledge_base_name"` // 知识库名称
	LastIndexTaskId   int64  `gorm:"column:last_index_task_id"`  // 上一次索引任务ID
}
