package vo

type KnowledgeDocument struct {
	DocumentId         int64  `gorm:"column:id"`                   // 文档ID
	DocumentName       string `gorm:"column:document_name"`        // 文档名称
	KnowledgeScopeCode string `gorm:"column:knowledge_scope_code"` // 知识范围编码
	KnowledgeScopeName string `gorm:"column:knowledge_scope_name"` // 知识范围名称
	BusinessCategory   string `gorm:"column:business_category"`    // 业务分类
	DocumentTags       string `gorm:"column:document_tags"`        // 文档标签
	LastIndexTaskId    int64  `gorm:"column:last_index_task_id"`   // 上一次索引任务ID
}
