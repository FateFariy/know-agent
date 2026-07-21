package executor

import (
	"github.com/duke-git/lancet/v2/strutil"

	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
	"github.com/swiftbit/know-agent/internal/domain/chat/support"
)

const (
	defaultNoEvidenceReply = "当前没有足够证据支持明确回答。"
)

// builder 包级别的流式事件构造器
var builder = support.StreamEventBuilder{}

// singleValueChan 将给定字符串包装为一个已关闭的带缓冲只读 channel，便于与流式管道拼接
func singleValueChan(content string) <-chan string {
	ch := make(chan string, 1)
	defer close(ch)
	ch <- content
	return ch
}

// publishThinking 发布思考事件
func publishThinking(convCtx *vo.ConversationContext, content string) error {
	if convCtx == nil || strutil.IsBlank(content) {
		return nil
	}
	convCtx.AddThinkingSteps(content)
	payload := builder.Thinking(content, convCtx.ConversationId, convCtx.ExchangeId)
	return support.SafeEmitNext(convCtx.Channel, payload)
}

// publishStatus 发布状态事件
func publishStatus(convCtx *vo.ConversationContext, content string) error {
	if convCtx == nil || strutil.IsBlank(content) {
		return nil
	}
	payload := builder.Status(content, convCtx.ConversationId, convCtx.ExchangeId)
	return support.SafeEmitNext(convCtx.Channel, payload)
}
