package vo

type DocumentProfile struct {
	DocumentId       int64  // 文档ID
	DocumentSummary  string // 文档摘要
	CoreTopics       string // 核心话题
	ExampleQuestions string // 示例问题
}
