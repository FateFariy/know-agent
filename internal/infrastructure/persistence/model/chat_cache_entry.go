package model

import (
	"github.com/swiftbit/know-agent/common"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
)

// ChatCacheEntry 语义缓存条目
type ChatCacheEntry struct {
	common.Model
	ChatMode           int                 `gorm:"column:chat_mode;"`                                     // 会话模式
	AllowedDocumentIds []int64             `gorm:"column:allowed_document_ids;type:json;serializer:json"` // 允许文档ID
	AllowedTaskIds     []int64             `gorm:"column:allowed_task_ids;type:json;serializer:json"`     // 允许任务ID
	KnowledgeBaseIds   []int64             `gorm:"column:knowledge_base_ids;type:json;serializer:json"`   // 知识库ID列表
	QueryText          string              `gorm:"column:query_text;type:text;"`                          // 用户查询文本
	Execution          *vo.CachedExecution `gorm:"column:execution;type:json;serializer:json"`            // 执行结果缓存
	AnswerDraft        string              `gorm:"column:answer_draft;type:text"`                         // 回答草稿
}
