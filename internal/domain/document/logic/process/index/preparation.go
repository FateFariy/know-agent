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

// PreparationPhase 准备阶段：加载策略步骤、推进任务状态
type PreparationPhase struct {
	deps *PhaseDeps
}

func NewPreparationPhase(deps *PhaseDeps) *PreparationPhase {
	return &PreparationPhase{deps: deps}
}

func (p *PreparationPhase) Name() string {
	return "准备阶段"
}

func (p *PreparationPhase) Execute(ctx context.Context, buildCtx *BuildContext) error {
	logx.Infof("开始执行索引构建任务，documentId=%d, taskId=%d, planId=%d",
		buildCtx.DocumentID, buildCtx.TaskID, buildCtx.PlanID)

	// 查询策略步骤列表
	pipelineSteps, err := p.deps.Repo.SelectStepListByPlanId(ctx, buildCtx.PlanID)
	if err != nil {
		return err
	}
	buildCtx.PipelineSteps = pipelineSteps
	logx.Infof("索引构建策略步骤读取完成，documentId=%d, taskId=%d, planId=%d, stepCount=%d",
		buildCtx.DocumentID, buildCtx.TaskID, buildCtx.PlanID, len(pipelineSteps))

	// 初始化时间
	buildCtx.StartTime = time.Now()
	buildCtx.BuildStartedNanos = time.Now()

	// 事务性推进任务状态到"切块执行中"
	markBuildingTx := func(txCtx context.Context) error {
		if err := p.deps.Repo.UpdateDocumentById(txCtx, &entity.Document{
			ID: buildCtx.Document.ID, IndexStatus: vo.IndexStatusBuilding,
		}); err != nil {
			return err
		}
		if err := p.deps.Repo.UpdateStepExecuteStatus(txCtx, buildCtx.Plan.ID, vo.StrategyExecuteStatusExecuting); err != nil {
			return err
		}
		chunkStartDetail, _ := json.Marshal(map[string]any{"strategySnapshot": buildCtx.Plan.StrategySnapshot})
		chunkStartLog := &entity.DocumentTaskLog{
			TaskId: buildCtx.TaskID, DocumentId: buildCtx.DocumentID,
			StageType: vo.TaskStageChunkExecute, EventType: vo.TaskEventStart,
			LogLevel: vo.LogLevelInfo, OperatorType: vo.OperatorTypeSystem,
			Content: "开始执行切块流水线", DetailJson: string(chunkStartDetail),
		}
		if err := p.deps.Repo.InsertTaskLog(txCtx, chunkStartLog); err != nil {
			return err
		}
		return p.deps.Repo.UpdateTaskById(txCtx, &entity.DocumentTask{
			ID: buildCtx.TaskID, TaskStatus: vo.TaskStatusRunning,
			CurrentStage: vo.TaskStageChunkExecute,
			StartTime:    utils.Pointer(time.Now()),
		})
	}
	return p.deps.Repo.Do(ctx, markBuildingTx)
}
