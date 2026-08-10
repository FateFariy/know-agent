package index

import (
	"context"
	"encoding/json"

	"github.com/duke-git/lancet/v2/strutil"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/document/adapter"
	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/document/model/enum"
)

type ChunkPostPhase struct {
	repo adapter.DocumentRepository
}

func NewChunkPostPhase(repo adapter.DocumentRepository) *ChunkPostPhase {
	return &ChunkPostPhase{repo: repo}
}

func (p *ChunkPostPhase) Name() string {
	return "切块后处理阶段"
}

func (p *ChunkPostPhase) Execute(ctx context.Context, buildCtx *Context) error {
	buildCtx.Task.CurrentStage = enum.TaskStageChunkPostProcess
	if err := p.repo.UpdateTaskById(ctx, &entity.DocumentTask{
		ID:           buildCtx.Task.ID,
		CurrentStage: enum.TaskStageChunkPostProcess,
	}); err != nil {
		return err
	}
	buildCtx.ParentCandidates = buildCtx.ParentCandidates.CleanupParentCandidates()

	// todo 带实现大模型增强
	// applyChunkKeywordQuestionEnrichment(ctx, buildCtx)

	parentChunks, chunks := p.buildParentChildEntities(buildCtx)
	fn := func(txCtx context.Context) error {
		// 批量写入父块
		if err := p.repo.InsertParentChunkBatch(txCtx, parentChunks); err != nil {
			return err
		}
		// 批量写入子块
		return p.repo.InsertChunkBatch(txCtx, chunks)
	}

	if err := p.repo.Do(ctx, fn); err != nil {
		return err
	}

	chunkEndDetail, _ := json.Marshal(map[string]any{
		"parentCount": len(buildCtx.ParentCandidates),
		"childCount":  buildCtx.ParentCandidates.CountChildCandidates(),
	})
	chunkEndLog := &entity.DocumentTaskLog{
		TaskId:       buildCtx.TaskId,
		DocumentId:   buildCtx.DocumentId,
		StageType:    enum.TaskStageChunkPostProcess,
		EventType:    enum.TaskEventComplete,
		LogLevel:     enum.LogLevelInfo,
		OperatorType: enum.OperatorTypeSystem,
		Content:      "切块后处理完成",
		DetailJson:   string(chunkEndDetail),
	}
	_ = p.repo.InsertTaskLog(ctx, chunkEndLog)

	return nil
}

// buildParentChildEntities 将父块候选转换为可落库的"父块实体 + 子块实体"双列表
func (p *ChunkPostPhase) buildParentChildEntities(buildCtx *Context) ([]*entity.DocumentParentChunk, []*entity.DocumentChunk) {
	parentChunks := make([]*entity.DocumentParentChunk, 0, len(buildCtx.ParentCandidates))
	chunks := make([]*entity.DocumentChunk, 0)

	globalChunkNo := 0
	for parentIdx, candidate := range buildCtx.ParentCandidates {
		parentBlock := &entity.DocumentParentChunk{
			ID:                utils.GetSnowflakeNextID(),
			DocumentId:        buildCtx.DocumentId,
			TaskId:            buildCtx.TaskId,
			PlanId:            buildCtx.PlanId,
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
					DocumentId:        buildCtx.DocumentId,
					TaskId:            buildCtx.TaskId,
					PlanId:            buildCtx.PlanId,
					ParentChunkId:     parentBlock.ID,
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
		parentChunks = append(parentChunks, parentBlock)
	}
	return parentChunks, chunks
}
