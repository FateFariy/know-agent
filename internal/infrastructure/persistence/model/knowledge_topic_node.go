package model

import "github.com/swiftbit/know-agent/common"

// KnowledgeTopicNode 知识话题节点实体
type KnowledgeTopicNode struct {
	common.Model
	KnowledgeBaseId     int64  `gorm:"column:knowledge_base_id;type:bigint"`         // 知识库ID
	TopicName           string `gorm:"column:topic_name;type:varchar(255)"`          // 主题名称
	ScopeId             int64  `gorm:"column:scope_id;type:bigint"`                  // 范围ID
	Description         string `gorm:"column:description;type:text"`                 // 描述
	Aliases             string `gorm:"column:aliases;type:text"`                     // 别名
	Examples            string `gorm:"column:examples;type:text"`                    // 示例
	AnswerShape         string `gorm:"column:answer_shape;type:text"`                // 答案结构
	ExecutionPreference string `gorm:"column:execution_preference;type:varchar(50)"` // 执行偏好
	SortOrder           int    `gorm:"column:sort_order;type:int"`                   // 排序顺序
}
