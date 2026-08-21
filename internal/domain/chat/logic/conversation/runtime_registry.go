package conversation

import (
	"sync"
)

// runtimeRegistry 运行时会话注册表
type runtimeRegistry struct {
	conversations sync.Map
}

func (r *runtimeRegistry) Register(conversationCtx *Context) bool {
	_, loaded := r.conversations.LoadOrStore(conversationCtx.ConversationId, conversationCtx)
	return !loaded
}

func (r *runtimeRegistry) Get(conversationId string) (*Context, bool) {
	task, ok := r.conversations.Load(conversationId)
	if !ok {
		return nil, false
	}
	return task.(*Context), true
}

func (r *runtimeRegistry) Remove(conversationId string, conversationCtx *Context) {
	r.conversations.CompareAndDelete(conversationId, conversationCtx)
}
