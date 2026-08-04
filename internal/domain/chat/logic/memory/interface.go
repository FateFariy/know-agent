package memory

import (
	"context"

	"github.com/swiftbit/know-agent/internal/domain/chat/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
)

// SessionMemoryManager 会话记忆管理器
type SessionMemoryManager interface {
	// LoadMemoryContext 加载会话记忆上下文
	LoadMemoryContext(ctx context.Context, conversationId string) (*vo.MemoryContext, error)

	// RefreshConversationSummaryAsync 异步刷新会话摘要
	RefreshConversationSummaryAsync(conversationId string)

	// GetConversationSummary 获取会话摘要
	GetConversationSummary(ctx context.Context, conversationId string) (*entity.ChatMemorySummary, error)

	// RebuildConversationSummary 重建会话摘要
	RebuildConversationSummary(ctx context.Context, conversationId string) (*entity.ChatMemorySummary, error)

	// DeleteConversationSummary 删除会话摘要
	DeleteConversationSummary(ctx context.Context, conversationId string) error
}
