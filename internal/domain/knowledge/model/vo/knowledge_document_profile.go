package vo

// KnowledgeDocumentProfile 文档画像
type KnowledgeDocumentProfile struct {
	DocumentId       int64
	DocumentSummary  string
	CoreTopics       string
	ExampleQuestions string
	DocumentType     string
	ProfileStatus    int
	Status           int
}
