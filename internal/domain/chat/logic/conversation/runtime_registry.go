package conversation

import (
	"sync"
)

// RuntimeRegistry 运行时会话注册表
type RuntimeRegistry struct {
	conversations sync.Map
}

func NewRuntimeRegistry() *RuntimeRegistry {
	return &RuntimeRegistry{}
}

func (r *RuntimeRegistry) Register(conversationCtx *Context) bool {
	_, loaded := r.conversations.LoadOrStore(conversationCtx.ConversationId, conversationCtx)
	return !loaded
}

func (r *RuntimeRegistry) Get(conversationId string) (*Context, bool) {
	task, ok := r.conversations.Load(conversationId)
	if !ok {
		return nil, false
	}
	return task.(*Context), true
}

func (r *RuntimeRegistry) Remove(conversationId string, conversationCtx *Context) {
	r.conversations.CompareAndDelete(conversationId, conversationCtx)
}
