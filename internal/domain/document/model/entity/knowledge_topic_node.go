package entity

import "github.com/swiftbit/know-agent/common"

// KnowledgeTopicNode 知识话题节点实体
type KnowledgeTopicNode struct {
	common.Model
	TopicCode           string `gorm:"column:topic_code"`           // 话题编码
	TopicName           string `gorm:"column:topic_name"`           // 话题名称
	ScopeCode           string `gorm:"column:scope_code"`           // 范围编码
	Description         string `gorm:"column:description"`          // 描述
	Aliases             string `gorm:"column:aliases"`              // 别名
	Examples            string `gorm:"column:examples"`             // 示例
	AnswerShape         string `gorm:"column:answer_shape"`         // 答案形态
	ExecutionPreference string `gorm:"column:execution_preference"` // 执行偏好
	SortOrder           int    `gorm:"column:sort_order"`           // 排序顺序
}
