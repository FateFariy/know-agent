package vo

// KnowledgeTopicDocumentRelation 主题-文档映射关系
type KnowledgeTopicDocumentRelation struct {
	TopicCode     string
	DocumentId    int64
	RelationScore float64
	Status        int
}
