package entity

// KnowledgeDocumentProfile 文档画像
type KnowledgeDocumentProfile struct {
	ID               int64  `gorm:"column:id"`                // 主键
	DocumentId       int64  `gorm:"column:document_id"`       // 文档ID
	DocumentSummary  string `gorm:"column:document_summary"`  // 文档摘要
	DocumentType     string `gorm:"column:document_type"`     // 文档类型
	CoreTopics       string `gorm:"column:core_topics"`       // 核心主题
	ExampleQuestions string `gorm:"column:example_questions"` // 示例问题
	ProfileStatus    int    `gorm:"column:profile_status"`    // 画像状态
}
