package index

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/swiftbit/know-agent/common/logx"
	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/document/model/enum"
	"github.com/swiftbit/know-agent/internal/domain/document/model/vo"
)

// GraphRagPhase GraphRAG 构建阶段
type GraphRagPhase struct {
	*PhaseDeps
}

func NewGraphRagPhase(deps *PhaseDeps) *GraphRagPhase {
	return &GraphRagPhase{deps}
}

func (p *GraphRagPhase) Name() string {
	return "GraphRAG 构建阶段"
}

func (p *GraphRagPhase) Execute(ctx context.Context, buildCtx *Context) error {
	vectorSize := len(buildCtx.ChildChunks)

	// 如果是从已提交 GraphRAG 恢复，跳过构建
	if buildCtx.ResumeCommittedGraph {
		logx.Infof("GraphRAG 已提交，跳过构建，documentId=%d, taskId=%d", buildCtx.DocumentId, buildCtx.TaskId)
		return p.finalizeGraphRag(ctx, buildCtx)
	}

	// ========== GraphRAG 构建阶段 ==========
	markGraphRagStartTx := func(txCtx context.Context) error {
		if err := p.Repo.UpdateTaskById(txCtx, &entity.DocumentTask{
			ID: buildCtx.TaskId, CurrentStage: enum.TaskStageGraphRag,
		}); err != nil {
			return err
		}
		graphStartDetail, _ := json.Marshal(map[string]any{
			"chunkCount":  vectorSize,
			"parentCount": len(buildCtx.ParentBlocks),
		})
		graphStartLog := &entity.DocumentTaskLog{
			TaskId: buildCtx.TaskId, DocumentId: buildCtx.DocumentId,
			StageType: enum.TaskStageGraphRag, EventType: enum.TaskEventStart,
			LogLevel: enum.LogLevelInfo, OperatorType: enum.OperatorTypeSystem,
			Content: "开始构建 GraphRAG 实体关系图谱", DetailJson: string(graphStartDetail),
		}
		return p.Repo.InsertTaskLog(txCtx, graphStartLog)
	}
	if err := p.Repo.Do(ctx, markGraphRagStartTx); err != nil {
		return err
	}

	// 执行 GraphRAG 构建
	graphRagStartedNanos := time.Now()
	graphRagBuildResult, err := p.GraphRagBuilder.RebuildDocumentGraph(ctx, buildCtx.DocumentId, buildCtx.TaskId, buildCtx.ChildChunks)
	if err != nil {
		// 构建失败，使用已有的结果或创建新的失败结果
		if graphRagBuildResult == nil {
			graphRagBuildResult = &vo.GraphRagBuildResult{
				GraphPersistenceOutcome: vo.GraphPersistenceOutcomeFailed,
				KgCommitted:             false,
			}
		}
		buildCtx.GraphRagBuildResult = graphRagBuildResult
		return p.handleGraphRagBuildFailure(ctx, buildCtx, err)
	}
	buildCtx.GraphRagBuildResult = graphRagBuildResult
	buildCtx.GraphRagCostMillis = time.Since(graphRagStartedNanos).Milliseconds()
	logx.Infof("GraphRAG 构建阶段完成，documentId=%d, taskId=%d, entityCount=%d, relationCount=%d, costMillis=%d",
		buildCtx.DocumentId, buildCtx.TaskId, graphRagBuildResult.EntityCount, graphRagBuildResult.RelationCount, buildCtx.GraphRagCostMillis)

	// 记录构建完成日志
	markGraphRagCompleteTx := func(txCtx context.Context) error {
		graphEndDetail, _ := json.Marshal(map[string]any{
			"entityCount":   graphRagBuildResult.EntityCount,
			"relationCount": graphRagBuildResult.RelationCount,
			"costMillis":    buildCtx.GraphRagCostMillis,
		})
		graphEndLog := &entity.DocumentTaskLog{
			TaskId: buildCtx.TaskId, DocumentId: buildCtx.DocumentId,
			StageType: enum.TaskStageGraphRag, EventType: enum.TaskEventComplete,
			LogLevel: enum.LogLevelInfo, OperatorType: enum.OperatorTypeSystem,
			Content: "GraphRAG 实体关系图谱构建完成", DetailJson: string(graphEndDetail),
		}
		return p.Repo.InsertTaskLog(txCtx, graphEndLog)
	}
	if err := p.Repo.Do(ctx, markGraphRagCompleteTx); err != nil {
		return err
	}

	// 最终化 GraphRAG 结果
	return p.finalizeGraphRag(ctx, buildCtx)
}

// finalizeGraphRag 最终化 GraphRAG 结果处理
func (p *GraphRagPhase) finalizeGraphRag(ctx context.Context, buildCtx *Context) error {
	buildResult := buildCtx.GraphRagBuildResult
	if buildResult == nil || buildResult.GraphPersistenceOutcome == "" {
		return p.handleGraphRagBuildFailure(ctx, buildCtx, errors.New("GraphRAG build did not return an explicit save outcome."))
	}

	// 处理最终结果
	graphFinalization := p.finalizeGraphRagOutcome(ctx, buildCtx, buildResult)
	buildCtx.GraphFinalization = graphFinalization
	buildCtx.GraphRagBuildResult = graphFinalization.Result

	// 处理类型化块列表
	var graphTypedChunkList []vo.TypedChunk
	if graphFinalization.TypedChunks != nil {
		graphTypedChunkList = graphFinalization.TypedChunks
	}
	buildCtx.GraphTypedChunkList = graphTypedChunkList

	// 检查处置结果
	if buildResult.OuterTaskDisposition == vo.OuterTaskDispositionRepairRequired {
		if err := p.Repo.UpdateTaskById(ctx, &entity.DocumentTask{
			ID: buildCtx.TaskId, TaskStatus: enum.TaskStatusRunning,
			CurrentStage: enum.TaskStageGraphTypedIndex,
		}); err != nil {
			return err
		}
		logx.Warnf("GraphRAG post-commit component requires repair; BUILD_INDEX remains RUNNING: documentId=%d, taskId=%d",
			buildCtx.DocumentId, buildCtx.TaskId)
		return nil // 需要外部修复
	}
	if buildResult.OuterTaskDisposition == vo.OuterTaskDispositionFailIndexTask {
		return p.handleGraphRagBuildFailure(ctx, buildCtx, buildResult)
	}

	return nil
}

// finalizeGraphRagOutcome 最终化 GraphRAG 结果
func (p *GraphRagPhase) finalizeGraphRagOutcome(ctx context.Context, buildCtx *Context,
	buildResult *vo.GraphRagBuildResult) *vo.GraphRagFinalization {

	var typedChunks []vo.TypedChunk
	typedOutcome := vo.ComponentOutcomeNotApplicable

	if !buildResult.KgCommitted || buildResult.GraphPersistenceOutcome == vo.GraphPersistenceOutcomeFailed {
		typedOutcome = vo.ComponentOutcomeNotApplicable
	} else {
		// TODO: 实现 listFrozenTypedChunks
		existingTypedChunks := []*entity.DocumentChunk{} // 占位

		existingTypedInterface := make([]vo.TypedChunk, len(existingTypedChunks))
		for i, chunk := range existingTypedChunks {
			existingTypedInterface[i] = chunk
		}

		graphEmpty := buildResult.GraphPersistenceOutcome == vo.GraphPersistenceOutcomeEmpty
		reuseSuccessfulTyped := buildCtx.ResumeCommittedGraph &&
			buildResult.TypedIndexOutcome == vo.ComponentOutcomeSuccess &&
			len(existingTypedChunks) > 0
		reuseEmptyTyped := buildCtx.ResumeCommittedGraph &&
			graphEmpty &&
			buildResult.TypedIndexOutcome == vo.ComponentOutcomeNotApplicable &&
			len(existingTypedChunks) == 0

		if reuseSuccessfulTyped {
			typedChunks = existingTypedInterface
			typedOutcome = vo.ComponentOutcomeSuccess
		} else if reuseEmptyTyped {
			typedOutcome = vo.ComponentOutcomeNotApplicable
		} else {
			if err := p.Repo.UpdateTaskById(ctx, &entity.DocumentTask{
				ID: buildCtx.TaskId, CurrentStage: enum.TaskStageGraphTypedIndex,
			}); err != nil {
				logx.Warnf("更新任务阶段失败: %v", err)
			}

			replaced, err := p.GraphRagBuilder.ReplaceTypedIndex(ctx, buildCtx.DocumentId, buildCtx.TaskId,
				buildCtx.PlanId, buildCtx.ChildChunks, p.nextChunkNo(buildCtx.ChildChunks))
			if err != nil {
				logx.Warnf("GraphRAG typed projection failed; preserving committed KG: documentId=%d, taskId=%d, message=%v",
					buildCtx.DocumentId, buildCtx.TaskId, err)
				typedChunks = []vo.TypedChunk{}
				typedOutcome = vo.ComponentOutcomeFailed
			} else {
				if replaced == nil {
					typedChunks = []vo.TypedChunk{}
				} else {
					typedChunks = make([]vo.TypedChunk, len(replaced))
					for i, chunk := range replaced {
						typedChunks[i] = chunk
					}
				}
				if graphEmpty && len(typedChunks) == 0 {
					typedOutcome = vo.ComponentOutcomeNotApplicable
				} else if len(typedChunks) == 0 {
					typedOutcome = vo.ComponentOutcomeFailed
				} else {
					typedOutcome = vo.ComponentOutcomeSuccess
				}
			}
		}
	}

	// 计算候选最终结果
	candidate := p.GraphRagOutcomePolicy.FinalizeOuterDisposition(buildResult, typedOutcome, vo.ObservationProjectionOutcomeSuccess)
	if candidate.OuterTaskDisposition == vo.OuterTaskDispositionRepairRequired {
		candidate = p.withdrawPendingCrossDocumentProjection(ctx, buildCtx, candidate)
	}

	// 标记检查点
	if err := p.GraphRagBuildCheckpoint.MarkOutcome(ctx, buildCtx.DocumentId, buildCtx.TaskId, candidate,
		p.resultAttempt(candidate), p.resultMaxAttempts(candidate)); err != nil {
		logx.Warnf("GraphRAG final outcome projection failed; BUILD_INDEX remains repairable: documentId=%d, taskId=%d, message=%v",
			buildCtx.DocumentId, buildCtx.TaskId, err)
		failedObservation := p.GraphRagOutcomePolicy.FinalizeOuterDisposition(buildResult, typedOutcome, vo.ObservationProjectionOutcomeFailed)
		failedObservation = p.withdrawPendingCrossDocumentProjection(ctx, buildCtx, failedObservation)
		return &vo.GraphRagFinalization{Result: failedObservation, TypedChunks: typedChunks}
	}

	return &vo.GraphRagFinalization{Result: candidate, TypedChunks: typedChunks}
}

// handleGraphRagBuildFailure 处理 GraphRAG 构建失败
func (p *GraphRagPhase) handleGraphRagBuildFailure(ctx context.Context, buildCtx *Context, cause error) error {
	logx.Errorf("GraphRAG 构建异常: documentId=%d, taskId=%d, planId=%d, err=%v",
		buildCtx.DocumentId, buildCtx.TaskId, buildCtx.PlanId, cause)

	failureResult := buildCtx.GraphRagBuildResult
	if failureResult == nil {
		failureResult = &vo.GraphRagBuildResult{
			GraphPersistenceOutcome: vo.GraphPersistenceOutcomeFailed,
			KgCommitted:             false,
		}
	}

	// 计算最终结果
	terminalResult := p.GraphRagOutcomePolicy.FinalizeOuterDisposition(
		failureResult, vo.ComponentOutcomeNotApplicable, vo.ObservationProjectionOutcomeSuccess)

	// 尝试标记检查点
	if err := p.GraphRagBuildCheckpoint.MarkOutcome(ctx, buildCtx.DocumentId, buildCtx.TaskId, terminalResult, 0, 1); err != nil {
		logx.Warnf("GraphRAG 检查点标记失败: documentId=%d, taskId=%d, err=%v", buildCtx.DocumentId, buildCtx.TaskId, err)
		terminalResult = p.GraphRagOutcomePolicy.FinalizeOuterDisposition(
			failureResult, vo.ComponentOutcomeNotApplicable, vo.ObservationProjectionOutcomeFailed)
	}

	return p.applyGraphFailureDisposition(ctx, buildCtx, terminalResult, cause)
}

// applyGraphFailureDisposition 应用图谱失败处置
func (p *GraphRagPhase) applyGraphFailureDisposition(ctx context.Context, buildCtx *Context,
	result *vo.GraphRagBuildResult, cause error) error {
	document := buildCtx.Document
	task := buildCtx.Task
	planId := buildCtx.PlanId

	failedStage := enum.TaskStageGraphRag
	if task.CurrentStage != 0 {
		failedStage = task.CurrentStage
	}

	if cause == nil {
		cause = errors.New("GraphRAG build failed")
	}

	markFailureTx := func(txCtx context.Context) error {
		if err := p.Repo.UpdateDocumentById(txCtx, &entity.Document{
			ID: document.ID, IndexStatus: enum.IndexStatusBuildFailed,
		}); err != nil {
			return err
		}
		if err := p.Repo.UpdateChunkByTaskId(txCtx, &entity.DocumentChunk{
			TaskId: task.ID, VectorStatus: enum.VectorStatusVectorFailed,
		}); err != nil {
			return err
		}
		if err := p.Repo.UpdateStepExecuteStatus(txCtx, planId, enum.StrategyExecuteStatusExecuteFailed); err != nil {
			return err
		}
		if err := p.Repo.UpdateTaskById(txCtx, &entity.DocumentTask{
			ID: task.ID, TaskStatus: enum.TaskStatusFailed, CurrentStage: failedStage,
		}); err != nil {
			return err
		}
		failDetail, _ := json.Marshal(map[string]any{"error": cause.Error(), "currentStage": failedStage})
		failLog := &entity.DocumentTaskLog{
			TaskId: task.ID, DocumentId: task.DocumentId,
			StageType: failedStage, EventType: enum.TaskEventFailed,
			LogLevel: enum.LogLevelError, OperatorType: enum.OperatorTypeSystem,
			Content: "GraphRAG 构建失败", DetailJson: string(failDetail),
		}
		return p.Repo.InsertTaskLog(txCtx, failLog)
	}
	if err := p.Repo.Do(ctx, markFailureTx); err != nil {
		logx.Warnf("图谱失败时收尾失败: taskId=%d, err=%v", task.ID, err)
		return err
	}
	return nil
}

// withdrawPendingCrossDocumentProjection 撤回待处理的跨文档投影
func (p *GraphRagPhase) withdrawPendingCrossDocumentProjection(ctx context.Context,
	buildCtx *Context, buildResult *vo.GraphRagBuildResult) *vo.GraphRagBuildResult {
	if buildResult == nil ||
		buildResult.CrossDocumentIndexOutcome != vo.ComponentOutcomeSuccess ||
		buildCtx.Document == nil ||
		buildCtx.Document.LastIndexTaskId == buildCtx.TaskId {
		return buildResult
	}
	if err := p.CrossDocumentIndexer.RebuildAll(ctx, 0, 0); err != nil {
		logx.Errorf("GraphRAG pending cross-document projection withdrawal failed; task cannot publish current I: documentId=%d, taskId=%d, message=%v",
			buildCtx.DocumentId, buildCtx.TaskId, err)
	} else {
		logx.Infof("GraphRAG pending cross-document projection withdrawn to active document pointers: documentId=%d, taskId=%d",
			buildCtx.DocumentId, buildCtx.TaskId)
	}
	return p.GraphRagOutcomePolicy.WithCrossDocumentOutcome(buildResult, vo.ComponentOutcomeFailed)
}

// resultAttempt 获取尝试次数
func (p *GraphRagPhase) resultAttempt(result *vo.GraphRagBuildResult) int {
	if result == nil {
		return 0
	}
	if result.Attempt < 0 {
		return 0
	}
	return result.Attempt
}

// resultMaxAttempts 获取最大尝试次数
func (p *GraphRagPhase) resultMaxAttempts(result *vo.GraphRagBuildResult) int {
	if result == nil {
		return 1
	}
	if result.MaxAttempts < 1 {
		return 1
	}
	return result.MaxAttempts
}

// nextChunkNo 获取下一个块编号
func (p *GraphRagPhase) nextChunkNo(chunks []*entity.DocumentChunk) int {
	if len(chunks) == 0 {
		return 1
	}
	maxNo := 0
	for _, chunk := range chunks {
		if chunk != nil && chunk.ChunkNo > maxNo {
			maxNo = chunk.ChunkNo
		}
	}
	return maxNo + 1
}
