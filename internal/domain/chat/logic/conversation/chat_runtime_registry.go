package conversation

import (
	"sync"
)

// ChatRuntimeRegistry 运行时会话注册表
type ChatRuntimeRegistry struct {
	conversations sync.Map
}

func (r *ChatRuntimeRegistry) Register(conversationCtx *Context) bool {
	_, loaded := r.conversations.LoadOrStore(conversationCtx.ConversationId, conversationCtx)
	return !loaded
}

func (r *ChatRuntimeRegistry) Get(conversationId string) (*Context, bool) {
	task, ok := r.conversations.Load(conversationId)
	if !ok {
		return nil, false
	}
	return task.(*Context), true
}

func (r *ChatRuntimeRegistry) Remove(conversationId string, conversationCtx *Context) {
	r.conversations.CompareAndDelete(conversationId, conversationCtx)
}
