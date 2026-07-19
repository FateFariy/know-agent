package adapter

import (
	"context"

	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
)

type DistributedLock interface {
	// TryLock 尝试获取锁
	TryLock(ctx context.Context, name string) error

	// Lock 获取锁
	Lock(ctx context.Context, name string) error

	// Unlock 释放锁
	Unlock(ctx context.Context, name string) error

	// Extend 锁续期
	Extend(ctx context.Context, name string) error
}

type Retriever interface {
	// Search 检索
	Search(ctx context.Context, query *vo.DocumentRetrieve) ([]*vo.DocumentChunk, error)
}

// CheckPointStore 检查点存储器
type CheckPointStore interface {
	// Get 获取检查点
	Get(ctx context.Context, checkPointID string) ([]byte, bool, error)

	// Set 设置检查点
	Set(ctx context.Context, checkPointID string, checkPoint []byte) error

	// Count 检查点数量
	Count(ctx context.Context, checkPointID string) (int, error)

	// Delete 删除检查点
	Delete(ctx context.Context, checkPointID string) (int, error)
}

type RerankOption struct {
	Model string // 重排序模型
	TopN  int    // 重排序TopN
}

type Option func(opt *RerankOption)

func WithModel(model string) Option {
	return func(opt *RerankOption) {
		opt.Model = model
	}
}

func WithTopN(topN int) Option {
	return func(opt *RerankOption) {
		opt.TopN = max(1, topN)
	}
}

func GetCommonOptions(base *RerankOption, opts ...Option) *RerankOption {
	if base == nil {
		base = &RerankOption{}
	}

	for _, opt := range opts {
		opt(base)
	}

	return base
}

// Reranker 重排序器
type Reranker interface {
	// Process 重排序
	Process(ctx context.Context, question string, chunks []*vo.DocumentChunk, opts ...Option) ([]*vo.DocumentChunk, error)
}
