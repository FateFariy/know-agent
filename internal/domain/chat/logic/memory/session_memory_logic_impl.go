package memory

import (
	"context"

	"github.com/swiftbit/know-agent/internal/domain/chat/logic/memory/strategy"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
)

// SessionMemoryLogicImpl 会话记忆逻辑实现
type SessionMemoryLogicImpl struct {
	memoryStrategy strategy.MemoryStrategy
}

// NewSessionMemoryLogic 创建会话记忆逻辑实例
func NewSessionMemoryLogic(memoryStrategy strategy.MemoryStrategy) *SessionMemoryLogicImpl {
	return &SessionMemoryLogicImpl{
		memoryStrategy: memoryStrategy,
	}
}

// LoadMemoryContext 加载会话记忆上下文
func (s *SessionMemoryLogicImpl) LoadMemoryContext(ctx context.Context, conversationId string, trace *vo.ConversationTrace) (*vo.MemoryContext, error) {
	return s.memoryStrategy.LoadMemoryContext(ctx, conversationId, trace)
}

// RefreshConversationSummaryAsync 异步刷新会话摘要
func (s *SessionMemoryLogicImpl) RefreshConversationSummaryAsync(ctx context.Context, conversationId string) {
	if summaryStrategy, ok := s.memoryStrategy.(*strategy.SummaryCompressionStrategy); ok {
		summaryStrategy.RefreshConversationSummaryAsync(ctx, conversationId)
	}
}

// GetConversationSummary 获取会话摘要
func (s *SessionMemoryLogicImpl) GetConversationSummary(ctx context.Context, conversationId string) (*entity.ChatMemorySummary, error) {
	if summaryStrategy, ok := s.memoryStrategy.(*strategy.SummaryCompressionStrategy); ok {
		return summaryStrategy.GetConversationSummary(ctx, conversationId)
	}
	return &entity.ChatMemorySummary{}, nil
}

// RebuildConversationSummary 重建会话摘要（删除现有摘要后重新生成）
func (s *SessionMemoryLogicImpl) RebuildConversationSummary(ctx context.Context, conversationId string) (*entity.ChatMemorySummary, error) {
	if summaryStrategy, ok := s.memoryStrategy.(*strategy.SummaryCompressionStrategy); ok {
		return summaryStrategy.RebuildConversationSummary(ctx, conversationId)
	}
	return &entity.ChatMemorySummary{}, nil
}

// DeleteConversationSummary 删除会话摘要
func (s *SessionMemoryLogicImpl) DeleteConversationSummary(ctx context.Context, conversationId string) error {
	if summaryStrategy, ok := s.memoryStrategy.(*strategy.SummaryCompressionStrategy); ok {
		return summaryStrategy.DeleteConversationSummary(ctx, conversationId)
	}
	return nil
}
