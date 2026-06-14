package model

import "github.com/swiftbit/know-agent/common"

// ChatDialogue 会话记录表
type ChatDialogue struct {
	common.Model
	ConversationId       string `gorm:"column:conversation_id"`        // 会话ID
	SessionStatus        int    `gorm:"column:session_status"`         // 会话状态
	ChatMode             int    `gorm:"column:chat_mode"`              // 聊天模式
	SelectedDocumentId   int64  `gorm:"column:selected_document_id"`   // 选中的文档ID
	SelectedDocumentName string `gorm:"column:selected_document_name"` // 选中的文档名称
}
