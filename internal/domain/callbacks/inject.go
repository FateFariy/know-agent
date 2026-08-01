package callbacks

import (
	"context"
)

// OnStart 供业务组件在方法开头调用
func OnStart[T any](ctx context.Context, input T) context.Context {
	ctx, _ = On(ctx, input, TimingOnStart, true)
	return ctx
}

// OnEnd 供业务组件在成功返回时调用
func OnEnd[T any](ctx context.Context, output T) context.Context {
	ctx, _ = On(ctx, output, TimingOnEnd, false)
	return ctx
}

// OnError 供业务组件在出错时调用
func OnError(ctx context.Context, err error) context.Context {
	ctx, _ = On(ctx, err, TimingOnError, false)
	return ctx
}
