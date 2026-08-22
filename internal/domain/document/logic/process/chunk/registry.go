package chunk

import (
	"github.com/swiftbit/know-agent/internal/domain/document/model/enum"
)

type Registry struct {
	registry map[string]Chunker
}

func NewChunkStrategyRegistry(strategies []Chunker) *Registry {
	registry := make(map[string]Chunker)
	for _, strategy := range strategies {
		registry[strategy.Name()] = strategy
	}

	return &Registry{registry: registry}
}

func (r *Registry) Get(strategy int) Chunker {
	return r.registry[enum.StrategyTypeName(strategy)]
}
