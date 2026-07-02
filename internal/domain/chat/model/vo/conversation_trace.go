package vo

import (
	"encoding/json"
	"reflect"
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
		StageId:   t.id,
		StartTime: time.Now(),
		StageCode: stageCode,
	}
}

// CompleteStage 阶段完成
func (t *ConversationTrace) CompleteStage(stageHandle *StageHandle, summaryText string, snapshot any) {
	t.id = stageHandle.StageId
	t.summaryText = summaryText
	t.setSnapshot(snapshot)
	t.durationMs = time.Since(stageHandle.StartTime).Milliseconds()
	t.stageState = ConversationTraceStageStateCompleted
}

// FailStage 阶段失败
func (t *ConversationTrace) FailStage(stageHandle *StageHandle, summaryText string, error error, snapshot any) {
	t.id = stageHandle.StageId
	t.stageState = ConversationTraceStageStateFailed
	t.summaryText = summaryText
	t.errorMessage = error.Error()
	t.setSnapshot(snapshot)
	t.durationMs = time.Since(stageHandle.StartTime).Milliseconds()
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

// GetExchangeId 获取交换ID
func (t *ConversationTrace) GetExchangeId() int64 {
	return t.exchangeId
}

// GetTraceId 获取追踪ID
func (t *ConversationTrace) GetTraceId() string {
	return t.traceId
}

// RecordChannelExecutions 记录渠道执行观测数据
// 使用反射来追加渠道执行数据，避免循环导入
func (t *ConversationTrace) RecordChannelExecutions(executions any) {
	if executions == nil || reflect.ValueOf(executions).IsNil() {
		return
	}

	// 获取ragObservation字段的反射值
	rv := reflect.ValueOf(t).Elem()
	field := rv.FieldByName("ragObservation")

	if !field.IsValid() || !field.CanSet() {
		return
	}

	// 如果ragObservation为nil，创建一个map来存储数据
	if field.IsNil() {
		obs := make(map[string][]any)
		obs["ChannelExecutions"] = []any{executions}
		field.Set(reflect.ValueOf(obs))
	} else {
		// 追加到现有数据
		obs := field.Interface().(map[string][]any)
		obs["ChannelExecutions"] = append(obs["ChannelExecutions"], executions)
	}
}

// RecordRetrievalResults 记录检索结果观测数据
// 使用反射来追加检索结果数据，避免循环导入
func (t *ConversationTrace) RecordRetrievalResults(results any) {
	if results == nil || reflect.ValueOf(results).IsNil() {
		return
	}

	// 获取ragObservation字段的反射值
	rv := reflect.ValueOf(t).Elem()
	field := rv.FieldByName("ragObservation")

	if !field.IsValid() || !field.CanSet() {
		return
	}

	// 如果ragObservation为nil，创建一个map来存储数据
	if field.IsNil() {
		obs := make(map[string][]any)
		obs["RetrievalResults"] = []any{results}
		field.Set(reflect.ValueOf(obs))
	} else {
		// 追加到现有数据
		obs := field.Interface().(map[string][]any)
		obs["RetrievalResults"] = append(obs["RetrievalResults"], results)
	}
}

// GetRagObservation 获取RAG观测数据
func (t *ConversationTrace) GetRagObservation() any {
	return t.ragObservation
}
