package vo

import (
	"time"

	list "github.com/duke-git/lancet/v2/datastructure/list"
)

type ConversationTrace struct {
	id               int64
	conversationId   string
	exchangeId       int64
	traceId          string
	modelUsageTraces *list.CopyOnWriteList[*ChatModelUsageTrace]
	limitStats       *ChatLimitStats
	stageCode        *ConversationTraceStage
	stageLevel       int
	stageState       ConversationTraceStageState
	parentStageId    int64
	summaryText      string
	executionMode    string
	errorMessage     string
	snapshotJson     string
	durationMs       int64
	ragObservation   any // RAG检索观测数据，使用any避免循环导入
}

type StageHandle struct {
	StageId   int64
	StartTime time.Time
	StageCode *ConversationTraceStage
}

func NewConversationTrace(conversationId string, exchangeId int64, traceId string) *ConversationTrace {
	return &ConversationTrace{
		conversationId:   conversationId,
		exchangeId:       exchangeId,
		traceId:          traceId,
		modelUsageTraces: list.NewCopyOnWriteList([]*ChatModelUsageTrace{}),
		limitStats:       &ChatLimitStats{},
	}
}

func (t *ConversationTrace) AddModelUsageTrace(trace *ChatModelUsageTrace) {
	if trace == nil {
		return
	}
	t.modelUsageTraces.Add(trace)
}

func (t *ConversationTrace) ConversationId() string {
	return t.conversationId
}

// ExchangeId 获取交换ID
func (t *ConversationTrace) ExchangeId() int64 {
	return t.exchangeId
}

// TraceId 获取追踪ID
func (t *ConversationTrace) TraceId() string {
	return t.traceId
}
