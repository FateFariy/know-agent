package conversation

import "context"

// agentContextKey 存储请求级 *Context（convCtx）的 ctx key
type agentContextKey struct{}

// WithAgentContext 将 convCtx 注入 ctx
func WithAgentContext(ctx context.Context, convCtx *Context) context.Context {
	return context.WithValue(ctx, agentContextKey{}, convCtx)
}

// AgentContextFrom 从 ctx 取出 convCtx；未注入时返回 nil
func AgentContextFrom(ctx context.Context) *Context {
	if v := ctx.Value(agentContextKey{}); v != nil {
		if convCtx, ok := v.(*Context); ok {
			return convCtx
		}
	}
	return nil
}
