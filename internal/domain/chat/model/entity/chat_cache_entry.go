package entity

import (
	"github.com/swiftbit/know-agent/internal/domain/chat/model/enum"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
)

// ChatCacheEntry 语义缓存条目
type ChatCacheEntry struct {
	ID                 int64               `gorm:"column:id;"`                                            // 缓存条目ID
	ChatMode           enum.ChatQueryMode  `gorm:"column:chat_mode;"`                                     // 会话模式
	AllowedDocumentIds []int64             `gorm:"column:allowed_document_ids;type:json;serializer:json"` // 允许文档ID
	AllowedTaskIds     []int64             `gorm:"column:allowed_task_ids;type:json;serializer:json"`     // 允许任务ID
	KnowledgeBaseIds   []int64             `gorm:"column:knowledge_base_ids;type:json;serializer:json"`   // 知识库ID列表
	QueryText          string              `gorm:"column:query_text;"`                                    // 用户查询文本
	Execution          *vo.CachedExecution `gorm:"column:execution;type:json;serializer:json"`            // 执行结果缓存
	AnswerDraft        string              `gorm:"column:answer_draft;"`                                  // 回答草稿
	Similarity         float32             `gorm:"-"`                                                     // 相似度
}

// Validate 校验可用性
func (e *ChatCacheEntry) Validate() bool {
	if e == nil || e.Execution == nil {
		return false
	}
	if e.Execution.RetrievalResult.IsEmpty() {
		return false
	}
	return true
}
