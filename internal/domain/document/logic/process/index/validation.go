package index

import (
	"context"
	"encoding/json"
	"time"

	"github.com/swiftbit/know-agent/common/logx"
	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/document/model/enum"
	"github.com/swiftbit/know-agent/internal/domain/document/model/vo"
)

// ValidationPhase 验证阶段：检查任务状态、读取 GraphRAG 检查点
type ValidationPhase struct {
	*PhaseDeps
}

func NewValidationPhase(deps *PhaseDeps) *ValidationPhase {
	return &ValidationPhase{PhaseDeps: deps}
}

func (p *ValidationPhase) Name() string {
	return "验证阶段"
}

func (p *ValidationPhase) Execute(ctx context.Context, buildCtx *BuildContext) error {
	// 前置检查：如果任务已成功或失败，跳过重复执行
	if buildCtx.Task.TaskStatus == enum.TaskStatusSuccess {
		logx.Infof("索引构建任务已成功，跳过重复执行，documentId=%d, taskId=%d, planId=%d",
			buildCtx.DocumentId, buildCtx.TaskId, buildCtx.PlanId)
		return nil // 已完成，直接返回
	}
	if buildCtx.Task.TaskStatus == enum.TaskStatusFailed {
		logx.Infof("索引构建任务已失败，跳过重复执行，documentId=%d, taskId=%d, planId=%d",
			buildCtx.DocumentId, buildCtx.TaskId, buildCtx.PlanId)
		return nil // 已失败，直接返回
	}

	// 读取已有的 GraphRAG 构建结果（用于断点恢复）
	graphRagBuildResult := p.readGraphRagBuildResult(buildCtx.Task)
	buildCtx.GraphRagBuildResult = graphRagBuildResult

	// 检查是否需要直接失败
	if graphRagBuildResult != nil && graphRagBuildResult.OuterTaskDisposition == vo.OuterTaskDispositionFailIndexTask {
		p.applyGraphFailureDisposition(ctx, buildCtx, nil)
	}

	return nil
}

// readGraphRagBuildResult 从任务扩展 JSON 读取 GraphRAG 构建结果
func (p *ValidationPhase) readGraphRagBuildResult(task *entity.DocumentTask) *vo.GraphRagBuildResult {
	if task == nil || task.ExtJson == "" {
		return nil
	}
	var wrapper struct {
		GraphRagBuild *vo.GraphRagBuildResult `json:"graphRagBuild"`
	}
	if err := json.Unmarshal([]byte(task.ExtJson), &wrapper); err != nil {
		logx.Warnf("Ignoring unreadable GraphRAG outcome checkpoint: taskId=%d, message=%v", task.ID, err)
		return nil
	}

	return wrapper.GraphRagBuild
}

// applyGraphFailureDisposition 应用图谱失败处置
func (p *ValidationPhase) applyGraphFailureDisposition(ctx context.Context, buildCtx *BuildContext, cause error) {
	failedStage := enum.TaskStageGraphRag
	if buildCtx.Task.CurrentStage != 0 {
		failedStage = buildCtx.Task.CurrentStage
	}
	errMsg := "Graph build failed"
	if cause != nil {
		errMsg = utils.BlankToDefault(cause.Error(), "Graph build failed")
	}

	markFailureTx := func(txCtx context.Context) error {
		if err := p.Repo.UpdateDocumentById(txCtx, &entity.Document{
			ID: buildCtx.Document.ID, IndexStatus: enum.IndexStatusBuildFailed,
		}); err != nil {
			return err
		}
		if err := p.Repo.UpdateStepExecuteStatus(txCtx, buildCtx.PlanId, enum.StrategyExecuteStatusExecuteFailed); err != nil {
			return err
		}
		return p.Repo.UpdateTaskById(txCtx, &entity.DocumentTask{
			ID: buildCtx.Task.ID, TaskStatus: enum.TaskStatusFailed, CurrentStage: failedStage,
			FinishTime: utils.Pointer(time.Now()), CostMillis: time.Since(buildCtx.StartTime).Milliseconds(),
			ErrorCode: utils.Pointer("TASK_FAILED"), ErrorMsg: utils.Pointer(errMsg),
		})
	}
	if err := p.Repo.Do(ctx, markFailureTx); err != nil {
		logx.Warnf("图谱失败时收尾失败: taskId=%d, err=%v", buildCtx.Task.ID, err)
	}
}
