package vo

import (
	list "github.com/duke-git/lancet/v2/datastructure/list"
)

type ConversationTrace struct {
	conversationId   string
	exchangeId       int64
	traceId          string
	modelUsageTraces *list.CopyOnWriteList[*ChatModelUsageTrace]
	limitStats       *ChatLimitStats
	StageLevel       int
	ExecutionMode    string
	SummaryText      string
	Snapshot         any
}

type StageHandle struct {
	StageId     int64
	StartTimeMs int64
	StageCode   string
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

func (t *ConversationTrace) StartStage(stageCode ConversationTraceStageCode, executionMode string, summaryText string, snapshot any) StageHandle {

	return t.modelUsageTraces.ToArray()
}
