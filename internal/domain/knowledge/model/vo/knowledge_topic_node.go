package vo

// KnowledgeTopicNode 主题节点
type KnowledgeTopicNode struct {
	TopicCode           string
	TopicName           string
	ScopeCode           string
	Description         string
	Aliases             string
	Examples            string
	AnswerShape         string
	ExecutionPreference string
	Status              int
}
