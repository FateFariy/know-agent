package index

import (
	"context"

	"github.com/swiftbit/know-agent/common/logx"
	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/document/model/enum"
	"github.com/swiftbit/know-agent/internal/domain/document/model/vo"
)

type ResumePhase struct {
	repo adapter.DocumentRepository
}

func NewResumePhase(repo adapter.DocumentRepository) *ResumePhase {
	return &ResumePhase{repo: repo}
}

func (r *ResumePhase) Name() string {
	return "恢复阶段"
}

func (p *ResumePhase) Execute(ctx context.Context, buildCtx *Context) error {
	if !buildCtx.ResumeCommittedGraph {
		return nil
	}
	// 列出已冻结的源块（用于断点恢复）
	chunks, err := p.repo.SelectChunks(ctx, map[string]any{
		"document_id": buildCtx.DocumentId,
		"task_id":     buildCtx.TaskId,
		"source_type": enum.ChunkSourceTypeGraphRAG,
	})
	if err != nil {
		return err
	}
	buildCtx.ChildChunks = chunks
	buildCtx.GraphRagBuildResult = p.repairCrossDocumentProjection(ctx, buildCtx.Document, buildCtx.TaskId, buildCtx.GraphRagBuildResult)
	logx.Infof("从已提交 GraphRAG outcome 恢复索引任务: documentId=%d, taskId=%d",
		buildCtx.DocumentId, buildCtx.TaskId)
	return nil
}

// repairCrossDocumentProjection 修复跨文档投影
func (p *ResumePhase) repairCrossDocumentProjection(ctx context.Context, document *entity.Document,
	taskId int64, buildResult *vo.GraphRagBuildResult) *vo.GraphRagBuildResult {
	alreadyActive := document != nil && document.LastIndexTaskId == taskId
	if alreadyActive && buildResult.CrossDocumentIndexOutcome == enum.ComponentOutcomeSuccess {
		return buildResult
	}
	if err := p.crossDocumentIndexer.RebuildAll(ctx, document.ID, taskId); err != nil {
		logx.Warnf("GraphRAG cross-document repair failed: documentId=%d, taskId=%d, message=%v", document.ID, taskId, err)
		return p.graphRagOutcomePolicy.WithCrossDocumentOutcome(buildResult, enum.ComponentOutcomeFailed)
	}
	return p.graphRagOutcomePolicy.WithCrossDocumentOutcome(buildResult, enum.ComponentOutcomeSuccess)
}
