package index

import (
	"context"
	"encoding/json"
	"time"

	"github.com/swiftbit/know-agent/common/logx"
	"github.com/swiftbit/know-agent/internal/domain/document/adapter"
	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/document/model/enum"
)

const embeddingBatch = 100 // 默认向量化批大小

// VectorizePhase 向量化阶段：批量向量化并回写状态
type VectorizePhase struct {
	repo adapter.DocumentRepository
	port *adapter.DocumentPort
}

func NewVectorizePhase(repo adapter.DocumentRepository, port *adapter.DocumentPort) *VectorizePhase {
	return &VectorizePhase{repo: repo, port: port}
}

func (p *VectorizePhase) Name() string {
	return "向量化阶段"
}

func (p *VectorizePhase) Execute(ctx context.Context, buildCtx *Context) error {
	if buildCtx.ResumeCommittedGraph {
		logx.Infof("GraphRAG 已提交，跳过构建向量，documentId=%d, taskId=%d", buildCtx.DocumentId, buildCtx.TaskId)
		return nil
	}

	task := &entity.DocumentTask{
		ID:           buildCtx.TaskId,
		CurrentStage: enum.TaskStageVectorize,
	}
	if err := p.repo.UpdateTaskById(ctx, task); err != nil {
		return err
	}

	vectorSize := len(buildCtx.ChildChunks)
	vectorBatch := (vectorSize + embeddingBatch - 1) / embeddingBatch
	detail := map[string]any{
		"chunkCount":          vectorSize,
		"embeddingBatchSize":  embeddingBatch,
		"embeddingBatchCount": vectorBatch,
		"vectorStoreType":     enum.VectorStoreTypeMilvus,
		"parentCount":         len(buildCtx.ParentBlocks),
	}
	// 记录"开始执行向量化"日志
	vectorStartDetail, _ := json.Marshal(detail)
	vectorStartLog := &entity.DocumentTaskLog{
		TaskId:       buildCtx.TaskId,
		DocumentId:   buildCtx.DocumentId,
		StageType:    enum.TaskStageVectorize,
		EventType:    enum.TaskEventStart,
		LogLevel:     enum.LogLevelInfo,
		OperatorType: enum.OperatorTypeSystem,
		Content:      "开始执行向量化",
		DetailJson:   string(vectorStartDetail),
	}
	_ = p.repo.InsertTaskLog(ctx, vectorStartLog)

	// 批量向量化
	vectorStartedTime := time.Now()
	if err := p.port.BuildVectors(ctx, buildCtx.ChildChunks); err != nil {
		return err
	}

	// 回写向量状态
	if err := p.repo.UpdateBatchChunkById(ctx, buildCtx.ChildChunks, "vector_id"); err != nil {
		return err
	}
	// 记录"向量化完成"日志
	detail["vectorCostMillis"] = time.Since(vectorStartedTime).Milliseconds()
	vectorEndDetail, _ := json.Marshal(detail)
	vectorEndLog := &entity.DocumentTaskLog{
		TaskId:       buildCtx.TaskId,
		DocumentId:   buildCtx.DocumentId,
		StageType:    enum.TaskStageVectorize,
		EventType:    enum.TaskEventComplete,
		LogLevel:     enum.LogLevelInfo,
		OperatorType: enum.OperatorTypeSystem,
		Content:      "向量化完成",
		DetailJson:   string(vectorEndDetail),
	}
	_ = p.repo.InsertTaskLog(ctx, vectorEndLog)

	return nil
}
