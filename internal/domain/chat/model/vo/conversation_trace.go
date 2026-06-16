package vo

import (
	"encoding/json"
	"time"

	list "github.com/duke-git/lancet/v2/datastructure/list"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/entity"
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
}

type StageHandle struct {
	StageId     int64
	StartTimeMs int64
	StageCode   *ConversationTraceStage
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

// StartStage 阶段开始
func (t *ConversationTrace) StartStage(stageCode *ConversationTraceStage, executionMode string, summaryText string, snapshot any) *StageHandle {
	t.id = utils.GetSnowflakeNextID()
	t.stageCode = stageCode
	t.stageLevel = 1
	t.executionMode = executionMode
	t.summaryText = summaryText
	t.setSnapshot(snapshot)
	t.stageState = ConversationTraceStageStateRunning
	return &StageHandle{
		StageId:     t.id,
		StartTimeMs: time.Now().UnixMilli(),
		StageCode:   stageCode,
	}
}

// CompleteStage 阶段完成
func (t *ConversationTrace) CompleteStage(stageHandle *StageHandle, summaryText string, snapshot any) {
	t.id = stageHandle.StageId
	t.summaryText = summaryText
	t.setSnapshot(snapshot)
	t.durationMs = time.Now().UnixMilli() - stageHandle.StartTimeMs
	t.stageState = ConversationTraceStageStateCompleted
}

// FailStage 阶段失败
func (t *ConversationTrace) FailStage(stageHandle *StageHandle, summaryText string, error error, snapshot any) {
	t.id = stageHandle.StageId
	t.stageState = ConversationTraceStageStateFailed
	t.summaryText = summaryText
	t.errorMessage = error.Error()
	t.setSnapshot(snapshot)
	t.durationMs = time.Now().UnixMilli() - stageHandle.StartTimeMs
}

// ConvChatExchangeTraceStage 转换为 ChatExchangeTraceStage
func (t *ConversationTrace) ConvChatExchangeTraceStage() *entity.ChatExchangeTraceStage {
	return &entity.ChatExchangeTraceStage{
		ID:             t.id,
		ConversationId: t.conversationId,
		ExchangeId:     t.exchangeId,
		TraceId:        t.traceId,
		StageCode:      t.stageCode.Code,
		StageName:      t.stageCode.Name,
		StageOrder:     t.stageCode.Order,
		StageLevel:     t.stageLevel,
		ParentStageId:  t.parentStageId,
		ExecutionMode:  t.executionMode,
		StageState:     t.stageState,
		DurationMs:     t.durationMs,
		SummaryText:    t.summaryText,
		ErrorMessage:   t.errorMessage,
		SnapshotJson:   t.snapshotJson,
	}
}

// setSnapshot 设置快照
func (t *ConversationTrace) setSnapshot(snapshot any) {
	snapshotJson, _ := json.Marshal(snapshot)
	t.snapshotJson = string(snapshotJson)
}
