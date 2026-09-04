package vo

import (
	"github.com/swiftbit/know-agent/internal/domain/chat/model/enum"
)

// CacheScope 语义缓存的数据隔离
type CacheScope struct {
	ChatMode           enum.ChatQueryMode
	AllowedDocumentIds []int64
	AllowedTaskIds     []int64
	KnowledgeBaseIds   []int64
}

// CachedExecution 语义缓存仅保留可安全复用的执行产物
type CachedExecution struct {
	RetrievalResult *RetrievalResult // 检索结果
}
