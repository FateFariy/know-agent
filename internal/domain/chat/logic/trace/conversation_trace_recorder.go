package trace

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/zeromicro/go-zero/core/logx"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/chat/adapter"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
)

type ConversationTraceRecorder struct {
	repo adapter.ChatRepository
}

func NewConversationTraceRecorder(repo adapter.ChatRepository) *ConversationTraceRecorder {
	return &ConversationTraceRecorder{
		repo: repo,
	}
}

// StartStage 阶段开始
func (t *ConversationTraceRecorder) StartStage(ctx context.Context, trace *vo.ConversationTrace, stageCode *vo.ConversationTraceStage,
	executionMode, summaryText string, snapshot any) (*vo.StageHandle, error) {
	if trace == nil {
		return nil, fmt.Errorf("invaild value")
	}
	conversationId := trace.ConversationId()
	stage := &entity.ChatExchangeTraceStage{
		ID:             utils.GetSnowflakeNextID(),
		ConversationId: conversationId,
		ExchangeId:     trace.ExchangeId(),
		TraceId:        trace.TraceId(),
		StageCode:      stageCode.Code,
		StageName:      stageCode.Name,
		StageOrder:     stageCode.Order,
		StageLevel:     1,
		ExecutionMode:  executionMode,
		StageState:     vo.ConversationTraceStageStateRunning,
		StartTime:      utils.Pointer(time.Now()),
		SummaryText:    utils.Pointer(summaryText),
		SnapshotJson:   utils.Pointer(t.snapshot(snapshot)),
	}
	if err := t.repo.InsertStage(ctx, stage); err != nil {
		logx.Alert(fmt.Sprintf("插入阶段信息失败: conversationId=%s err=%v", conversationId, err))
		return nil, err
	}
	return &vo.StageHandle{
		StageId:        stage.ID,
		ConversationId: conversationId,
		StartTime:      time.Now(),
		StageCode:      stageCode,
	}, nil
}

// CompleteStage 阶段完成
func (t *ConversationTraceRecorder) CompleteStage(ctx context.Context, stageHandle *vo.StageHandle, summaryText string, snapshot any) error {
	return t.updateStage(ctx, stageHandle, vo.ConversationTraceStageStateCompleted, summaryText, "", snapshot)
}

// FailStage 阶段失败
func (t *ConversationTraceRecorder) FailStage(ctx context.Context, stageHandle *vo.StageHandle, summaryText string, err error, snapshot any) error {
	return t.updateStage(ctx, stageHandle, vo.ConversationTraceStageStateFailed, summaryText, err.Error(), snapshot)
}

// updateStage 更新阶段信息
func (t *ConversationTraceRecorder) updateStage(ctx context.Context, stageHandle *vo.StageHandle,
	stageState int, summaryText, errMsg string, snapshot any) error {
	if stageHandle == nil {
		return nil
	}
	stage := &entity.ChatExchangeTraceStage{
		ID:           stageHandle.StageId,
		StageState:   stageState,
		SummaryText:  utils.Pointer(summaryText),
		DurationMs:   time.Since(stageHandle.StartTime).Milliseconds(),
		ErrorMessage: utils.Pointer(errMsg),
		SnapshotJson: utils.Pointer(t.snapshot(snapshot)),
		EndTime:      utils.Pointer(time.Now()),
	}
	if err := t.repo.UpdateStageById(ctx, stage); err != nil {
		logx.Alert(fmt.Sprintf("更新阶段信息失败: conversationId=%s err=%v", stageHandle.ConversationId, err))
		return err
	}
	return nil
}

// snapshot 获取快照
func (t *ConversationTraceRecorder) snapshot(snapshot any) string {
	snapshotJson, _ := json.Marshal(snapshot)
	return string(snapshotJson)
}
