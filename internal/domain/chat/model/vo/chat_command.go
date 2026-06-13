package vo

type ChatCommand struct {
	Question           string // 问题内容
	ConversationId     int64  // 会话ID
	ChatMode           string // 聊天模式
	SelectedDocumentId int64  // 选中的文档ID
}
