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
	"github.com/swiftbit/know-agent/internal/domain/document/model/vo"
)

// ChunkingPhase 切块阶段：执行切块流水线、构建父子块实体、持久化
type ChunkingPhase struct {
	deps *PhaseDeps
}

func NewChunkingPhase(deps *PhaseDeps) *ChunkingPhase {
	return &ChunkingPhase{deps: deps}
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
		buildCtx.DocumentID, buildCtx.TaskID)
	return nil
}

// executeChunkingPipeline 执行切块流水线
func (p *ChunkingPhase) executeChunkingPipeline(ctx context.Context, buildCtx *BuildContext) error {
	// 下载解析文本
	parsedText, err := p.deps.Port.DownloadText(ctx, buildCtx.Document.ParseTextPath)
	if err != nil {
		return err
	}

	// 按步骤执行切块流水线
	chunkStartedNanos := time.Now()
	parentCandidates, err := p.deps.Coordinator.BuildParentBlocks(ctx, buildCtx.Document, buildCtx.PipelineSteps, parsedText)
	if err != nil {
		return err
	}
	buildCtx.ParentCandidates = parentCandidates
	buildCtx.ChunkCostMillis = time.Since(chunkStartedNanos).Milliseconds()
	logx.Infof("切块流水线执行完成，documentId=%d, taskId=%d, parentCount=%d, childCount=%d, costMillis=%d",
		buildCtx.DocumentID, buildCtx.TaskID, len(parentCandidates), p.countChildCandidates(parentCandidates), buildCtx.ChunkCostMillis)

	// 事务性标记切块完成
	markChunkCompleteTx := func(txCtx context.Context) error {
		if err := p.deps.Repo.UpdateStepExecuteStatus(txCtx, buildCtx.Plan.ID, vo.StrategyExecuteStatusExecuteSuccess); err != nil {
			return err
		}
		chunkEndDetail, _ := json.Marshal(map[string]any{
			"parentCount": len(parentCandidates),
			"childCount":  p.countChildCandidates(parentCandidates),
			"costMillis":  buildCtx.ChunkCostMillis,
		})
		chunkEndLog := &entity.DocumentTaskLog{
			TaskId: buildCtx.TaskID, DocumentId: buildCtx.DocumentID,
			StageType: vo.TaskStageChunkExecute, EventType: vo.TaskEventComplete,
			LogLevel: vo.LogLevelInfo, OperatorType: vo.OperatorTypeSystem,
			Content: "切块执行完成", DetailJson: string(chunkEndDetail),
		}
		if err := p.deps.Repo.InsertTaskLog(txCtx, chunkEndLog); err != nil {
			return err
		}
		return p.deps.Repo.UpdateTaskById(txCtx, &entity.DocumentTask{
			ID: buildCtx.TaskID, CurrentStage: vo.TaskStageChunkPostProcess,
		})
	}
	if err := p.deps.Repo.Do(ctx, markChunkCompleteTx); err != nil {
		return err
	}

	// 清理候选并构造持久化实体
	processStartedNanos := time.Now()
	finalCandidates := p.cleanupParentCandidates(parentCandidates)
	parentBlocks, childChunks := p.buildParentChildEntities(buildCtx.DocumentID, buildCtx.TaskID, buildCtx.PlanID, finalCandidates)
	buildCtx.ParentBlocks = parentBlocks
	buildCtx.ChildChunks = childChunks
	buildCtx.ProcessCostMillis = time.Since(processStartedNanos).Milliseconds()
	logx.Infof("切块后处理完成，documentId=%d, taskId=%d, parentCount=%d, childCount=%d, costMillis=%d",
		buildCtx.DocumentID, buildCtx.TaskID, len(finalCandidates), p.countChildCandidates(finalCandidates), buildCtx.ProcessCostMillis)

	// 事务性批量落库
	persistBlocksTx := func(txCtx context.Context) error {
		if err := p.deps.Repo.InsertParentBlockBatch(txCtx, parentBlocks); err != nil {
			return err
		}
		if err := p.deps.Repo.InsertChunkBatch(txCtx, childChunks); err != nil {
			return err
		}
		chunkPostDetail, _ := json.Marshal(map[string]any{
			"parentCount": len(finalCandidates),
			"childCount":  p.countChildCandidates(finalCandidates),
			"costMillis":  buildCtx.ProcessCostMillis,
		})
		chunkPostLog := &entity.DocumentTaskLog{
			TaskId: buildCtx.TaskID, DocumentId: buildCtx.DocumentID,
			StageType: vo.TaskStageChunkPostProcess, EventType: vo.TaskEventComplete,
			LogLevel: vo.LogLevelInfo, OperatorType: vo.OperatorTypeSystem,
			Content: "切块后处理完成", DetailJson: string(chunkPostDetail),
		}
		if err := p.deps.Repo.InsertTaskLog(txCtx, chunkPostLog); err != nil {
			return err
		}
		return p.deps.Repo.UpdateTaskById(txCtx, &entity.DocumentTask{
			ID: buildCtx.TaskID, CurrentStage: vo.TaskStageVectorize,
		})
	}
	return p.deps.Repo.Do(ctx, persistBlocksTx)
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
					VectorStatus:      vo.VectorStatusWaitVector,
				})
				parentBlock.ChildCount++
			}
		}
		parentBlock.EndChunkNo = globalChunkNo - 1
		parentBlocks = append(parentBlocks, parentBlock)
	}
	return parentBlocks, chunks
}
