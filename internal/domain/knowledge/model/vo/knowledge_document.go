package vo

// KnowledgeDocument 可检索文档的核心描述元数据
type KnowledgeDocument struct {
	DocumentId         int64
	DocumentName       string
	KnowledgeScopeCode string
	KnowledgeScopeName string
	BusinessCategory   string
	DocumentTags       string
	LastIndexTaskId    int64
	Status             int
	IndexStatus        int
}
