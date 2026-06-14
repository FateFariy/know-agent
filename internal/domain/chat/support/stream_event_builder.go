package support

import (
	"encoding/json"
	"time"

	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
)

// StreamEventBuilder 流式事件构建器, 用于构建 SSE 流式响应中的各类事件 JSON 字符串
type StreamEventBuilder struct {
}

// NewStreamEventBuilder 创建流式事件构建器实例
func NewStreamEventBuilder() *StreamEventBuilder {
	return &StreamEventBuilder{}
}

// Text 生成文本类型事件
func (b *StreamEventBuilder) Text(content string) string {
	return b.TextWithMetadata(content, nil)
}

// TextWithMetadata 生成带元数据的文本类型事件
func (b *StreamEventBuilder) TextWithMetadata(content string, metadata *vo.StreamEventMetadata) string {
	return b.build(b.event("text", content, metadata))
}

// Thinking 生成思考类型事件
func (b *StreamEventBuilder) Thinking(content string) string {
	return b.ThinkingWithMetadata(content, nil)
}

// ThinkingWithMetadata 生成带元数据的思考类型事件
func (b *StreamEventBuilder) ThinkingWithMetadata(content string, metadata *vo.StreamEventMetadata) string {
	return b.build(b.event("thinking", content, metadata))
}

// Status 生成状态类型事件
func (b *StreamEventBuilder) Status(content string) string {
	return b.StatusWithMetadata(content, nil)
}

// StatusWithMetadata 生成带元数据的状态类型事件
func (b *StreamEventBuilder) StatusWithMetadata(content string, metadata *vo.StreamEventMetadata) string {
	return b.build(b.event("status", content, metadata))
}

// Error 生成错误类型事件
func (b *StreamEventBuilder) Error(content string) string {
	return b.ErrorWithMetadata(content, nil)
}

// ErrorWithMetadata 生成带元数据的错误类型事件
func (b *StreamEventBuilder) ErrorWithMetadata(content string, metadata *vo.StreamEventMetadata) string {
	return b.build(b.event("error", content, metadata))
}

// References 生成引用类型事件
func (b *StreamEventBuilder) References(references []*vo.SearchReference) string {
	return b.ReferencesWithMetadata(references, nil)
}

// ReferencesWithMetadata 生成带元数据的引用类型事件
func (b *StreamEventBuilder) ReferencesWithMetadata(references []*vo.SearchReference, metadata *vo.StreamEventMetadata) string {
	count := 0
	if references != nil {
		count = len(references)
	}
	payload := b.event("reference", references, metadata)
	payload["count"] = count
	return b.build(payload)
}

// Recommendations 生成推荐类型事件
func (b *StreamEventBuilder) Recommendations(recommendations []string) string {
	return b.RecommendationsWithMetadata(recommendations, nil)
}

// RecommendationsWithMetadata 生成带元数据的推荐类型事件
func (b *StreamEventBuilder) RecommendationsWithMetadata(recommendations []string, metadata *vo.StreamEventMetadata) string {
	count := 0
	if recommendations != nil {
		count = len(recommendations)
	}
	payload := b.event("recommend", recommendations, metadata)
	payload["count"] = count
	return b.build(payload)
}

// event 构建事件载荷
func (b *StreamEventBuilder) event(eventType string, content any, metadata *vo.StreamEventMetadata) map[string]any {
	payload := map[string]any{
		"type":      eventType,
		"content":   content,
		"timestamp": time.Now().Format(time.RFC3339),
	}
	if metadata != nil {
		if metadata.ConversationId != "" {
			payload["conversationId"] = metadata.ConversationId
		}
		if metadata.ExchangeId > 0 {
			payload["exchangeId"] = metadata.ExchangeId
		}
	}
	return payload
}

// build 将载荷序列化为 JSON 字符串
func (b *StreamEventBuilder) build(payload map[string]any) string {
	data, _ := json.Marshal(payload)
	return string(data)
}
