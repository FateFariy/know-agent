package callbacks

import (
	"context"
)

type CtxManagerKey struct{}
type CtxRunInfoKey struct{}

type manager struct {
	runInfo        *RunInfo
	handlers       []Handler
	globalHandlers []Handler
}

var GlobalHandlers []Handler

func newManager(runInfo *RunInfo, handlers ...Handler) (*manager, bool) {
	if len(handlers)+len(GlobalHandlers) == 0 {
		return nil, false
	}

	hs := make([]Handler, len(GlobalHandlers))
	copy(hs, GlobalHandlers)

	return &manager{
		globalHandlers: hs,
		handlers:       handlers,
		runInfo:        runInfo,
	}, true
}

// 从 ctx 读取 manager，若不存在返回 nil
func managerFromCtx(ctx context.Context) *manager {
	v := ctx.Value(CtxManagerKey{})
	if m, ok := v.(*manager); ok && m != nil {
		return m
	}

	return nil
}

// 将 manager 写入 ctx
func ctxWithManager(ctx context.Context, m *manager) context.Context {
	return context.WithValue(ctx, CtxManagerKey{}, m)
}

func (m *manager) withRunInfo(runInfo *RunInfo) *manager {
	n := *m
	n.runInfo = runInfo
	return &n
}

// InitCallbacks 初始化或替换 Manager
func InitCallbacks(ctx context.Context, info *RunInfo, handlers ...Handler) context.Context {
	mgr, ok := newManager(info, handlers...)
	if ok {
		return ctxWithManager(ctx, mgr)
	}

	return ctxWithManager(ctx, nil)
}

// AppendHandlers 追加 Handler
func AppendHandlers(ctx context.Context, info *RunInfo, handlers ...Handler) context.Context {
	cbm := managerFromCtx(ctx)
	if cbm == nil {
		return InitCallbacks(ctx, info, handlers...)
	}
	nh := make([]Handler, len(cbm.handlers)+len(handlers))
	copy(nh[:len(cbm.handlers)], cbm.handlers)
	copy(nh[len(cbm.handlers):], handlers)
	return InitCallbacks(ctx, info, nh...)
}

func AppendGlobalHandlers(handlers ...Handler) {
	GlobalHandlers = append(GlobalHandlers, handlers...)
}
