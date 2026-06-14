package vo

type ChatCommand struct {
	Question           string // 问题内容
	ConversationId     string // 会话ID
	ChatMode           int    // 聊天模式
	SelectedDocumentId int64  // 选中的文档ID
}
