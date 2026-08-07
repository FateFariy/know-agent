package index

import (
	"context"
	"encoding/json"
	"slices"
	"time"

	"github.com/duke-git/lancet/v2/slice"
	"github.com/duke-git/lancet/v2/strutil"

	"github.com/swiftbit/know-agent/common/logx"
	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/document/model/enum"
	"github.com/swiftbit/know-agent/internal/domain/document/model/vo"
)

// ChunkingPhase 切块阶段：执行切块流水线、构建父子块实体、持久化
type ChunkingPhase struct {
	*PhaseDeps
}

func NewChunkingPhase(deps *PhaseDeps) *ChunkingPhase {
	return &ChunkingPhase{PhaseDeps: deps}
}

func (p *ChunkingPhase) Name() string {
	return "切块阶段"
}

func (p *ChunkingPhase) Execute(ctx context.Context, buildCtx *BuildContext) error {
	// 检查是否需要从已提交 GraphRAG 结果恢复
	buildCtx.ResumeCommittedGraph = p.isCommittedGraph(buildCtx.GraphRagBuildResult)

	if buildCtx.ResumeCommittedGraph {
		return p.resumeFromCommittedGraph(ctx, buildCtx)
	}
	return p.executeChunkingPipeline(ctx, buildCtx)
}

// isCommittedGraph 检查图谱是否已提交
func (p *ChunkingPhase) isCommittedGraph(result *vo.GraphRagBuildResult) bool {
	return result != nil && result.KgCommitted &&
		result.GraphPersistenceOutcome != "" &&
		result.GraphPersistenceOutcome != vo.GraphPersistenceOutcomeFailed
}

// resumeFromCommittedGraph 从已提交的 GraphRAG outcome 恢复
func (p *ChunkingPhase) resumeFromCommittedGraph(ctx context.Context, buildCtx *BuildContext) error {
	buildCtx.ParentBlocks = []*entity.DocumentParentBlock{}
	buildCtx.ChildChunks = []*entity.DocumentChunk{} // TODO: 实现 listFrozenSourceChunks
	// graphRagBuildResult = repairCrossDocumentProjection(...)
	logx.Infof("从已提交 GraphRAG outcome 恢复索引任务: documentId=%d, taskId=%d",
		buildCtx.DocumentId, buildCtx.TaskId)
	return nil
}

// executeChunkingPipeline 执行切块流水线
func (p *ChunkingPhase) executeChunkingPipeline(ctx context.Context, buildCtx *BuildContext) error {
	// 下载解析文本
	parsedText, err := p.Port.DownloadText(ctx, buildCtx.Document.ParseTextPath)
	if err != nil {
		return err
	}

	// 按步骤执行切块流水线
	chunkStartedNanos := time.Now()
	parentCandidates, err := p.Coordinator.BuildParentBlocks(ctx, buildCtx.Document, buildCtx.PipelineSteps, parsedText)
	if err != nil {
		return err
	}
	buildCtx.ParentCandidates = parentCandidates
	costMillis := time.Since(chunkStartedNanos).Milliseconds()
	logx.Infof("切块流水线执行完成，documentId=%d, taskId=%d, parentCount=%d, childCount=%d, costMillis=%d",
		buildCtx.DocumentId, buildCtx.TaskId, len(parentCandidates), p.countChildCandidates(parentCandidates), costMillis)

	// 事务性标记切块完成
	markChunkCompleteTx := func(txCtx context.Context) error {
		if err = p.Repo.UpdateStepExecuteStatus(txCtx, buildCtx.Plan.ID, enum.StrategyExecuteStatusExecuteSuccess); err != nil {
			return err
		}
		chunkEndDetail, _ := json.Marshal(map[string]any{
			"parentCount": len(parentCandidates),
			"childCount":  p.countChildCandidates(parentCandidates),
			"costMillis":  costMillis,
		})
		chunkEndLog := &entity.DocumentTaskLog{
			TaskId: buildCtx.TaskId, DocumentId: buildCtx.DocumentId,
			StageType: enum.TaskStageChunkExecute, EventType: enum.TaskEventComplete,
			LogLevel: enum.LogLevelInfo, OperatorType: enum.OperatorTypeSystem,
			Content: "切块执行完成", DetailJson: string(chunkEndDetail),
		}
		if err = p.Repo.InsertTaskLog(txCtx, chunkEndLog); err != nil {
			return err
		}
		return p.Repo.UpdateTaskById(txCtx, &entity.DocumentTask{
			ID: buildCtx.TaskId, CurrentStage: enum.TaskStageChunkPostProcess,
		})
	}
	if err = p.Repo.Do(ctx, markChunkCompleteTx); err != nil {
		return err
	}

	// 清理候选并构造持久化实体
	processStartedNanos := time.Now()
	finalCandidates := p.cleanupParentCandidates(parentCandidates)
	parentBlocks, childChunks := p.buildParentChildEntities(buildCtx.DocumentId, buildCtx.TaskId, buildCtx.PlanId, finalCandidates)
	buildCtx.ParentBlocks = parentBlocks
	buildCtx.ChildChunks = childChunks
	costMillis := time.Since(processStartedNanos).Milliseconds()
	logx.Infof("切块后处理完成，documentId=%d, taskId=%d, parentCount=%d, childCount=%d, costMillis=%d",
		buildCtx.DocumentId, buildCtx.TaskId, len(finalCandidates), p.countChildCandidates(finalCandidates), costMillis)

	// 事务性批量落库
	persistBlocksTx := func(txCtx context.Context) error {
		if err = p.Repo.InsertParentBlockBatch(txCtx, parentBlocks); err != nil {
			return err
		}
		if err = p.Repo.InsertChunkBatch(txCtx, childChunks); err != nil {
			return err
		}
		chunkPostDetail, _ := json.Marshal(map[string]any{
			"parentCount": len(finalCandidates),
			"childCount":  p.countChildCandidates(finalCandidates),
			"costMillis":  costMillis,
		})
		chunkPostLog := &entity.DocumentTaskLog{
			TaskId: buildCtx.TaskId, DocumentId: buildCtx.DocumentId,
			StageType: enum.TaskStageChunkPostProcess, EventType: enum.TaskEventComplete,
			LogLevel: enum.LogLevelInfo, OperatorType: enum.OperatorTypeSystem,
			Content: "切块后处理完成", DetailJson: string(chunkPostDetail),
		}
		if err = p.Repo.InsertTaskLog(txCtx, chunkPostLog); err != nil {
			return err
		}
		return p.Repo.UpdateTaskById(txCtx, &entity.DocumentTask{
			ID: buildCtx.TaskId, CurrentStage: enum.TaskStageVectorize,
		})
	}
	return p.Repo.Do(ctx, persistBlocksTx)
}

// countChildCandidates 计算子块候选数
func (p *ChunkingPhase) countChildCandidates(parentBlockCandidateList []*vo.ParentBlockCandidate) int {
	count := 0
	for _, candidate := range parentBlockCandidateList {
		for _, child := range candidate.ChildChunks {
			if child != nil && strutil.IsNotBlank(child.Text) {
				count++
			}
		}
	}
	return count
}

// cleanupParentCandidates 过滤"文本为空"或"无子块"的父块候选
func (p *ChunkingPhase) cleanupParentCandidates(candidates []*vo.ParentBlockCandidate) []*vo.ParentBlockCandidate {
	return slice.Filter(candidates, func(_ int, item *vo.ParentBlockCandidate) bool {
		fn := func(child *vo.ChunkCandidate) bool { return child != nil && strutil.IsNotBlank(child.Text) }
		return item != nil && strutil.IsNotBlank(item.Text) && slices.ContainsFunc(item.ChildChunks, fn)
	})
}

// buildParentChildEntities 将父块候选转换为可落库的"父块实体 + 子块实体"双列表
func (p *ChunkingPhase) buildParentChildEntities(documentId, taskId, planId int64,
	candidates []*vo.ParentBlockCandidate) ([]*entity.DocumentParentBlock, []*entity.DocumentChunk) {

	parentBlocks := make([]*entity.DocumentParentBlock, 0, len(candidates))
	chunks := make([]*entity.DocumentChunk, 0)

	globalChunkNo := 0
	for parentIdx, candidate := range candidates {
		parentBlock := &entity.DocumentParentBlock{
			ID:                utils.GetSnowflakeNextID(),
			DocumentId:        documentId,
			TaskId:            taskId,
			PlanId:            planId,
			ParentNo:          parentIdx + 1,
			SourceType:        candidate.SourceType,
			SectionPath:       candidate.SectionPath,
			StructureNodeId:   candidate.StructureNodeId,
			StructureNodeType: candidate.StructureNodeType,
			CanonicalPath:     candidate.CanonicalPath,
			ItemIndex:         candidate.ItemIndex,
			ParentText:        candidate.Text,
			CharCount:         utils.Len(candidate.Text),
			TokenCount:        utils.EstimateTokens(candidate.Text),
			StartChunkNo:      globalChunkNo,
		}

		for _, child := range candidate.ChildChunks {
			if child != nil && strutil.IsNotBlank(child.Text) {
				globalChunkNo++
				chunks = append(chunks, &entity.DocumentChunk{
					ID:                utils.GetSnowflakeNextID(),
					DocumentId:        documentId,
					TaskId:            taskId,
					PlanId:            planId,
					ParentBlockId:     parentBlock.ID,
					ChunkNo:           globalChunkNo,
					SourceType:        child.SourceType,
					SectionPath:       utils.BlankToDefault(child.SectionPath, candidate.SectionPath),
					StructureNodeId:   child.StructureNodeId,
					StructureNodeType: child.StructureNodeType,
					CanonicalPath:     child.CanonicalPath,
					ItemIndex:         child.ItemIndex,
					ChunkText:         child.Text,
					CharCount:         utils.Len(child.Text),
					TokenCount:        utils.EstimateTokens(child.Text),
					VectorStatus:      enum.VectorStatusWaitVector,
				})
				parentBlock.ChildCount++
			}
		}
		parentBlock.EndChunkNo = globalChunkNo - 1
		parentBlocks = append(parentBlocks, parentBlock)
	}
	return parentBlocks, chunks
}
