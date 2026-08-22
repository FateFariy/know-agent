package index

import (
	"context"
	"encoding/json"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/document/adapter"
	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/document/model/enum"
)

type ChunkPostStage struct {
	repo adapter.DocumentRepository
}

func NewChunkPostPhase(repo adapter.DocumentRepository) *ChunkPostStage {
	return &ChunkPostStage{repo: repo}
}

func (p *ChunkPostStage) Name() string {
	return "切块后处理阶段"
}

func (p *ChunkPostStage) Execute(ctx context.Context, buildCtx *Context) error {
	// 检查是否需要从已提交 GraphRAG 结果恢复
	if buildCtx.ResumeCommittedGraph {
		return nil
	}

	buildCtx.Task.CurrentStage = enum.TaskStageChunkPostProcess
	if err := p.repo.UpdateTaskById(ctx, &entity.DocumentTask{
		ID:           buildCtx.Task.ID,
		CurrentStage: enum.TaskStageChunkPostProcess,
	}); err != nil {
		return err
	}
	buildCtx.ParentCandidates = buildCtx.ParentCandidates.WithoutValidChildren()

	parentChunks, childChunks := p.buildParentChildEntities(buildCtx)
	buildCtx.ParentChunks = parentChunks
	buildCtx.ChildChunks = childChunks

	fn := func(txCtx context.Context) error {
		// 批量写入父块
		if err := p.repo.InsertParentChunkBatch(txCtx, parentChunks); err != nil {
			return err
		}
		// 批量写入子块
		return p.repo.InsertChunkBatch(txCtx, childChunks)
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
func (p *ChunkPostStage) buildParentChildEntities(buildCtx *Context) ([]*entity.DocumentParentChunk, []*entity.DocumentChunk) {
	parentChunks := make([]*entity.DocumentParentChunk, 0, len(buildCtx.ParentCandidates))
	childChunks := make([]*entity.DocumentChunk, 0)

	globalChunkNo := 0
	for parentIdx, candidate := range buildCtx.ParentCandidates {
		parentChunk := &entity.DocumentParentChunk{
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
			PageRange:         candidate.PageRange,
			SourceBlockIds:    candidate.SourceBlockIds,
		}

		for _, child := range candidate.ChildChunks {
			if child != nil && child.Text != "" {
				globalChunkNo++
				childChunks = append(childChunks, &entity.DocumentChunk{
					ID:                utils.GetSnowflakeNextID(),
					DocumentId:        buildCtx.DocumentId,
					TaskId:            buildCtx.TaskId,
					PlanId:            buildCtx.PlanId,
					ParentChunkId:     parentChunk.ID,
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
					ContentWithWeight: utils.BlankToDefault(child.ContentWithWeight, child.Text),
					ChunkType:         child.ChunkType,
					Title:             child.Title,
					Keywords:          child.Keywords,
					Questions:         child.Questions,
					PageNo:            child.PageNo,
					PageRange:         child.PageRange,
					BboxJson:          child.BboxJson,
					SourceBlockIds:    child.SourceBlockIds,
				})
				parentChunk.ChildCount++
			}
		}
		parentChunk.EndChunkNo = globalChunkNo - 1
		parentChunks = append(parentChunks, parentChunk)
	}
	return parentChunks, childChunks
}
