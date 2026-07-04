package memory

import (
	"context"
	"fmt"

	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
)

// Strategy 记忆策略接口
type Strategy interface {
	// LoadMemoryContext 加载会话记忆上下文
	LoadMemoryContext(ctx context.Context, conversationId string, trace *vo.ConversationTrace) (*vo.MemoryContext, error)

	// GetStrategyType 获取记忆策略类型
	GetStrategyType() string
}

// StrategyFactory 记忆策略工厂
type StrategyFactory struct {
	strategyMap map[string]Strategy
}

// NewMemoryStrategyFactory 创建记忆策略工厂
func NewMemoryStrategyFactory() *StrategyFactory {
	return &StrategyFactory{
		strategyMap: make(map[string]Strategy),
	}
}

func (f *StrategyFactory) RegisterStrategy(strategy Strategy) {
	f.strategyMap[strategy.GetStrategyType()] = strategy
}

// GetStrategy 获取指定类型的策略
func (f *StrategyFactory) GetStrategy(strategyType string) (Strategy, error) {
	if strategy, ok := f.strategyMap[strategyType]; ok {
		return strategy, nil
	}
	return nil, fmt.Errorf("unsupported memory strategy type: %s", strategyType)
}
