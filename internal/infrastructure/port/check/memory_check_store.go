package check

import (
	"context"
	"sync"

	"github.com/swiftbit/know-agent/internal/domain/chat/adapter/checkpoint"
)

type MemoryCheckPointStore struct {
	sync.Map
}

// NewMemoryCheckPointStore 创建基于内存的检查点存储器。
func NewMemoryCheckPointStore() *MemoryCheckPointStore {
	return &MemoryCheckPointStore{}
}

var _ checkpoint.Store = (*MemoryCheckPointStore)(nil)

func (m *MemoryCheckPointStore) Get(ctx context.Context, checkPointId string) ([]byte, bool, error) {
	return nil, false, nil
}

func (m *MemoryCheckPointStore) Set(ctx context.Context, checkPointId string, checkPoint []byte) error {
	return nil
}

func (m *MemoryCheckPointStore) Count(ctx context.Context, checkPointId string) (int, error) {
	return 0, nil
}

func (m *MemoryCheckPointStore) Delete(ctx context.Context, checkPointId string) (int, error) {
	return 0, nil
}
