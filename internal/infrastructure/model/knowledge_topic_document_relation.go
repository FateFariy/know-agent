package model

import "github.com/swiftbit/know-agent/common"

// KnowledgeTopicDocumentRelation 主题-文档映射关系
type KnowledgeTopicDocumentRelation struct {
	common.Model
	TopicCode      string  `gorm:"column:topic_code"`      // 主题编码
	DocumentId     int64   `gorm:"column:document_id"`     // 文档ID
	RelationScore  float64 `gorm:"column:relation_score"`  // 关联分数
	RelationSource string  `gorm:"column:relation_source"` // 关联来源
	Reason         string  `gorm:"column:reason"`          // 关联原因
}
