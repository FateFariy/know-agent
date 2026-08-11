package adapter

import (
	"context"

	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
)

// CheckPointStore 检查点存储器
type CheckPointStore interface {
	// Get 获取检查点
	Get(ctx context.Context, checkPointId string) ([]byte, bool, error)

	// Set 设置检查点
	Set(ctx context.Context, checkPointId string, checkPoint []byte) error

	// Count 检查点数量
	Count(ctx context.Context, checkPointId string) (int, error)

	// Delete 删除检查点
	Delete(ctx context.Context, checkPointId string) (int, error)
}

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

type KeywordRetriever interface {
	// SearchByKeyword  按关键词检索
	SearchByKeyword(ctx context.Context, query *vo.DocumentRetrieve) ([]*vo.DocumentChunk, error)
}

type VectorRetriever interface {
	// SearchByVector 按向量检索
	SearchByVector(ctx context.Context, query *vo.DocumentRetrieve) ([]*vo.DocumentChunk, error)
}

type DocumentFetcher interface {
	// FetchRetrieveDocuments 获取文档
	FetchRetrieveDocuments(ctx context.Context, id string) ([]byte, error)

	// QueryParentChunks 查询父块
	QueryParentChunks(ctx context.Context, id string) ([]*vo.DocumentChunk, error)
}

// Sink 面向客户端的事件输出端口
type Sink interface {
	WriteFrame(e *entity.Event) error
}
