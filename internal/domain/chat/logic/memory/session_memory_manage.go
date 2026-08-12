package memory

import (
	"context"

	"github.com/swiftbit/know-agent/internal/domain/chat/logic/memory/strategy"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/aggregate"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/entity"
)

// SessionMemoryManageImpl 会话记忆管理器实现
type SessionMemoryManageImpl struct {
	strategy strategy.Memory
}

func NewSessionMemoryManageImpl(memoryStrategy strategy.Memory) *SessionMemoryManageImpl {
	return &SessionMemoryManageImpl{
		strategy: memoryStrategy,
	}
}

// LoadMemoryContext 加载会话记忆上下文
func (s *SessionMemoryManageImpl) LoadMemoryContext(ctx context.Context, conversationId string) (*aggregate.Conversation, error) {
	return s.strategy.LoadMemoryContext(ctx, conversationId)
}

// RefreshConversationSummaryAsync 异步刷新会话摘要
func (s *SessionMemoryManageImpl) RefreshConversationSummaryAsync(conversationId string) {
	if summaryStrategy, ok := s.strategy.(*strategy.SummaryCompressionStrategy); ok {
		summaryStrategy.RefreshConversationSummaryAsync(conversationId)
	}
}

// GetConversationSummary 获取会话摘要
func (s *SessionMemoryManageImpl) GetConversationSummary(ctx context.Context, conversationId string) (*entity.ChatMemorySummary, error) {
	if summaryStrategy, ok := s.strategy.(*strategy.SummaryCompressionStrategy); ok {
		return summaryStrategy.GetConversationSummary(ctx, conversationId)
	}
	return &entity.ChatMemorySummary{}, nil
}

// RebuildConversationSummary 重建会话摘要（删除现有摘要后重新生成）
func (s *SessionMemoryManageImpl) RebuildConversationSummary(ctx context.Context, conversationId string) (*entity.ChatMemorySummary, error) {
	if summaryStrategy, ok := s.strategy.(*strategy.SummaryCompressionStrategy); ok {
		return summaryStrategy.RebuildConversationSummary(ctx, conversationId)
	}
	return &entity.ChatMemorySummary{}, nil
}

// DeleteConversationSummary 删除会话摘要
func (s *SessionMemoryManageImpl) DeleteConversationSummary(ctx context.Context, conversationId string) error {
	if summaryStrategy, ok := s.strategy.(*strategy.SummaryCompressionStrategy); ok {
		return summaryStrategy.DeleteConversationSummary(ctx, conversationId)
	}
	return nil
}
