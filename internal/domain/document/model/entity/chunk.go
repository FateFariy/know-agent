package entity

// TopicDocumentRelation 话题文档关联实体
type TopicDocumentRelation struct {
	ID             int64  `gorm:"column:id"`              // 主键ID
	TopicCode      string `gorm:"column:topic_code"`      // 话题编码
	DocumentId     int64  `gorm:"column:document_id"`     // 文档ID
	RelationScore  string `gorm:"column:relation_score"`  // 关联分数
	RelationSource string `gorm:"column:relation_source"` // 关联来源
	Reason         string `gorm:"column:reason"`          // 关联原因
}
