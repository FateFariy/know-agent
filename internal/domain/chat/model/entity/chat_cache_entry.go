package entity

import (
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
)

// ChatCacheEntry 语义缓存条目
type ChatCacheEntry struct {
	ID          int64               `gorm:"column:id;"`                                 // 缓存条目ID
	Scope       *vo.CacheScope      `gorm:"column:scope;type:json;serializer:json"`     // 缓存作用域
	QueryText   string              `gorm:"column:query_text;"`                         // 用户查询文本
	Execution   *vo.CachedExecution `gorm:"column:execution;type:json;serializer:json"` // 执行结果缓存
	AnswerDraft string              `gorm:"column:answer_draft;"`                       // 回答草稿
}

// Validate 校验可用性
func (e *ChatCacheEntry) Validate() bool {
	if e == nil || e.Execution == nil {
		return false
	}
	if e.Execution.RetrievalResult.IsEmpty() {
		return false
	}
	if e.Execution.PromptAssemblyResult == nil {
		return false
	}
	return true
}
