package memory

import (
	"fmt"

	"github.com/swiftbit/know-agent/internal/domain/chat/logic/memory/strategy"
)

// StrategyRegistry 记忆策略注册中心
type StrategyRegistry struct {
	strategyMap map[string]strategy.Memory
}

func NewMemoryStrategyRegistry() *StrategyRegistry {
	return &StrategyRegistry{
		strategyMap: make(map[string]strategy.Memory),
	}
}

func (f *StrategyRegistry) Register(strategy strategy.Memory) {
	f.strategyMap[strategy.GetStrategyType()] = strategy
}

// Get 获取指定类型的策略
func (f *StrategyRegistry) Get(strategyType string) (strategy.Memory, error) {
	if strategy, ok := f.strategyMap[strategyType]; ok {
		return strategy, nil
	}
	return nil, fmt.Errorf("unsupported memory strategy type: %s", strategyType)
}
