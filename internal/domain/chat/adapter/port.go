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

type VectorDB interface {
	// SearchByVector 基于 Milvus 向量数据库进行相似度检索
	SearchByVector(ctx context.Context, query *vo.DocumentRetrieve) ([]*vo.DocumentChunk, error)
}

type KeywordDB interface {
	// SearchByKeyword 基于关键词索引进行检索
	SearchByKeyword(ctx context.Context, query *vo.DocumentRetrieve) ([]*vo.DocumentChunk, error)
}
