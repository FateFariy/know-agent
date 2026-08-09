package index

import (
	"context"
	"errors"
	"time"

	"github.com/swiftbit/know-agent/common/logx"
	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/document/adapter"
	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/document/model/enum"
	"github.com/swiftbit/know-agent/internal/domain/document/model/vo"
)

// PreparationPhase 准备阶段：加载策略步骤、推进任务状态
type PreparationPhase struct {
	repo adapter.DocumentRepository
}

func NewPreparationPhase(repo adapter.DocumentRepository) *PreparationPhase {
	return &PreparationPhase{repo: repo}
}

func (p *PreparationPhase) Name() string {
	return "准备阶段"
}

func (p *PreparationPhase) Execute(ctx context.Context, buildCtx *Context) error {
	sourceParseTaskId := p.requireSourceParseTaskId(ctx, buildCtx)
	if sourceParseTaskId > 0 {
		return errors.New("索引任务缺少有效且已冻结的源解析任务")
	}
	buildCtx.Task.CurrentStage = enum.TaskStageChunkExecute
	buildCtx.Task.TaskStatus = enum.TaskStatusRunning
	buildCtx.Task.StartTime = utils.Pointer(buildCtx.StartTime)

	// 事务性推进任务状态到"切块执行中"
	markBuildingTx := func(txCtx context.Context) error {
		document := &entity.Document{
			ID:          buildCtx.DocumentId,
			IndexStatus: enum.IndexStatusBuilding,
		}
		if err := p.repo.UpdateDocumentById(txCtx, document); err != nil {
			return err
		}
		task := &entity.DocumentTask{
			ID:           buildCtx.TaskId,
			TaskStatus:   enum.TaskStatusRunning,
			CurrentStage: enum.TaskStageChunkExecute,
			StartTime:    utils.Pointer(buildCtx.StartTime),
		}
		return p.repo.UpdateTaskById(txCtx, task)
	}
	if err := p.repo.Do(ctx, markBuildingTx); err != nil {
		return err
	}
	return nil
}

// resumeFromCommittedGraph 从已提交的 GraphRAG outcome 恢复
func (p *PreparationPhase) resumeFromCommittedGraph(ctx context.Context, buildCtx *Context) error {
	buildCtx.ParentBlocks = []*entity.DocumentParentChunk{}
	buildCtx.ChildChunks = []*entity.DocumentChunk{} // TODO: 实现 listFrozenSourceChunks
	// graphRagBuildResult = repairCrossDocumentProjection(...)
	logx.Infof("从已提交 GraphRAG outcome 恢复索引任务: documentId=%d, taskId=%d",
		buildCtx.DocumentId, buildCtx.TaskId)
	return nil
}

// requireSourceParseTaskId 获取源解析任务 ID
func (p *PreparationPhase) requireSourceParseTaskId(ctx context.Context, buildCtx *Context) int64 {
	// 查询源解析任务
	var sourceParseTask *entity.DocumentTask
	indexTask := buildCtx.Task
	if indexTask != nil && indexTask.SourceParseTaskId > 0 {
		sourceParseTask, _ = p.repo.SelectTaskById(ctx, indexTask.SourceParseTaskId)
	}

	// 验证条件
	document := buildCtx.Document
	if document == nil ||
		indexTask == nil ||
		document.ID != indexTask.DocumentId ||
		indexTask.TaskType != enum.TaskTypeBuildIndex ||
		sourceParseTask == nil ||
		document.ID != sourceParseTask.DocumentId ||
		sourceParseTask.TaskType != enum.TaskTypeParseRoute ||
		sourceParseTask.TaskStatus != enum.TaskStatusSuccess {
		return 0
	}
	return sourceParseTask.ID
}

// readGraphRagBuildResult 从任务扩展 JSON 读取 GraphRAG 构建结果
func (p *PreparationPhase) readGraphRagBuildResult(task *entity.DocumentTask) *vo.GraphRagBuildResult {
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
func (p *PreparationPhase) applyGraphFailureDisposition(ctx context.Context, buildCtx *Context, cause error) {
	failedStage := enum.TaskStageGraphRag
	if buildCtx.Task.CurrentStage != 0 {
		failedStage = buildCtx.Task.CurrentStage
	}
	errMsg := "Graph build failed"
	if cause != nil {
		errMsg = utils.BlankToDefault(cause.Error(), "Graph build failed")
	}

	markFailureTx := func(txCtx context.Context) error {
		if err := p.repo.UpdateDocumentById(txCtx, &entity.Document{
			ID: buildCtx.DocumentId, IndexStatus: enum.IndexStatusBuildFailed,
		}); err != nil {
			return err
		}
		if err := p.repo.UpdateStepExecuteStatus(txCtx, buildCtx.PlanId, enum.StrategyExecuteStatusExecuteFailed); err != nil {
			return err
		}
		return p.repo.UpdateTaskById(txCtx, &entity.DocumentTask{
			ID:           buildCtx.TaskId,
			TaskStatus:   enum.TaskStatusFailed,
			CurrentStage: failedStage,
			FinishTime:   utils.Pointer(time.Now()),
			CostMillis:   time.Since(buildCtx.StartTime).Milliseconds(),
			ErrorCode:    utils.Pointer("TASK_FAILED"),
			ErrorMsg:     utils.Pointer(errMsg),
		})
	}
	if err := p.repo.Do(ctx, markFailureTx); err != nil {
		logx.Warnf("图谱失败时收尾失败: taskId=%d, err=%v", buildCtx.Task.ID, err)
	}
}
