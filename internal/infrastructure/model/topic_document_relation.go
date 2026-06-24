package model

import "github.com/swiftbit/know-agent/common"

type TopicDocumentRelation struct {
	common.Model
	topicCode      string  `gorm:"column:topic_code"`      // 主题编码
	documentId     int64   `gorm:"column:document_id"`     // 文档ID
	relationScore  float64 `gorm:"column:relation_score"`  // 关联分数
	relationSource string  `gorm:"column:relation_source"` // 关联来源
	reason         string  `gorm:"column:reason"`          // 关联原因
}
