package index

import (
	"context"
	"encoding/json"
	"time"

	"github.com/swiftbit/know-agent/common/logx"
	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/document/model/enum"
)

// PreparationPhase 准备阶段：加载策略步骤、推进任务状态
type PreparationPhase struct {
	*PhaseDeps
}

func NewPreparationPhase(deps *PhaseDeps) *PreparationPhase {
	return &PreparationPhase{PhaseDeps: deps}
}

func (p *PreparationPhase) Name() string {
	return "准备阶段"
}

func (p *PreparationPhase) Execute(ctx context.Context, buildCtx *BuildContext) error {
	logx.Infof("开始执行索引构建任务，documentId=%d, taskId=%d, planId=%d",
		buildCtx.DocumentId, buildCtx.TaskId, buildCtx.PlanId)

	// 查询策略步骤列表
	pipelineSteps, err := p.Repo.SelectStepListByPlanId(ctx, buildCtx.PlanId)
	if err != nil {
		return err
	}
	buildCtx.PipelineSteps = pipelineSteps
	logx.Infof("索引构建策略步骤读取完成，documentId=%d, taskId=%d, planId=%d, stepCount=%d",
		buildCtx.DocumentId, buildCtx.TaskId, buildCtx.PlanId, len(pipelineSteps))

	// 初始化时间
	buildCtx.BuildStartedTime = time.Now()

	// 事务性推进任务状态到"切块执行中"
	markBuildingTx := func(txCtx context.Context) error {
		if err = p.Repo.UpdateDocumentById(txCtx, &entity.Document{
			ID: buildCtx.Document.ID, IndexStatus: enum.IndexStatusBuilding,
		}); err != nil {
			return err
		}
		if err = p.Repo.UpdateStepExecuteStatus(txCtx, buildCtx.Plan.ID, enum.StrategyExecuteStatusExecuting); err != nil {
			return err
		}
		chunkStartDetail, _ := json.Marshal(map[string]any{"strategySnapshot": buildCtx.Plan.StrategySnapshot})
		chunkStartLog := &entity.DocumentTaskLog{
			TaskId:       buildCtx.TaskId,
			DocumentId:   buildCtx.DocumentId,
			StageType:    enum.TaskStageChunkExecute,
			EventType:    enum.TaskEventStart,
			LogLevel:     enum.LogLevelInfo,
			OperatorType: enum.OperatorTypeSystem,
			Content:      "开始执行切块流水线",
			DetailJson:   string(chunkStartDetail),
		}
		if err = p.Repo.InsertTaskLog(txCtx, chunkStartLog); err != nil {
			return err
		}
		return p.Repo.UpdateTaskById(txCtx, &entity.DocumentTask{
			ID: buildCtx.TaskId, TaskStatus: enum.TaskStatusRunning,
			CurrentStage: enum.TaskStageChunkExecute,
			StartTime:    utils.Pointer(buildCtx.StartTime),
		})
	}
	return p.Repo.Do(ctx, markBuildingTx)
}
