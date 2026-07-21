package support

import (
	"encoding/json"
	"time"

	"github.com/duke-git/lancet/v2/strutil"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
)

// streamEvent 流式事件结构体
type streamEvent struct {
	Type           string `json:"type"`                     // 事件类型
	Content        any    `json:"content"`                  // 事件内容
	Timestamp      string `json:"timestamp"`                // 时间戳
	ConversationId string `json:"conversationId,omitempty"` // 会话ID（可选）
	ExchangeId     int64  `json:"exchangeId,omitempty"`     // 交换ID（可选）
	Count          *int   `json:"count,omitempty"`          // 数量（可选，使用指针区分是否设置）
}

// StreamEventBuilder 流式事件构建器, 用于构建 SSE 流式响应中的各类事件 JSON 字符串
type StreamEventBuilder struct {
}

// Text 构建文本类型事件
func (b *StreamEventBuilder) Text(content string, conversationId string, exchangeId int64) string {
	return b.build(b.event("text", content, conversationId, exchangeId))
}

// Thinking 构建思考类型事件
func (b *StreamEventBuilder) Thinking(content string, conversationId string, exchangeId int64) string {
	return b.build(b.event("thinking", content, conversationId, exchangeId))
}

// Status 构建状态类型事件
func (b *StreamEventBuilder) Status(content string, conversationId string, exchangeId int64) string {
	return b.build(b.event("status", content, conversationId, exchangeId))
}

// Error 构建错误类型事件
func (b *StreamEventBuilder) Error(content string, conversationId string, exchangeId int64) string {
	return b.build(b.event("error", content, conversationId, exchangeId))
}

// References 构建引用类型事件
func (b *StreamEventBuilder) References(references []*vo.SearchReference, conversationId string, exchangeId int64) string {
	payload := b.event("reference", references, conversationId, exchangeId)
	payload.Count = utils.Pointer(len(references))
	return b.build(payload)
}

// Recommendations 构建推荐类型事件
func (b *StreamEventBuilder) Recommendations(recommendations []string, conversationId string, exchangeId int64) string {
	payload := b.event("recommend", recommendations, conversationId, exchangeId)
	payload.Count = utils.Pointer(len(recommendations))
	return b.build(payload)
}

// Finish 构建完成事件
func (b *StreamEventBuilder) Finish(conversationId string, exchangeId int64) string {
	return b.build(b.event("finish", nil, conversationId, exchangeId))
}

// event 构建事件载荷
func (b *StreamEventBuilder) event(eventType string, content any, conversationId string, exchangeId int64) *streamEvent {
	return &streamEvent{
		Type:           eventType,
		Content:        content,
		Timestamp:      time.Now().Format(time.DateTime),
		ConversationId: strutil.Trim(conversationId),
		ExchangeId:     exchangeId,
	}
}

// build 将载荷序列化为 JSON 字符串
func (b *StreamEventBuilder) build(event *streamEvent) string {
	data, _ := json.Marshal(event)
	return string(data)
}
