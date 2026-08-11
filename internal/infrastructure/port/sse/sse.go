package sse

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/duke-git/lancet/v2/strutil"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/chat/adapter"
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

type Sink struct {
	w io.Writer
	f http.Flusher
	//m *observability.Metrics
}

var _ adapter.Sink = (*Sink)(nil)

func NewSink(w io.Writer) (*Sink, error) {
	f, ok := w.(http.Flusher)
	if !ok {
		return nil, fmt.Errorf("api: response writer does not support flushing")
	}
	return &Sink{w: w, f: f}, nil
}

// Text 发送文本类型事件
func (s *Sink) Text(content string, conversationId string, exchangeId int64) error {
	return s.WriteFrame(s.newEvent("text", content, conversationId, exchangeId))
}

// Thinking 发送思考类型事件
func (s *Sink) Thinking(content string, conversationId string, exchangeId int64) error {
	return s.WriteFrame(s.newEvent("thinking", content, conversationId, exchangeId))
}

// Status 发送状态类型事件
func (s *Sink) Status(content string, conversationId string, exchangeId int64) error {
	return s.WriteFrame(s.newEvent("status", content, conversationId, exchangeId))
}

// Error 发送错误类型事件
func (s *Sink) Error(content string, conversationId string, exchangeId int64) error {
	return s.WriteFrame(s.newEvent("error", content, conversationId, exchangeId))
}

// References 发送引用类型事件
func (s *Sink) References(references []*vo.SearchReference, conversationId string, exchangeId int64) error {
	payload := s.newEvent("reference", references, conversationId, exchangeId)
	payload.Count = utils.Pointer(len(references))
	return s.WriteFrame(payload)
}

// Recommendations 发送推荐类型事件
func (s *Sink) Recommendations(recommendations []string, conversationId string, exchangeId int64) error {
	payload := s.newEvent("recommend", recommendations, conversationId, exchangeId)
	payload.Count = utils.Pointer(len(recommendations))
	return s.WriteFrame(payload)
}

// Finish 构建完成事件
func (s *Sink) Finish(conversationId string, exchangeId int64) error {
	return s.WriteFrame(s.newEvent("finish", nil, conversationId, exchangeId))
}

// newEvent 构建事件载荷
func (s *Sink) newEvent(eventType string, content any, conversationId string, exchangeId int64) *streamEvent {
	return &streamEvent{
		Type:           eventType,
		Content:        content,
		Timestamp:      time.Now().Format("2006-01-02 15:04:05.000"),
		ConversationId: strutil.Trim(conversationId),
		ExchangeId:     exchangeId,
	}
}

// WriteFrame 写入事件帧
func (s *Sink) WriteFrame(event *streamEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	if _, err = fmt.Fprint(s.w, string(data)); err != nil {
		return err
	}
	s.f.Flush()
	return nil
}
