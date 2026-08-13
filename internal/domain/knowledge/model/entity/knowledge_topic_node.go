package entity

// KnowledgeTopicNode 主题节点
type KnowledgeTopicNode struct {
	ID                  int64  `gorm:"column:id"`                   // 主键
	TopicName           string `gorm:"column:topic_name"`           // 主题名称
	KnowledgeBaseId     int64  `gorm:"column:knowledge_base_id"`    // 知识库ID
	ScopeId             int64  `gorm:"column:scope_id"`             // 范围ID
	Description         string `gorm:"column:description"`          // 描述
	Aliases             string `gorm:"column:aliases"`              // 别名
	Examples            string `gorm:"column:examples"`             // 示例
	AnswerShape         string `gorm:"column:answer_shape"`         // 答案结构
	ExecutionPreference string `gorm:"column:execution_preference"` // 执行偏好
	SortOrder           int    `gorm:"column:sort_order"`           // 排序顺序
}
