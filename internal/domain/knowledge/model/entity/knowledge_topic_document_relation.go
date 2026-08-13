package entity

// KnowledgeTopicDocumentRelation 主题-文档映射关系
type KnowledgeTopicDocumentRelation struct {
	ID              int64   `gorm:"column:id"`                // 主键
	KnowledgeBaseId int64   `gorm:"column:knowledge_base_id"` // 所属知识库ID
	TopicId         int64   `gorm:"column:topic_id"`          // 关联Topic ID
	DocumentId      int64   `gorm:"column:document_id"`       // 关联文档ID
	RelationScore   float64 `gorm:"column:relation_score"`    // 关联置信度
	RelationSource  string  `gorm:"column:relation_source"`   // 关系来源
	Reason          string  `gorm:"column:reason"`            // 关联原因
	DocumentName    string  `gorm:"-"`                        // 文档名称
}
