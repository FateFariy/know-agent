package index

import (
	"context"
	"encoding/json"
	"time"

	"github.com/swiftbit/know-agent/common/logx"
	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/document/model/vo"
)

// RaptorPhase RAPTOR 层级摘要树构建阶段
type RaptorPhase struct {
	deps *PhaseDeps
}

func NewRaptorPhase(deps *PhaseDeps) *RaptorPhase {
	return &RaptorPhase{deps: deps}
}

func (p *RaptorPhase) Name() string {
	return "RAPTOR 构建阶段"
}

func (p *RaptorPhase) Execute(ctx context.Context, buildCtx *BuildContext) error {
	vectorSize := len(buildCtx.ChildChunks)

	// 标记开始 RAPTOR 构建
	markRaptorStartTx := func(txCtx context.Context) error {
		if err := p.deps.Repo.UpdateTaskById(txCtx, &entity.DocumentTask{
			ID: buildCtx.TaskID, CurrentStage: vo.TaskStageRaptor,
		}); err != nil {
			return err
		}
		raptorStartDetail, _ := json.Marshal(map[string]any{
			"chunkCount":  vectorSize,
			"parentCount": len(buildCtx.ParentBlocks),
		})
		raptorStartLog := &entity.DocumentTaskLog{
			TaskId: buildCtx.TaskID, DocumentId: buildCtx.DocumentID,
			StageType: vo.TaskStageRaptor, EventType: vo.TaskEventStart,
			LogLevel: vo.LogLevelInfo, OperatorType: vo.OperatorTypeSystem,
			Content: "开始构建 RAPTOR 层级摘要树", DetailJson: string(raptorStartDetail),
		}
		return p.deps.Repo.InsertTaskLog(txCtx, raptorStartLog)
	}
	if err := p.deps.Repo.Do(ctx, markRaptorStartTx); err != nil {
		return err
	}

	// 执行 RAPTOR 构建
	raptorStartedNanos := time.Now()
	raptorBuildResult, err := p.deps.RaptorBuilder.RebuildDocumentTree(ctx, buildCtx.DocumentID, buildCtx.TaskID, buildCtx.ChildChunks)
	if err != nil {
		return err
	}
	buildCtx.RaptorBuildResult = raptorBuildResult
	buildCtx.RaptorCostMillis = time.Since(raptorStartedNanos).Milliseconds()
	logx.Infof("RAPTOR 构建阶段完成，documentId=%d, taskId=%d, nodeCount=%d, levelCount=%d, costMillis=%d",
		buildCtx.DocumentID, buildCtx.TaskID, raptorBuildResult.NodeCount, raptorBuildResult.LevelCount, buildCtx.RaptorCostMillis)

	// 记录完成日志
	markRaptorCompleteTx := func(txCtx context.Context) error {
		raptorEndDetail, _ := json.Marshal(map[string]any{
			"nodeCount":   raptorBuildResult.NodeCount,
			"levelCount":  raptorBuildResult.LevelCount,
			"sourceCount": raptorBuildResult.SourceChunkCount,
			"costMillis":  buildCtx.RaptorCostMillis,
		})
		raptorEndLog := &entity.DocumentTaskLog{
			TaskId: buildCtx.TaskID, DocumentId: buildCtx.DocumentID,
			StageType: vo.TaskStageRaptor, EventType: vo.TaskEventComplete,
			LogLevel: vo.LogLevelInfo, OperatorType: vo.OperatorTypeSystem,
			Content: "RAPTOR 层级摘要树构建完成", DetailJson: string(raptorEndDetail),
		}
		return p.deps.Repo.InsertTaskLog(txCtx, raptorEndLog)
	}
	return p.deps.Repo.Do(ctx, markRaptorCompleteTx)
}
