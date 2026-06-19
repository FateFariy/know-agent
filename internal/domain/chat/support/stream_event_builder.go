package support

import (
	"encoding/json"
	"time"

	"github.com/duke-git/lancet/v2/strutil"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
)

// StreamEvent 流式事件结构体
type StreamEvent struct {
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

// Text 生成文本类型事件
func (b *StreamEventBuilder) Text(content string) string {
	return b.TextWithMetadata(content, "", 0)
}

// TextWithMetadata 生成带元数据的文本类型事件
func (b *StreamEventBuilder) TextWithMetadata(content string, conversationId string, exchangeId int64) string {
	return b.build(b.event("text", content, conversationId, exchangeId))
}

// Thinking 生成思考类型事件
func (b *StreamEventBuilder) Thinking(content string) string {
	return b.ThinkingWithMetadata(content, "", 0)
}

// ThinkingWithMetadata 生成带元数据的思考类型事件
func (b *StreamEventBuilder) ThinkingWithMetadata(content string, conversationId string, exchangeId int64) string {
	return b.build(b.event("thinking", content, conversationId, exchangeId))
}

// Status 生成状态类型事件
func (b *StreamEventBuilder) Status(content string) string {
	return b.StatusWithMetadata(content, "", 0)
}

// StatusWithMetadata 生成带元数据的状态类型事件
func (b *StreamEventBuilder) StatusWithMetadata(content string, conversationId string, exchangeId int64) string {
	return b.build(b.event("status", content, conversationId, exchangeId))
}

// Error 生成错误类型事件
func (b *StreamEventBuilder) Error(content string) string {
	return b.ErrorWithMetadata(content, "", 0)
}

// ErrorWithMetadata 生成带元数据的错误类型事件
func (b *StreamEventBuilder) ErrorWithMetadata(content string, conversationId string, exchangeId int64) string {
	return b.build(b.event("error", content, conversationId, exchangeId))
}

// References 生成引用类型事件
func (b *StreamEventBuilder) References(references []*vo.SearchReference) string {
	return b.ReferencesWithMetadata(references, "", 0)
}

// ReferencesWithMetadata 生成带元数据的引用类型事件
func (b *StreamEventBuilder) ReferencesWithMetadata(references []*vo.SearchReference, conversationId string, exchangeId int64) string {
	payload := b.event("reference", references, conversationId, exchangeId)
	payload.Count = utils.Pointer(len(references))
	return b.build(payload)
}

// Recommendations 生成推荐类型事件
func (b *StreamEventBuilder) Recommendations(recommendations []string) string {
	return b.RecommendationsWithMetadata(recommendations, "", 0)
}

// RecommendationsWithMetadata 生成带元数据的推荐类型事件
func (b *StreamEventBuilder) RecommendationsWithMetadata(recommendations []string, conversationId string, exchangeId int64) string {
	payload := b.event("recommend", recommendations, conversationId, exchangeId)
	payload.Count = utils.Pointer(len(recommendations))
	return b.build(payload)
}

// event 构建事件载荷
func (b *StreamEventBuilder) event(eventType string, content any, conversationId string, exchangeId int64) *StreamEvent {
	payload := &StreamEvent{
		Type:      eventType,
		Content:   content,
		Timestamp: time.Now().Format(time.RFC3339),
	}
	payload.ConversationId = strutil.Trim(conversationId)
	payload.ExchangeId = exchangeId
	return payload
}

// build 将载荷序列化为 JSON 字符串
func (b *StreamEventBuilder) build(event *StreamEvent) string {
	data, _ := json.Marshal(event)
	return string(data)
}
