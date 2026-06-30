package persistence

import (
	"context"

	"gorm.io/gorm"
)

// transactionCtxKey 上下文存储事务gorm.DB唯一key
var transactionCtxKey struct{}

// transactionManager 事务管理器，负责从上下文读取事务DB、注入事务到上下文
type transactionManager struct {
	db *gorm.DB
}

// Do 开启事务并执行 fn, 将事务 DB 注入 context
func (t *transactionManager) Do(ctx context.Context, fn func(txCtx context.Context) error) error {
	return t.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txCtx := context.WithValue(ctx, transactionCtxKey, tx)
		return fn(txCtx)
	})
}

// dbWithContext 优先读取上下文内事务db，无事务则返回默认db，自动绑定传入ctx
//
// - 若在 Do 的回调中（ctx 已注入 tx），返回该事务 tx
// - 否则返回 Repository 持有的默认 db（每次操作为独立 auto-commit）
func (t *transactionManager) dbWithContext(ctx context.Context) *gorm.DB {
	if v := ctx.Value(transactionCtxKey); v != nil {
		if tx, ok := v.(*gorm.DB); ok {
			return tx.WithContext(ctx)
		}
	}
	return t.db.WithContext(ctx)
}
