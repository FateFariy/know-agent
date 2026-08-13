package model

import "github.com/swiftbit/know-agent/common"

// KnowledgeTopicDocumentRelation 主题-文档映射关系
type KnowledgeTopicDocumentRelation struct {
	common.Model
	KnowledgeBaseId int64   `gorm:"column:knowledge_base_id;type:bigint"`     // 所属知识库ID
	TopicId         int64   `gorm:"column:topic_id;type:bigint"`              // 关联Topic ID
	DocumentId      int64   `gorm:"column:document_id;type:bigint"`           // 关联文档ID
	RelationScore   float64 `gorm:"column:relation_score;type:decimal(10,4)"` // 关联置信度
	RelationSource  string  `gorm:"column:relation_source;type:varchar(255)"` // 关系来源
	Reason          string  `gorm:"column:reason;type:text"`                  // 关联原因
}
