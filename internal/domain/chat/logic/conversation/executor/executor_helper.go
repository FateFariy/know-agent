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

// singleValueChan 将给定字符串包装为一个已关闭的带缓冲只读 channel，便于与流式管道拼接。
// channel 容量为 1，内容立即写入并在函数返回前通过 defer 关闭，接收方可直接读取。
func singleValueChan(content string) <-chan string {
	ch := make(chan string, 1)
	defer close(ch)
	ch <- content
	return ch
}

// ========================
// 流式事件下发辅助
// ========================

// publishThinking 发布思考事件
func publishThinking(convCtx *vo.ConversationContext, content string) error {
	if convCtx == nil || strutil.IsBlank(content) {
		return nil
	}
	convCtx.AddThinkingSteps(content)
	payload := builder.ThinkingWithMetadata(content, convCtx.ConversationId, convCtx.ExchangeId)
	return support.SafeEmitNext(convCtx.Channel, payload)
}

// publishStatus 发布状态事件
func publishStatus(convCtx *vo.ConversationContext, content string) error {
	if convCtx == nil || strutil.IsBlank(content) {
		return nil
	}
	payload := builder.StatusWithMetadata(content, convCtx.ConversationId, convCtx.ExchangeId)
	return support.SafeEmitNext(convCtx.Channel, payload)
}

// publishReferences 发布引用事件
func publishReferences(convCtx *vo.ConversationContext, refs []*vo.SearchReference) error {
	if convCtx == nil || len(refs) == 0 {
		return nil
	}
	convCtx.AddReferences(refs...)
	payload := builder.ReferencesWithMetadata(refs, convCtx.ConversationId, convCtx.ExchangeId)
	return support.SafeEmitNext(convCtx.Channel, payload)
}

// publishRecommendations 发布推荐追问事件
func publishRecommendations(convCtx *vo.ConversationContext, recommendations []string) error {
	if convCtx == nil || len(recommendations) == 0 {
		return nil
	}
	payload := builder.RecommendationsWithMetadata(recommendations, convCtx.ConversationId, convCtx.ExchangeId)
	return support.SafeEmitNext(convCtx.Channel, payload)
}

// publishText 下发普通文本片段
func publishText(convCtx *vo.ConversationContext, content string) error {
	if convCtx == nil || strutil.IsBlank(content) {
		return nil
	}
	convCtx.WriteAnswerBuffer(content)
	payload := builder.TextWithMetadata(content, convCtx.ConversationId, convCtx.ExchangeId)
	return support.SafeEmitNext(convCtx.Channel, payload)
}
