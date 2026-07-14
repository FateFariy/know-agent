package entity

// KnowledgeTopicNode 主题节点
type KnowledgeTopicNode struct {
	ID                  int64  `gorm:"column:id"`                   // 主键
	TopicCode           string `gorm:"column:topic_code"`           // 话题编码
	TopicName           string `gorm:"column:topic_name"`           // 话题名称
	ScopeCode           string `gorm:"column:scope_code"`           // 范围编码
	Description         string `gorm:"column:description"`          // 描述
	Aliases             string `gorm:"column:aliases"`              // 别名
	Examples            string `gorm:"column:examples"`             // 示例
	AnswerShape         string `gorm:"column:answer_shape"`         // 答案形态
	ExecutionPreference string `gorm:"column:execution_preference"` // 执行偏好
	SortOrder           int    `gorm:"column:sort_order"`           // 排序顺序
	OperatorId          string `gorm:"-"`                           // 操作员ID
}
