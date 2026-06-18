package strategy

import (
	"context"

	"github.com/duke-git/lancet/v2/strutil"

	"github.com/swiftbit/know-agent/internal/domain/chat/adapter"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
	"github.com/swiftbit/know-agent/internal/svc"
)

const (
	SlidingWindow string = "sliding_window" // 滑动窗口策略
)

// SlidingWindowStrategy 滑动窗口策略实现
type SlidingWindowStrategy struct {
	baseMemoryStrategy
	repo                     adapter.ChatRepository
	keepRecentTurns          int
	questionHistoryMaxChars  int
	recentTranscriptMaxChars int
}

// NewSlidingWindowStrategy 创建滑动窗口策略实例
func NewSlidingWindowStrategy(svcCtx *svc.ServiceContext, repo adapter.ChatRepository) *SlidingWindowStrategy {
	return &SlidingWindowStrategy{
		repo:                     repo,
		keepRecentTurns:          svcCtx.Config.Memory.RewriteHistoryTurns,
		questionHistoryMaxChars:  svcCtx.Config.Memory.QuestionHistoryMaxChars,
		recentTranscriptMaxChars: svcCtx.Config.Memory.RecentTranscriptMaxChars,
	}
}

// LoadMemoryContext 加载会话记忆上下文（滑动窗口策略）
func (s *SlidingWindowStrategy) LoadMemoryContext(ctx context.Context, conversationId string, tracer *vo.ConversationTrace) (*vo.MemoryContext, error) {
	memoryCtx := &vo.MemoryContext{}
	if strutil.IsBlank(conversationId) {
		return memoryCtx, nil
	}

	// 返回最近对话
	recentExchanges, err := s.repo.ListRecentExchanges(ctx, conversationId, s.keepRecentTurns*3)
	if err != nil {
		return nil, err
	}

	// 渲染最近对话记录
	memoryCtx.RecentTranscript = s.renderRecentTranscript(recentExchanges, s.keepRecentTurns, s.recentTranscriptMaxChars)
	memoryCtx.QuestionRecentTranscript = s.renderRecentQuestionTranscript(recentExchanges, s.keepRecentTurns, s.questionHistoryMaxChars)
	memoryCtx.AssembledHistory = memoryCtx.RecentTranscript

	return memoryCtx, nil
}

// GetStrategyType 获取策略类型
func (s *SlidingWindowStrategy) GetStrategyType() string {
	return SlidingWindow
}
