package callbacks

import "context"

// On 是总入口
func On[T any](ctx context.Context, inOut T, timing Timing, isStart bool) (context.Context, T) {
	m := managerFromCtx(ctx)
	if m == nil {
		return ctx, inOut
	}

	// 1. 深拷贝当前 manager，防止后续修改污染原始 ctx（因为 ctx 是不可变的）
	newM := *m

	var info *RunInfo
	// 如果是 OnStart，将 runInfo 拿出来存入 ctx 供后续使用
	if isStart {
		info = newM.runInfo
		newM.runInfo = nil
		ctx = context.WithValue(ctx, CtxRunInfoKey{}, info)
	} else {
		// 如果是 OnEnd/OnError，从 ctx 中读取之前的 runInfo
		if v := ctx.Value(CtxRunInfoKey{}); v != nil {
			info = v.(*RunInfo)
		} else {
			info = newM.runInfo // fallback
		}
	}
	hs := append([]Handler{}, newM.globalHandlers...)
	hs = append(hs, newM.handlers...)

	// 2. 根据时序决定执行顺序
	if timing == TimingOnStart {
		// 倒序执行（洋葱入栈）：从最后一个 Handler 开始
		for i := len(hs) - 1; i >= 0; i-- {
			ctx = hs[i].OnStart(ctx, info, inOut)
		}
		return ctxWithManager(ctx, &newM), inOut
	}

	if timing == TimingOnEnd {
		// 正序执行（洋葱出栈）：从第一个 Handler 开始
		for _, h := range hs {
			ctx = h.OnEnd(ctx, info, inOut)
		}
		return ctxWithManager(ctx, &newM), inOut
	}

	// Error 逻辑同理，正序执行
	if timing == TimingOnError {
		// 注意：error 要传给 Handler，但此处 inOut 是 error，需要类型断言
		if err, ok := any(inOut).(error); ok {
			for _, h := range hs {
				ctx = h.OnError(ctx, info, err)
			}
		}
		return ctxWithManager(ctx, &newM), inOut
	}

	return ctx, inOut
}

// EnsureRunInfo 懒加载初始化
func EnsureRunInfo(ctx context.Context, info *RunInfo) context.Context {
	cbm := managerFromCtx(ctx)
	if cbm == nil {
		return InitCallbacks(ctx, info)
	}
	if cbm.runInfo == nil {
		return ReuseHandlers(ctx, info)
	}
	return ctx
}

func ReuseHandlers(ctx context.Context, info *RunInfo) context.Context {
	cbm := managerFromCtx(ctx)
	if cbm == nil {
		return InitCallbacks(ctx, info)
	}
	return ctxWithManager(ctx, cbm.withRunInfo(info))
}
