package vo

type ChatCommand struct {
	Question                   string   // 问题内容
	ConversationId             string   // 会话ID
	ChatMode                   string   // 聊天模式
	SelectedDocumentId         int64    // 选中的文档ID
	KnowledgeBaseSelectionMode string   // 知识库选择模式
	SelectedKnowledgeBaseIds   []string // 选择的知识库Ids
}
