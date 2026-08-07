package analysis

import (
	"context"
	"encoding/json"
	"time"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/document/adapter"
	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/document/model/enum"
	"github.com/swiftbit/know-agent/internal/domain/document/model/vo"
)

// FinalizationPhase 收尾阶段：持久化策略方案、更新文档状态、标记任务完成
type FinalizationPhase struct {
	repo adapter.DocumentRepository
	port *adapter.DocumentPort
}

func NewFinalizationPhase(repo adapter.DocumentRepository, port *adapter.DocumentPort) *FinalizationPhase {
	return &FinalizationPhase{repo: repo, port: port}
}

func (p *FinalizationPhase) Name() string {
	return "收尾阶段"
}

func (p *FinalizationPhase) Execute(ctx context.Context, parseCtx *Context) error {
	planDraft := parseCtx.StrategyPlanDraft
	document := parseCtx.Document
	task := parseCtx.Task

	// 事务性持久化策略方案 → 更新文档状态 → 收尾任务 → 写入日志
	finalizeTx := func(txCtx context.Context) error {
		// 持久化方案和步骤
		planId, err := p.persistPlanAndSteps(txCtx, parseCtx)
		if err != nil {
			return err
		}
		parseCtx.PlanId = planId

		// 更新文档：解析成功、策略已推荐、统计信息
		structureNodeCount := len(parseCtx.SaveCtx.StructureNodes)
		updatedDoc := &entity.Document{
			ID:                  document.ID,
			ParseStatus:         enum.ParseStatusParseSuccess,
			StrategyStatus:      enum.StrategyStatusRecommended,
			CharCount:           parseCtx.AnalysisResult.CharCount,
			TokenCount:          parseCtx.AnalysisResult.TokenCount,
			StructureLevel:      parseCtx.AnalysisResult.StructureLevel,
			ContentQualityLevel: parseCtx.AnalysisResult.ContentQualityLevel,
			ParseTextPath:       parseCtx.SaveCtx.ParsedTextPath,
			ParseErrorMsg:       utils.Pointer(""),
			CurrentPlanId:       planId,
			LastParseTaskId:     parseCtx.TaskId,
			StructureNodeCount:  structureNodeCount,
		}
		if err = p.repo.UpdateDocumentById(txCtx, updatedDoc); err != nil {
			return err
		}

		// 标记任务成功完成
		taskUpdate := &entity.DocumentTask{
			ID:           task.ID,
			TaskStatus:   enum.TaskStatusSuccess,
			CurrentStage: enum.TaskStageStrategyRoute,
			FinishTime:   utils.Pointer(time.Now()),
			CostMillis:   time.Since(parseCtx.StartTime).Milliseconds(),
			ErrorCode:    utils.Pointer(""),
			ErrorMsg:     utils.Pointer(""),
		}
		if err = p.repo.UpdateTaskById(ctx, taskUpdate); err != nil {
			return err
		}

		// 写入"系统已生成推荐策略"日志
		recommendDetail, _ := json.Marshal(map[string]any{
			"planId":             planId,
			"strategySnapshot":   planDraft.StrategySnapshot,
			"parentStepCount":    len(planDraft.ParentSteps),
			"childStepCount":     len(planDraft.ChildSteps),
			"structureNodeCount": structureNodeCount,
			"recommendReason":    planDraft.RecommendReason,
			"costMills":          time.Since(parseCtx.StartTime).Milliseconds(),
		})
		recommendLog := &entity.DocumentTaskLog{
			TaskId:       parseCtx.TaskId,
			DocumentId:   parseCtx.DocumentId,
			StageType:    enum.TaskStageStrategyRoute,
			EventType:    enum.TaskEventComplete,
			LogLevel:     enum.LogLevelInfo,
			OperatorType: enum.OperatorTypeSystem,
			Content:      "系统已生成推荐策略",
			DetailJson:   string(recommendDetail),
		}
		return p.repo.InsertTaskLog(txCtx, recommendLog)
	}
	return p.repo.Do(ctx, finalizeTx)
}

// persistPlanAndSteps 持久化策略方案和步骤
func (p *FinalizationPhase) persistPlanAndSteps(ctx context.Context, parseCtx *Context) (int64, error) {
	planDraft := parseCtx.StrategyPlanDraft
	document := parseCtx.Document

	planId := utils.GetSnowflakeNextID()
	latestVersion, err := p.repo.SelectLatestPlanVersion(ctx, document.ID)
	if err != nil {
		return 0, err
	}

	// 构造并插入计划主体
	strategyPlan := &entity.DocumentStrategyPlan{
		ID:               planId,
		DocumentId:       document.ID,
		PlanVersion:      latestVersion + 1,
		PlanSource:       enum.PlanSourceSystemRecommend,
		PlanStatus:       enum.PlanStatusWaitConfirm,
		StrategyCount:    len(planDraft.ParentSteps) + len(planDraft.ChildSteps),
		StrategySnapshot: planDraft.StrategySnapshot,
		RecommendReason:  planDraft.RecommendReason,
	}
	if err = p.repo.InsertPlan(ctx, strategyPlan); err != nil {
		return 0, err
	}

	// 批量写入流水线步骤
	parentSteps := p.convertStepDraftsToEntities(planId, document, enum.PipelineTypeParent, planDraft)
	childSteps := p.convertStepDraftsToEntities(planId, document, enum.PipelineTypeChild, planDraft)
	steps := append(parentSteps, childSteps...)
	if err = p.repo.InsertStepBatch(ctx, steps); err != nil {
		return 0, err
	}

	return planId, nil
}

func (p *FinalizationPhase) convertStepDraftsToEntities(planId int64, document *entity.Document, pipelineType enum.PipelineType, draft *vo.DocumentStrategyPlanDraft) []*entity.DocumentStrategyStep {
	stepDrafts := draft.ParentSteps
	if pipelineType == enum.PipelineTypeChild {
		stepDrafts = draft.ChildSteps
	}
	steps := make([]*entity.DocumentStrategyStep, 0, len(stepDrafts))
	for orderIdx, step := range stepDrafts {
		steps = append(steps, &entity.DocumentStrategyStep{
			ID:              utils.GetSnowflakeNextID(),
			PlanId:          planId,
			DocumentId:      document.ID,
			PipelineType:    pipelineType,
			StepNo:          orderIdx + 1,
			StrategyType:    step.StrategyType,
			StrategyRole:    step.StrategyRole,
			SourceType:      step.SourceType,
			ExecuteStatus:   enum.StrategyExecuteStatusWaitExecute,
			RecommendReason: draft.RecommendReason,
		})
	}
	return steps
}
