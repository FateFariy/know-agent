package entity

import "time"

// ChatDialogue 会话记录
type ChatDialogue struct {
	ID                         int64     `gorm:"column:id"`                                                           // 对话ID
	ConversationId             string    `gorm:"column:conversation_id"`                                              // 会话ID
	SessionStatus              int       `gorm:"column:session_status"`                                               // 会话状态
	ChatMode                   int       `gorm:"column:chat_mode"`                                                    // 聊天模式
	SelectedDocumentId         int64     `gorm:"column:selected_document_id"`                                         // 选中的文档ID
	SelectedDocumentName       string    `gorm:"column:selected_document_name"`                                       // 选中的文档名称
	KnowledgeBaseSelectionMode string    `gorm:"column:knowledge_base_selection_mode"`                                // 知识库选择模式
	SelectedKnowledgeBaseIds   []int64   `gorm:"column:selected_knowledge_base_ids_json;type:json;serializer:json"`   // 选中知识库ID列表JSON
	SelectedKnowledgeBaseNames []string  `gorm:"column:selected_knowledge_base_names_json;type:json;serializer:json"` // 选中知识库名称列表JSON
	CreateTime                 time.Time `gorm:"column:create_time"`                                                  // 创建时间
	UpdateTime                 time.Time `gorm:"column:update_time"`                                                  // 更新时间
	Question                   string    `gorm:"-"`                                                                   // 问题
}
