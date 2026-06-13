package vo

type KnowledgeDocumentDescriptor struct {
	DocumentId         int64  // 文档ID
	DocumentName       string // 文档名称
	LastIndexTaskId    int64  // 最后一次索引任务ID
	KnowledgeScopeCode string // 知识域代码
	KnowledgeScopeName string // 知识域名称
	BusinessCategory   string // 业务类别
	DocumentTags       string // 文档标签
}
