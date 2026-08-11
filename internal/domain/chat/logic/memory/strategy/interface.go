package strategy

import (
	"context"

	"github.com/swiftbit/know-agent/internal/domain/chat/model/aggregate"
)

// Memory 记忆策略接口
type Memory interface {
	// LoadMemoryContext 加载会话记忆上下文
	LoadMemoryContext(ctx context.Context, conversationId string) (*aggregate.Conversation, error)

	// GetStrategyType 获取记忆策略类型
	GetStrategyType() string
}
