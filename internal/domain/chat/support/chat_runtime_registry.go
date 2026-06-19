package support

import (
	"sync"

	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
)

// ChatRuntimeRegistry 运行时会话注册表
type ChatRuntimeRegistry struct {
	conversations sync.Map
}

func (r *ChatRuntimeRegistry) Register(conversationCtx *vo.ConversationContext) bool {
	_, loaded := r.conversations.LoadOrStore(conversationCtx.ConversationId, conversationCtx)
	return !loaded
}

func (r *ChatRuntimeRegistry) Get(conversationId string) (*vo.ConversationContext, bool) {
	task, ok := r.conversations.Load(conversationId)
	if !ok {
		return nil, false
	}
	return task.(*vo.ConversationContext), true
}

func (r *ChatRuntimeRegistry) Remove(conversationId string, conversationCtx *vo.ConversationContext) {
	r.conversations.CompareAndDelete(conversationId, conversationCtx)
}
