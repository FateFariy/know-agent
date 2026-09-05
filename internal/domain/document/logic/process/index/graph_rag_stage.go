package index

import (
	"context"
	"encoding/json"
	"time"

	"github.com/swiftbit/know-agent/common/logx"
	"github.com/swiftbit/know-agent/internal/domain/document/adapter"
	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/document/model/enum"
	"github.com/swiftbit/know-agent/internal/domain/document/model/vo"
	errorx "github.com/swiftbit/know-agent/internal/error"
)

// graphRagStage GraphRAG 构建阶段（M1：抽取→合并→算法→落库 Neo4j）
type graphRagStage struct {
	builder GraphRagBuilder
	repo    adapter.DocumentRepository
}

// NewGraphRagPhase 创建 GraphRAG 构建阶段
func NewGraphRagPhase(repo adapter.DocumentRepository, builder GraphRagBuilder) Stage {
	return &graphRagStage{builder: builder, repo: repo}
}

func (s *graphRagStage) Name() string { return "GraphRagStage" }

// Execute 执行 GraphRAG 构建。
// 图谱为增强依赖：未装配 builder / 无子块 / 已提交过图谱（断点恢复）时直接跳过；
// 构建失败记录 FAILED 结果并返回 ErrGraphRagBuildFailed，由责任链按降级语义 continue，
// 保证「图谱失败不影响基础检索」且索引任务仍能正常完成收尾。
func (s *graphRagStage) Execute(ctx context.Context, buildCtx *Context) error {
	if s.builder == nil {
		return nil
	}
	if buildCtx == nil || len(buildCtx.ChildChunks) == 0 {
		return nil
	}
	if buildCtx.ResumeCommittedGraph {
		logx.Infof("从已提交 GraphRAG outcome 恢复索引任务，跳过图谱构建，documentId=%d, taskId=%d",
			buildCtx.DocumentId, buildCtx.TaskId)
		return nil
	}

	kbId := int64(0)
	if buildCtx.Document != nil {
		kbId = buildCtx.Document.KnowledgeBaseId
	}

	// 推进任务阶段 + 记录开始日志
	buildCtx.Task.CurrentStage = enum.TaskStageGraphRag
	if err := s.repo.UpdateTaskById(ctx, &entity.DocumentTask{
		ID:           buildCtx.TaskId,
		CurrentStage: enum.TaskStageGraphRag,
	}); err != nil {
		return err
	}
	graphStartDetail, _ := json.Marshal(map[string]any{
		"chunkCount": len(buildCtx.ChildChunks),
	})
	graphStartLog := &entity.DocumentTaskLog{
		TaskId:       buildCtx.TaskId,
		DocumentId:   buildCtx.DocumentId,
		StageType:    enum.TaskStageGraphRag,
		EventType:    enum.TaskEventStart,
		LogLevel:     enum.LogLevelInfo,
		OperatorType: enum.OperatorTypeSystem,
		Content:      "开始构建 GraphRAG 实体关系图谱",
		DetailJson:   string(graphStartDetail),
	}
	_ = s.repo.InsertTaskLog(ctx, graphStartLog)

	// 执行图谱构建（LLM 抽取 → 消解合并 → PageRank → 社区发现 → 全删全插落库）
	graphStartedTime := time.Now()
	result, err := s.builder.RebuildDocumentGraph(ctx, buildCtx.DocumentId, buildCtx.TaskId, kbId, buildCtx.ChildChunks)
	costMillis := time.Since(graphStartedTime).Milliseconds()
	if err != nil {
		if result == nil {
			result = &vo.GraphRagBuildResult{}
		}
		result.GraphPersistenceOutcome = enum.GraphPersistenceOutcomeFailed
		result.GraphPersistenceReason = err.Error()
		buildCtx.GraphRagBuildResult = result
		graphFailDetail, _ := json.Marshal(map[string]any{
			"costMillis":        costMillis,
			"degradationReason": result.GraphPersistenceReason,
		})
		graphFailLog := &entity.DocumentTaskLog{
			TaskId:       buildCtx.TaskId,
			DocumentId:   buildCtx.DocumentId,
			StageType:    enum.TaskStageGraphRag,
			EventType:    enum.TaskEventFailed,
			LogLevel:     enum.LogLevelError,
			OperatorType: enum.OperatorTypeSystem,
			Content:      "GraphRAG 实体关系图谱构建失败（不阻断索引任务）",
			DetailJson:   string(graphFailDetail),
		}
		_ = s.repo.InsertTaskLog(ctx, graphFailLog)
		logx.Warnf("GraphRAG 构建失败，按降级策略继续完成索引任务: documentId=%d, taskId=%d, err=%v",
			buildCtx.DocumentId, buildCtx.TaskId, err)
		return errorx.ErrGraphRagBuildFailed
	}
	if result == nil {
		logx.Warnf("GraphRAG builder 未返回结果，跳过本次图谱构建: documentId=%d, taskId=%d",
			buildCtx.DocumentId, buildCtx.TaskId)
		return nil
	}
	buildCtx.GraphRagBuildResult = result

	// 记录完成日志
	graphEndDetail, _ := json.Marshal(map[string]any{
		"entityCount":        result.EntityCount,
		"relationCount":      result.RelationCount,
		"evidenceCount":      result.EvidenceCount,
		"communityCount":     result.CommunityCount,
		"graphRagCostMillis": costMillis,
	})
	graphEndLog := &entity.DocumentTaskLog{
		TaskId:       buildCtx.TaskId,
		DocumentId:   buildCtx.DocumentId,
		StageType:    enum.TaskStageGraphRag,
		EventType:    enum.TaskEventComplete,
		LogLevel:     enum.LogLevelInfo,
		OperatorType: enum.OperatorTypeSystem,
		Content:      "GraphRAG 实体关系图谱构建完成",
		DetailJson:   string(graphEndDetail),
	}
	_ = s.repo.InsertTaskLog(ctx, graphEndLog)
	return nil
}
