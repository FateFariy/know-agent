package index

import (
	"context"
	"errors"

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

// isCommittedGraph 检查图谱是否已提交
func (p *PreparationPhase) isCommittedGraph(result *vo.GraphRagBuildResult) bool {
	return result != nil && result.KgCommitted &&
		result.GraphPersistenceOutcome != "" &&
		result.GraphPersistenceOutcome != vo.GraphPersistenceOutcomeFailed
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
