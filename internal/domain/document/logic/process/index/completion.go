package index

import (
	"context"
	"encoding/json"
	"time"

	"github.com/swiftbit/know-agent/common/logx"
	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/document/model/vo"
)

// CompletionPhase 完成阶段：事务性更新任务/方案/文档状态
type CompletionPhase struct {
	deps *PhaseDeps
}

func NewCompletionPhase(deps *PhaseDeps) *CompletionPhase {
	return &CompletionPhase{deps: deps}
}

func (p *CompletionPhase) Name() string {
	return "完成阶段"
}

func (p *CompletionPhase) Execute(ctx context.Context, buildCtx *BuildContext) error {
	buildCtx.TotalCostMillis = time.Since(buildCtx.BuildStartedNanos).Milliseconds()

	// 事务性最终状态更新
	finalizeTx := func(txCtx context.Context) error {
		// 任务阶段推进到"存储完成"
		if err := p.deps.Repo.UpdateTaskById(txCtx, &entity.DocumentTask{
			ID: buildCtx.TaskID, CurrentStage: vo.TaskStageStoreComplete,
		}); err != nil {
			return err
		}
		// 方案状态标记为已执行
		if err := p.deps.Repo.UpdatePlanById(txCtx, &entity.DocumentStrategyPlan{
			ID: buildCtx.PlanID, PlanStatus: vo.PlanStatusExecuted,
		}); err != nil {
			return err
		}
		// 文档索引状态更新为构建成功
		if err := p.deps.Repo.UpdateDocumentById(txCtx, &entity.Document{
			ID: buildCtx.DocumentID, IndexStatus: vo.IndexStatusBuildSuccess,
			LastIndexTaskId: buildCtx.TaskID,
		}); err != nil {
			return err
		}
		// 写入成功耗时/统计
		if err := p.finishTaskSuccess(txCtx, buildCtx); err != nil {
			return err
		}
		// 索引构建完成日志
		buildCompleteDetail, _ := json.Marshal(map[string]any{
			"parentBlockCount":     len(buildCtx.ParentBlocks),
			"chunkCount":           len(buildCtx.ChildChunks),
			"graphTypedChunkCount": len(buildCtx.GraphTypedChunkList),
			"costMillis":           buildCtx.TotalCostMillis,
		})
		buildCompleteLog := &entity.DocumentTaskLog{
			TaskId: buildCtx.TaskID, DocumentId: buildCtx.DocumentID,
			StageType: vo.TaskStageStoreComplete, EventType: vo.TaskEventComplete,
			LogLevel: vo.LogLevelInfo, OperatorType: vo.OperatorTypeSystem,
			Content: "索引构建完成", DetailJson: string(buildCompleteDetail),
		}
		return p.deps.Repo.InsertTaskLog(txCtx, buildCompleteLog)
	}
	if err := p.deps.Repo.Do(ctx, finalizeTx); err != nil {
		return err
	}

	logx.Infof("索引构建任务执行完成，documentId=%d, taskId=%d, planId=%d, parentCount=%d, chunkCount=%d, costMillis=%d",
		buildCtx.DocumentID, buildCtx.TaskID, buildCtx.PlanID,
		len(buildCtx.ParentBlocks), len(buildCtx.ChildChunks), buildCtx.TotalCostMillis)
	return nil
}

// finishTaskSuccess 将任务标记为成功状态
func (p *CompletionPhase) finishTaskSuccess(ctx context.Context, buildCtx *BuildContext) error {
	task := buildCtx.Task
	// 检查任务是否已开始
	if task.StartTime == nil {
		task.StartTime = utils.Pointer(buildCtx.StartTime)
	}

	return p.deps.Repo.UpdateTaskById(ctx, &entity.DocumentTask{
		ID:           task.ID,
		TaskStatus:   vo.TaskStatusSuccess,
		CurrentStage: vo.TaskStageStoreComplete,
		FinishTime:   utils.Pointer(time.Now()),
		CostMillis:   time.Since(*task.StartTime).Milliseconds(),
		ErrorCode:    utils.Pointer(""),
		ErrorMsg:     utils.Pointer(""),
	})
}
