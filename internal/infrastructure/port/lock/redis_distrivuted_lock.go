package lock

import (
	"context"
	"sync"
	"time"

	"github.com/go-redsync/redsync/v4"

	"github.com/swiftbit/know-agent/internal/domain/chat/adapter"
	errorx "github.com/swiftbit/know-agent/internal/error"
)

type RedisMutexLock struct {
	redSync  *redsync.Redsync
	mutexMap sync.Map
}

var _ adapter.DistributedLock = (*RedisMutexLock)(nil)

func NewRedisMutexLock(redSync *redsync.Redsync) *RedisMutexLock {
	return &RedisMutexLock{
		redSync: redSync,
	}
}

// TryLock 尝试加锁分布式锁
func (r *RedisMutexLock) TryLock(ctx context.Context, name string) error {
	return r.getOrStoreMutex(name).TryLockContext(ctx)
}

// Lock 加锁分布式锁
func (r *RedisMutexLock) Lock(ctx context.Context, name string) error {
	return r.getOrStoreMutex(name).LockContext(ctx)
}

// Unlock 释放分布式锁
func (r *RedisMutexLock) Unlock(ctx context.Context, name string) error {
	if mutex, ok := r.getMutex(name); ok {
		r.mutexMap.Delete(name)
		_, err := mutex.UnlockContext(ctx)
		return err
	}
	return errorx.ErrDistributedLockNotFound.Format(name)
}

// Extend 续期分布式锁租约
func (r *RedisMutexLock) Extend(ctx context.Context, name string) error {
	if mutex, ok := r.getMutex(name); ok {
		_, err := mutex.ExtendContext(ctx)
		return err
	}
	return errorx.ErrDistributedLockNotFound.Format(name)
}

// getMutex 获取分布式锁
func (r *RedisMutexLock) getMutex(name string) (*redsync.Mutex, bool) {
	if value, ok := r.mutexMap.Load(name); ok {
		return value.(*redsync.Mutex), true
	}
	return nil, false
}

// getOrStoreMutex 获取或存储分布式锁
func (r *RedisMutexLock) getOrStoreMutex(name string) *redsync.Mutex {
	expiry := redsync.WithExpiry(10 * time.Second)
	mutex := r.redSync.NewMutex(name, expiry)
	if value, ok := r.mutexMap.LoadOrStore(name, mutex); !ok {
		mutex, ok = value.(*redsync.Mutex)
	}
	return mutex
}
