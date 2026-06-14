package adapter

import "context"

type DistributedLock interface {
	// TryLock 尝试获取锁
	TryLock(ctx context.Context) error

	// Lock 获取锁
	Lock(ctx context.Context) error

	// Unlock 释放锁
	Unlock(ctx context.Context) error

	// Extend 锁续期
	Extend(ctx context.Context) error
}
