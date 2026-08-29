package conversation

import (
	"github.com/swiftbit/know-agent/internal/domain/chat/model/enum"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
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
	Mode                 enum.ExecutionMode          // 执行模式
	RetrievalPlan        *vo.RetrievalPlan           // 检索计划
	RetrievalResult      *vo.RetrievalResult         // 检索结果
	PromptAssemblyResult *vo.RagPromptAssemblyResult // Prompt 组装结果
}

// CacheHit 语义缓存命中结果
type CacheHit struct {
	Entry      *CacheEntry
	Similarity float64
}
