package index

import (
	"context"
	"encoding/json"
	"time"

	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/document/model/vo"
)

const embeddingBatch = 100 // 默认向量化批大小

// VectorizePhase 向量化阶段：批量向量化并回写状态
type VectorizePhase struct {
	*PhaseDeps
}

func NewVectorizePhase(deps *PhaseDeps) *VectorizePhase {
	return &VectorizePhase{PhaseDeps: deps}
}

func (p *VectorizePhase) Name() string {
	return "向量化阶段"
}

func (p *VectorizePhase) Execute(ctx context.Context, buildCtx *BuildContext) error {
	vectorSize := len(buildCtx.ChildChunks)
	vectorBatch := (vectorSize + embeddingBatch - 1) / embeddingBatch

	detail := map[string]any{
		"chunkCount":          vectorSize,
		"embeddingBatchSize":  embeddingBatch,
		"embeddingBatchCount": vectorBatch,
		"vectorStoreType":     vo.VectorStoreTypeMilvus,
		"parentCount":         len(buildCtx.ParentBlocks),
	}
	// 记录"开始执行向量化"日志
	markVectorStartTx := func(txCtx context.Context) error {
		vectorStartDetail, _ := json.Marshal(detail)
		vectorStartLog := &entity.DocumentTaskLog{
			TaskId: buildCtx.TaskID, DocumentId: buildCtx.DocumentID,
			StageType: vo.TaskStageVectorize, EventType: vo.TaskEventStart,
			LogLevel: vo.LogLevelInfo, OperatorType: vo.OperatorTypeSystem,
			Content: "开始执行向量化", DetailJson: string(vectorStartDetail),
		}
		return p.Repo.InsertTaskLog(txCtx, vectorStartLog)
	}
	if err := p.Repo.Do(ctx, markVectorStartTx); err != nil {
		return err
	}

	// 批量向量化
	vectorStartedTime := time.Now()
	if err := p.Port.BuildVectors(ctx, buildCtx.ChildChunks); err != nil {
		return err
	}
	buildCtx.VectorCostMillis = time.Since(vectorStartedTime).Milliseconds()

	// 回写向量状态
	if err := p.Repo.UpdateBatchChunkById(ctx, buildCtx.ChildChunks); err != nil {
		return err
	}

	// 记录"向量化完成"日志
	markVectorCompleteTx := func(txCtx context.Context) error {
		detail["vectorCostMillis"] = buildCtx.VectorCostMillis
		vectorEndDetail, _ := json.Marshal(detail)
		vectorEndLog := &entity.DocumentTaskLog{
			TaskId: buildCtx.TaskID, DocumentId: buildCtx.DocumentID,
			StageType: vo.TaskStageVectorize, EventType: vo.TaskEventComplete,
			LogLevel: vo.LogLevelInfo, OperatorType: vo.OperatorTypeSystem,
			Content: "向量化完成", DetailJson: string(vectorEndDetail),
		}
		return p.Repo.InsertTaskLog(txCtx, vectorEndLog)
	}
	return p.Repo.Do(ctx, markVectorCompleteTx)
}
