package strategy

import (
	"fmt"
)

// MemoryStrategyFactory 记忆策略工厂
type MemoryStrategyFactory struct {
	strategyMap map[string]MemoryStrategy
}

// NewMemoryStrategyFactory 创建记忆策略工厂
func NewMemoryStrategyFactory() *MemoryStrategyFactory {
	return &MemoryStrategyFactory{
		strategyMap: make(map[string]MemoryStrategy),
	}
}

func (f *MemoryStrategyFactory) RegisterStrategy(strategy MemoryStrategy) {
	f.strategyMap[strategy.GetStrategyType()] = strategy
}

// GetStrategy 获取指定类型的策略
func (f *MemoryStrategyFactory) GetStrategy(strategyType string) (MemoryStrategy, error) {
	if strategy, ok := f.strategyMap[strategyType]; ok {
		return strategy, nil
	}
	return nil, fmt.Errorf("unsupported memory strategy type: %s", strategyType)
}
