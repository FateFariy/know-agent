package vo

import (
	list "github.com/duke-git/lancet/v2/datastructure/list"
)

type ConversationTrace struct {
	id               int64
	conversationId   string
	exchangeId       int64
	traceId          string
	modelUsageTraces *list.CopyOnWriteList[*ChatModelUsageTrace]
}

func NewConversationTrace(conversationId string, exchangeId int64, traceId string) *ConversationTrace {
	return &ConversationTrace{
		conversationId:   conversationId,
		exchangeId:       exchangeId,
		traceId:          traceId,
		modelUsageTraces: list.NewCopyOnWriteList([]*ChatModelUsageTrace{}),
	}
}

// AddModelUsageTrace 添加模型调用轨迹
func (t *ConversationTrace) AddModelUsageTrace(trace *ChatModelUsageTrace) {
	if trace == nil {
		return
	}
	t.modelUsageTraces.Add(trace)
}

// SnapshotModelUsageTraces 获取模型调用轨迹的快照
func (t *ConversationTrace) SnapshotModelUsageTraces() []*ChatModelUsageTrace {
	if t == nil {
		return nil
	}
	return t.modelUsageTraces.SubList(0, t.modelUsageTraces.Size())
}

// ConversationId 获取对话ID
func (t *ConversationTrace) ConversationId() string {
	if t == nil {
		return ""
	}
	return t.conversationId
}

// ExchangeId 获取交换ID
func (t *ConversationTrace) ExchangeId() int64 {
	if t == nil {
		return 0
	}
	return t.exchangeId
}

// TraceId 获取追踪ID
func (t *ConversationTrace) TraceId() string {
	if t == nil {
		return ""
	}
	return t.traceId
}
