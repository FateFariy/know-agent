package conversation

import (
	"time"

	"github.com/swiftbit/know-agent/internal/domain/chat/model/enum"
)

// CacheEntry 语义缓存条目
type CacheEntry struct {
	ID          string
	Version     int
	Scope       *CacheScope
	QueryText   string           // 改写问题
	Execution   *CachedExecution // 可复用执行产物
	AnswerDraft string           // 始终填充（双写）
	Hint        enum.ReuseStrategy
	HitCount    int
	CreatedAt   time.Time
	UpdatedAt   time.Time
	ExpireAt    time.Time
}

// Validate 校验可用性
func (e *CacheEntry) Validate() bool {
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
