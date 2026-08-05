package index

import (
	"context"
	"encoding/json"

	"github.com/swiftbit/know-agent/common/logx"
	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/document/model/vo"
)

// ValidationPhase 验证阶段：检查任务状态、读取 GraphRAG 检查点
type ValidationPhase struct {
	deps *PhaseDeps
}

func NewValidationPhase(deps *PhaseDeps) *ValidationPhase {
	return &ValidationPhase{deps: deps}
}

func (p *ValidationPhase) Name() string {
	return "验证阶段"
}

func (p *ValidationPhase) Execute(ctx context.Context, buildCtx *BuildContext) error {
	// 前置检查：如果任务已成功或失败，跳过重复执行
	if buildCtx.Task.TaskStatus == vo.TaskStatusSuccess {
		logx.Infof("索引构建任务已成功，跳过重复执行，documentId=%d, taskId=%d, planId=%d",
			buildCtx.DocumentID, buildCtx.TaskID, buildCtx.PlanID)
		return nil // 已完成，直接返回
	}
	if buildCtx.Task.TaskStatus == vo.TaskStatusFailed {
		logx.Infof("索引构建任务已失败，跳过重复执行，documentId=%d, taskId=%d, planId=%d",
			buildCtx.DocumentID, buildCtx.TaskID, buildCtx.PlanID)
		return nil // 已失败，直接返回
	}

	// 读取已有的 GraphRAG 构建结果（用于断点恢复）
	graphRagBuildResult := p.readGraphRagBuildResult(buildCtx.Task)
	buildCtx.GraphRagBuildResult = graphRagBuildResult

	// 检查是否需要直接失败
	if graphRagBuildResult != nil && graphRagBuildResult.OuterTaskDisposition == vo.OuterTaskDispositionFailIndexTask {
		p.applyGraphFailureDisposition(ctx, buildCtx.Document, buildCtx.Task, buildCtx.PlanID, graphRagBuildResult, nil)
		return nil
	}

	return nil
}

// readGraphRagBuildResult 从任务扩展 JSON 读取 GraphRAG 构建结果
func (p *ValidationPhase) readGraphRagBuildResult(task *entity.DocumentTask) *vo.GraphRagBuildResult {
	if task == nil || task.ExtJson == "" {
		return nil
	}
	var state map[string]any
	if err := json.Unmarshal([]byte(task.ExtJson), &state); err != nil {
		logx.Warnf("Ignoring unreadable GraphRAG outcome checkpoint: taskId=%d, message=%v", task.ID, err)
		return nil
	}
	rawState, ok := state["graphRagBuild"]
	if !ok {
		return nil
	}
	stateMap, ok := rawState.(map[string]any)
	if !ok {
		return nil
	}

	result := &vo.GraphRagBuildResult{}
	if v, ok := stateMap["entityCount"].(float64); ok {
		result.EntityCount = int(v)
	}
	if v, ok := stateMap["relationCount"].(float64); ok {
		result.RelationCount = int(v)
	}
	if v, ok := stateMap["evidenceCount"].(float64); ok {
		result.EvidenceCount = int(v)
	}
	if v, ok := stateMap["communityCount"].(float64); ok {
		result.CommunityCount = int(v)
	}
	if v, ok := stateMap["graphPersistenceOutcome"].(string); ok {
		result.GraphPersistenceOutcome = vo.GraphPersistenceOutcome(v)
	}
	if v, ok := stateMap["graphPersistenceReason"].(string); ok {
		result.GraphPersistenceReason = v
	}
	if v, ok := stateMap["kgCommitted"].(bool); ok {
		result.KgCommitted = v
	}
	if v, ok := stateMap["typedIndexOutcome"].(string); ok {
		result.TypedIndexOutcome = vo.ComponentOutcome(v)
	}
	if v, ok := stateMap["crossDocumentIndexOutcome"].(string); ok {
		result.CrossDocumentIndexOutcome = vo.ComponentOutcome(v)
	}
	if v, ok := stateMap["derivedIndexOutcome"].(string); ok {
		result.DerivedIndexOutcome = vo.DerivedIndexOutcome(v)
	}
	if v, ok := stateMap["observationProjectionOutcome"].(string); ok {
		result.ObservationProjectionOutcome = vo.ObservationProjectionOutcome(v)
	}
	if v, ok := stateMap["outerTaskDisposition"].(string); ok {
		result.OuterTaskDisposition = vo.OuterTaskDisposition(v)
	}
	if v, ok := stateMap["pythonInvocationOutcome"].(string); ok {
		result.PythonInvocationOutcome = vo.InvocationOutcome(v)
	}
	if v, ok := stateMap["advisorInvocationOutcome"].(string); ok {
		result.AdvisorInvocationOutcome = vo.InvocationOutcome(v)
	}
	if v, ok := stateMap["attempt"].(float64); ok {
		result.Attempt = int(v)
	}
	if v, ok := stateMap["maxAttempts"].(float64); ok {
		result.MaxAttempts = int(v)
	}

	return result
}

// applyGraphFailureDisposition 应用图谱失败处置
func (p *ValidationPhase) applyGraphFailureDisposition(ctx context.Context, document *entity.Document,
	task *entity.DocumentTask, planId int64, result *vo.GraphRagBuildResult, cause error) {
	failedStage := vo.TaskStageGraphRag
	if task.CurrentStage != 0 {
		failedStage = task.CurrentStage
	}
	if cause == nil {
		cause = &vo.GraphRagBuildFailureException{Result: result}
	}

	markFailureTx := func(txCtx context.Context) error {
		if err := p.deps.Repo.UpdateDocumentById(txCtx, &entity.Document{
			ID: document.ID, IndexStatus: vo.IndexStatusBuildFailed,
		}); err != nil {
			return err
		}
		if err := p.deps.Repo.UpdateChunkByTaskId(txCtx, &entity.DocumentChunk{
			TaskId: task.ID, VectorStatus: vo.VectorStatusVectorFailed,
		}); err != nil {
			return err
		}
		if err := p.deps.Repo.UpdateStepExecuteStatus(txCtx, planId, vo.StrategyExecuteStatusExecuteFailed); err != nil {
			return err
		}
		if err := p.deps.Repo.UpdateTaskById(txCtx, &entity.DocumentTask{
			ID: task.ID, TaskStatus: vo.TaskStatusFailed, CurrentStage: failedStage,
		}); err != nil {
			return err
		}
		failDetail, _ := json.Marshal(map[string]any{"error": cause.Error(), "currentStage": failedStage})
		failLog := &entity.DocumentTaskLog{
			TaskId: task.ID, DocumentId: task.DocumentId,
			StageType: failedStage, EventType: vo.TaskEventFailed,
			LogLevel: vo.LogLevelError, OperatorType: vo.OperatorTypeSystem,
			Content: "GraphRAG 构建失败", DetailJson: string(failDetail),
		}
		return p.deps.Repo.InsertTaskLog(txCtx, failLog)
	}
	if err := p.deps.Repo.Do(ctx, markFailureTx); err != nil {
		logx.Warnf("图谱失败时收尾失败: taskId=%d, err=%v", task.ID, err)
	}
}
