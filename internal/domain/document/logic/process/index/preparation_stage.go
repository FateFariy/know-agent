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
	errorx "github.com/swiftbit/know-agent/internal/error"
)

// PreparationStage 准备阶段：加载策略步骤、推进任务状态
type PreparationStage struct {
	repo    adapter.DocumentRepository
	storage adapter.Storage
}

func NewPreparationStage(repo adapter.DocumentRepository, storage adapter.Storage) *PreparationStage {
	return &PreparationStage{
		repo:    repo,
		storage: storage,
	}
}

func (p *PreparationStage) Name() string {
	return "准备阶段"
}

func (p *PreparationStage) Execute(ctx context.Context, buildCtx *Context) error {
	// 读取已有的 GraphRAG 构建结果（用于断点恢复）
	buildCtx.GraphRagBuildResult = buildCtx.Task.ReadGraphRagBuildResult()

	// 检查是否需要直接失败
	buildCtx.ResumeCommittedGraph = buildCtx.GraphRagBuildResult.IsCommittedGraph()

	if buildCtx.ResumeCommittedGraph {
		return errorx.ErrGraphRagBuildFailed
	}

	sourceParseTaskId := p.requireSourceParseTaskId(ctx, buildCtx)
	if sourceParseTaskId > 0 {
		return errors.New("索引任务缺少有效且已冻结的源解析任务")
	}

	// 下载原始文件内容
	rawFileBytes, err := p.storage.DownloadObject(ctx, buildCtx.Document.ObjectName)
	if err != nil {
		return err
	}
	buildCtx.RawFileBytes = rawFileBytes

	// 推进任务状态到"切块执行中"
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
	return p.repo.Do(ctx, markBuildingTx)
}

// requireSourceParseTaskId 获取源解析任务 ID
func (p *PreparationStage) requireSourceParseTaskId(ctx context.Context, buildCtx *Context) int64 {
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

// applyGraphFailureDisposition 应用图谱失败处置
func (p *PreparationStage) applyGraphFailureDisposition(ctx context.Context, buildCtx *Context, cause error) {
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
