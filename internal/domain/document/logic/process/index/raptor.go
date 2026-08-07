package index

import (
	"context"
	"encoding/json"
	"time"

	"github.com/swiftbit/know-agent/common/logx"
	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/document/model/enum"
)

// RaptorPhase RAPTOR 层级摘要树构建阶段
type RaptorPhase struct {
	*PhaseDeps
}

func NewRaptorPhase(deps *PhaseDeps) *RaptorPhase {
	return &RaptorPhase{PhaseDeps: deps}
}

func (p *RaptorPhase) Name() string {
	return "RAPTOR 构建阶段"
}

func (p *RaptorPhase) Execute(ctx context.Context, buildCtx *Context) error {
	vectorSize := len(buildCtx.ChildChunks)

	// 标记开始 RAPTOR 构建
	markRaptorStartTx := func(txCtx context.Context) error {
		if err := p.Repo.UpdateTaskById(txCtx, &entity.DocumentTask{
			ID: buildCtx.TaskId, CurrentStage: enum.TaskStageRaptor,
		}); err != nil {
			return err
		}
		raptorStartDetail, _ := json.Marshal(map[string]any{
			"chunkCount":  vectorSize,
			"parentCount": len(buildCtx.ParentBlocks),
		})
		raptorStartLog := &entity.DocumentTaskLog{
			TaskId:       buildCtx.TaskId,
			DocumentId:   buildCtx.DocumentId,
			StageType:    enum.TaskStageRaptor,
			EventType:    enum.TaskEventStart,
			LogLevel:     enum.LogLevelInfo,
			OperatorType: enum.OperatorTypeSystem,
			Content:      "开始构建 RAPTOR 层级摘要树",
			DetailJson:   string(raptorStartDetail),
		}
		return p.Repo.InsertTaskLog(txCtx, raptorStartLog)
	}
	if err := p.Repo.Do(ctx, markRaptorStartTx); err != nil {
		return err
	}

	// 执行 RAPTOR 构建
	raptorStartedNanos := time.Now()
	raptorBuildResult, err := p.RaptorBuilder.RebuildDocumentTree(ctx, buildCtx.DocumentId, buildCtx.TaskId, buildCtx.ChildChunks)
	if err != nil {
		return err
	}
	buildCtx.RaptorBuildResult = raptorBuildResult
	buildCtx.RaptorCostMillis = time.Since(raptorStartedNanos).Milliseconds()
	logx.Infof("RAPTOR 构建阶段完成，documentId=%d, taskId=%d, nodeCount=%d, levelCount=%d, costMillis=%d",
		buildCtx.DocumentId, buildCtx.TaskId, raptorBuildResult.NodeCount, raptorBuildResult.LevelCount, buildCtx.RaptorCostMillis)

	// 记录完成日志
	markRaptorCompleteTx := func(txCtx context.Context) error {
		raptorEndDetail, _ := json.Marshal(map[string]any{
			"nodeCount":   raptorBuildResult.NodeCount,
			"levelCount":  raptorBuildResult.LevelCount,
			"sourceCount": raptorBuildResult.SourceChunkCount,
			"costMillis":  buildCtx.RaptorCostMillis,
		})
		raptorEndLog := &entity.DocumentTaskLog{
			TaskId: buildCtx.TaskId, DocumentId: buildCtx.DocumentId,
			StageType: enum.TaskStageRaptor, EventType: enum.TaskEventComplete,
			LogLevel: enum.LogLevelInfo, OperatorType: enum.OperatorTypeSystem,
			Content: "RAPTOR 层级摘要树构建完成", DetailJson: string(raptorEndDetail),
		}
		return p.Repo.InsertTaskLog(txCtx, raptorEndLog)
	}
	return p.Repo.Do(ctx, markRaptorCompleteTx)
}
