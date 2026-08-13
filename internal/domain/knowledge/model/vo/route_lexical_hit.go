package vo

// RouteLexicalHit 词面索引命中结果
type RouteLexicalHit struct {
	RouteId         string
	EntityType      string
	DocumentId      int64
	KnowledgeBaseId int64
	ScopeId         int64
	TopicId         int64
	DocumentName    string
	Score           float64
}
