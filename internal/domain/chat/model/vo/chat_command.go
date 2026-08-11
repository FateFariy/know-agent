package vo

import "github.com/swiftbit/know-agent/internal/domain/chat/model/enum"

type ChatCommand struct {
	Question           string             // 问题内容
	ConversationId     string             // 会话ID
	ChatMode           enum.ChatQueryMode // 聊天模式
	SelectedDocumentId int64              // 选中的文档ID
}
