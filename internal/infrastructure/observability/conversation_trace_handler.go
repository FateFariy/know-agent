package observability

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/swiftbit/know-agent/common/logx"
	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/callbacks"
	"github.com/swiftbit/know-agent/internal/domain/chat/adapter"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
)

var _ callbacks.Handler = (*ConversationTraceHandler)(nil)

var registerOnce sync.Once

type ConversationTraceHandler struct {
	repo adapter.ChatRepository
}

func NewConversationTraceRecorder(repo adapter.ChatRepository) *ConversationTraceHandler {
	r := &ConversationTraceHandler{repo: repo}
	registerOnce.Do(func() {
		callbacks.AppendGlobalHandlers(r)
	})
	return r
}

// OnStart 实现 callbacks.Handler，将追踪阶段信息落库
func (t *ConversationTraceHandler) OnStart(ctx context.Context, info *callbacks.RunInfo, input any) context.Context {
	trace := vo.TraceFromCtx(ctx)
	if trace == nil {
		logx.Warn("ConversationTraceHandler.OnStart: ConversationTrace 未在上下文中找到")
		return ctx
	}

	stageCode, _ := info.StageCode.(*vo.ConversationTraceStage)
	if stageCode == nil {
		logx.Warn("ConversationTraceHandler.OnStart: StageCode 无效")
		return ctx
	}

	var summaryText string
	var snapshot any
	if stageInput, ok := input.(*vo.StageInput); ok && stageInput != nil {
		summaryText = stageInput.SummaryText
		snapshot = stageInput.Snapshot
	}

	conversationId := trace.ConversationId()
	stage := &entity.ChatExchangeTraceStage{
		ID:             info.StageId,
		ConversationId: conversationId,
		ExchangeId:     trace.ExchangeId(),
		TraceId:        trace.TraceId(),
		StageCode:      stageCode.Code,
		StageName:      stageCode.Name,
		StageOrder:     stageCode.Order,
		StageLevel:     1,
		ExecutionMode:  info.ExecutionMode,
		StageState:     vo.ConversationTraceStageStateRunning,
		StartTime:      utils.Pointer(info.StartTime),
		SummaryText:    utils.Pointer(summaryText),
		SnapshotJson:   utils.Pointer(t.snapshot(snapshot)),
	}
	if err := t.repo.InsertStage(ctx, stage); err != nil {
		logx.Warnf("插入阶段信息失败: conversationId=%s err=%v", conversationId, err)
	}
	return ctx
}

// OnEnd 实现 callbacks.Handler，更新追踪阶段为完成状态
func (t *ConversationTraceHandler) OnEnd(ctx context.Context, info *callbacks.RunInfo, output any) context.Context {
	trace := vo.TraceFromCtx(ctx)
	if trace == nil {
		return ctx
	}
	stageOutput, ok := output.(*vo.StageOutput)
	if !ok || stageOutput == nil {
		return ctx
	}
	t.updateStage(ctx, info.StageId, trace.ConversationId(), info.StartTime,
		vo.ConversationTraceStageStateCompleted, stageOutput.SummaryText, "", stageOutput.Snapshot)
	return ctx
}

// OnError 实现 callbacks.Handler，更新追踪阶段为失败状态
func (t *ConversationTraceHandler) OnError(ctx context.Context, info *callbacks.RunInfo, err error) context.Context {
	trace := vo.TraceFromCtx(ctx)
	if trace == nil {
		return ctx
	}

	var stageErr *vo.StageError
	summaryText, errMsg := "", err.Error()
	if ok := errors.As(err, &stageErr); ok {
		summaryText = stageErr.SummaryText
		if stageErr.Err != nil {
			errMsg = stageErr.Err.Error()
		}
	}
	t.updateStage(ctx, info.StageId, trace.ConversationId(), info.StartTime,
		vo.ConversationTraceStageStateFailed, summaryText, errMsg, nil)
	return ctx
}

// updateStage 更新阶段信息
func (t *ConversationTraceHandler) updateStage(ctx context.Context, stageId int64, conversationId string,
	startTime time.Time, stageState int, summaryText, errMsg string, snapshot any) {
	stage := &entity.ChatExchangeTraceStage{
		ID:           stageId,
		StageState:   stageState,
		SummaryText:  utils.Pointer(summaryText),
		DurationMs:   time.Since(startTime).Milliseconds(),
		ErrorMessage: utils.Pointer(errMsg),
		SnapshotJson: utils.Pointer(t.snapshot(snapshot)),
		EndTime:      utils.Pointer(time.Now()),
	}
	if err := t.repo.UpdateStageById(ctx, stage); err != nil {
		logx.Warnf("更新阶段信息失败: conversationId=%s stageId=%d err=%v", conversationId, stageId, err)
	}
}

// snapshot 获取快照
func (t *ConversationTraceHandler) snapshot(snapshot any) string {
	snapshotJson, _ := json.Marshal(snapshot)
	return string(snapshotJson)
}
