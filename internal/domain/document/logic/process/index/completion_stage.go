package index

import (
	"context"
	"encoding/json"
	"time"

	"github.com/swiftbit/know-agent/common/logx"
	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/document/adapter"
	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/document/model/enum"
)

// CompletionStage 完成阶段：事务性更新任务/方案/文档状态
type CompletionStage struct {
	repo adapter.DocumentRepository
}

func NewCompletionStage(repo adapter.DocumentRepository) *CompletionStage {
	return &CompletionStage{repo: repo}
}

func (p *CompletionStage) Name() string {
	return "完成阶段"
}

func (p *CompletionStage) Execute(ctx context.Context, buildCtx *Context) error {
	totalCostMillis := time.Since(buildCtx.BuildStartedTime).Milliseconds()

	// 事务性最终状态更新
	finalizeTx := func(txCtx context.Context) error {
		// 任务阶段推进到"存储完成"
		if err := p.repo.UpdateTaskById(txCtx, &entity.DocumentTask{
			ID: buildCtx.TaskId, CurrentStage: enum.TaskStageStoreComplete,
		}); err != nil {
			return err
		}
		// 方案状态标记为已执行
		if err := p.repo.UpdatePlanById(txCtx, &entity.DocumentStrategyPlan{
			ID:         buildCtx.PlanId,
			PlanStatus: enum.PlanStatusExecuted,
		}); err != nil {
			return err
		}
		// 文档索引状态更新为构建成功
		if err := p.repo.UpdateDocumentById(txCtx, &entity.Document{
			ID:              buildCtx.DocumentId,
			IndexStatus:     enum.IndexStatusBuildSuccess,
			LastIndexTaskId: buildCtx.TaskId,
		}); err != nil {
			return err
		}
		// 写入成功耗时/统计
		completeTask := &entity.DocumentTask{
			ID:           buildCtx.Task.ID,
			TaskStatus:   enum.TaskStatusSuccess,
			CurrentStage: enum.TaskStageStoreComplete,
			FinishTime:   utils.Pointer(time.Now()),
			CostMillis:   time.Since(buildCtx.StartTime).Milliseconds(),
			ErrorCode:    utils.Pointer(""),
			ErrorMsg:     utils.Pointer(""),
		}
		if err := p.repo.UpdateTaskById(ctx, completeTask); err != nil {
			return err
		}
		// 索引构建完成日志
		buildCompleteDetail, _ := json.Marshal(map[string]any{
			"parentBlockCount":     len(buildCtx.ParentChunks),
			"chunkCount":           len(buildCtx.ChildChunks),
			"graphTypedChunkCount": len(buildCtx.GraphTypedChunkList),
			"costMillis":           totalCostMillis,
		})
		buildCompleteLog := &entity.DocumentTaskLog{
			TaskId:       buildCtx.TaskId,
			DocumentId:   buildCtx.DocumentId,
			StageType:    enum.TaskStageStoreComplete,
			EventType:    enum.TaskEventComplete,
			LogLevel:     enum.LogLevelInfo,
			OperatorType: enum.OperatorTypeSystem,
			Content:      "索引构建完成",
			DetailJson:   string(buildCompleteDetail),
		}
		_ = p.repo.InsertTaskLog(txCtx, buildCompleteLog)

		return nil
	}
	if err := p.repo.Do(ctx, finalizeTx); err != nil {
		return err
	}

	logx.Infof("索引构建任务执行完成，documentId=%d, taskId=%d, planId=%d, parentCount=%d, chunkCount=%d, costMillis=%d",
		buildCtx.DocumentId, buildCtx.TaskId, buildCtx.PlanId,
		len(buildCtx.ParentChunks), len(buildCtx.ChildChunks), totalCostMillis)
	return nil
}
