package chunk

import (
	"context"
	"fmt"
	"sync"
)

// Registry 策略注册表
// 以策略名称为 key 存储策略实例，支持动态注册与选择。
// 使用 sync.RWMutex 以允许多 goroutine 并发读，以及写时加锁。
type Registry struct {
	mu         sync.RWMutex
	strategies map[string]Strategy
}

// NewRegistry 创建一个新的注册表，并默认注册内置的四种策略。
// 调用方可通过 RegisterOverride 替换默认实现。
func NewRegistry() *Registry {
	return &Registry{
		strategies: make(map[string]Strategy, 4),
	}
}

// Register 注册策略，如果名称已存在则返回错误
func (r *Registry) Register(s Strategy) error {
	if s == nil {
		return fmt.Errorf("chunk: strategy is nil")
	}
	name := s.Name()
	if name == "" {
		return fmt.Errorf("chunk: strategy name is empty")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.strategies[name]; exists {
		return fmt.Errorf("chunk: strategy %q already registered", name)
	}
	r.strategies[name] = s
	return nil
}

// RegisterOverride 注册或覆盖同名策略
func (r *Registry) RegisterOverride(s Strategy) {
	if s == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.strategies[s.Name()] = s
}

// Unregister 移除指定策略
func (r *Registry) Unregister(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.strategies, name)
}

// Has 判断指定名称的策略是否已注册
func (r *Registry) Has(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.strategies[name]
	return ok
}

// Get 根据名称获取策略，若不存在则返回 nil,false
func (r *Registry) Get(name string) (Strategy, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	s, ok := r.strategies[name]
	return s, ok
}

// Names 返回已注册策略的名称列表（不保证顺序）
func (r *Registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.strategies))
	for n := range r.strategies {
		names = append(names, n)
	}
	return names
}

// Run 按名称执行单一策略，找不到时返回错误
func (r *Registry) Run(ctx context.Context, name string, input *Input) ([]*Output, error) {
	s, ok := r.Get(name)
	if !ok {
		return nil, fmt.Errorf("chunk: strategy %q not found", name)
	}
	return s.Chunk(ctx, input)
}

// RunPipeline 顺序执行一组策略，前一个策略的每个输出会作为下一个策略的输入
func (r *Registry) RunPipeline(ctx context.Context, names []string, input *Input) ([]*Output, error) {
	if input == nil {
		return nil, nil
	}
	currentInputs := []*Input{input}
	for _, name := range names {
		s, ok := r.Get(name)
		if !ok {
			return nil, fmt.Errorf("chunk: strategy %q not found", name)
		}
		nextInputs := make([]*Output, 0, len(currentInputs))
		for _, in := range currentInputs {
			outputs, err := s.Chunk(ctx, in)
			if err != nil {
				return nil, err
			}
			nextInputs = append(nextInputs, outputs...)
		}
		// 将 outputs 转换回 Inputs，传递给下一级策略
		currentInputs = make([]*Input, 0, len(nextInputs))
		for _, out := range nextInputs {
			currentInputs = append(currentInputs, &Input{
				SectionPath:   out.SectionPath,
				CanonicalPath: out.CanonicalPath,
				ItemIndex:     out.ItemIndex,
				Text:          out.Text,
				SourceType:    out.SourceType,
			})
		}
	}

	// 将最后一级的 Inputs 转回 Outputs 输出
	result := make([]*Output, 0, len(currentInputs))
	for _, in := range currentInputs {
		result = append(result, &Output{
			SectionPath:   in.SectionPath,
			CanonicalPath: in.CanonicalPath,
			ItemIndex:     in.ItemIndex,
			Text:          in.Text,
			SourceType:    in.SourceType,
		})
	}
	return result, nil
}
