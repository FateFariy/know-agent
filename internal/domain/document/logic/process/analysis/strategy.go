package analysis

import (
	"context"
	"encoding/json"

	"github.com/cloudwego/eino/callbacks"

	"github.com/swiftbit/know-agent/internal/domain/document/adapter"
	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/document/model/enum"
)

// StrategyPhase 策略推荐阶段：推进任务到策略路由、生成推荐策略
type StrategyPhase struct {
	repo        adapter.DocumentRepository
	recommender StrategyRecommender
}

func NewStrategyPhase(repo adapter.DocumentRepository, recommender StrategyRecommender) *StrategyPhase {
	return &StrategyPhase{
		repo:        repo,
		recommender: recommender,
	}
}

func (p *StrategyPhase) Name() string {
	return "策略推荐阶段"
}

func (p *StrategyPhase) Execute(ctx context.Context, parseCtx *Context) (err error) {
	ctx = callbacks.OnStart(ctx, struct{}{})
	defer func() {
		if err != nil {
			callbacks.OnError(ctx, err)
		}
	}()

	// 推进任务阶段到"策略路由"
	if err = p.repo.UpdateTaskById(ctx, &entity.DocumentTask{
		ID: parseCtx.TaskId, CurrentStage: enum.TaskStageStrategyRoute,
	}); err != nil {
		return
	}

	// 写入"开始分析解析结果并生成推荐策略"日志
	strategyStartDetail, _ := json.Marshal(map[string]any{
		"structureNodeCount": len(parseCtx.StructureNodes),
		"charCount":          parseCtx.AnalysisResult.CharCount,
		"tokenCount":         parseCtx.AnalysisResult.TokenCount,
	})
	strategyStartLog := &entity.DocumentTaskLog{
		TaskId:       parseCtx.TaskId,
		DocumentId:   parseCtx.DocumentId,
		StageType:    enum.TaskStageStrategyRoute,
		EventType:    enum.TaskEventStart,
		LogLevel:     enum.LogLevelInfo,
		OperatorType: enum.OperatorTypeSystem,
		Content:      "开始分析解析结果并生成推荐策略。",
		DetailJson:   string(strategyStartDetail),
	}
	if err = p.repo.InsertTaskLog(ctx, strategyStartLog); err != nil {
		return
	}

	// 调用策略推荐器生成推荐方案
	planDraft, err := p.recommender.Recommend(ctx, parseCtx.Document, parseCtx.AnalysisResult)
	if err != nil {
		return
	}
	parseCtx.StrategyPlanDraft = planDraft

	callbacks.OnEnd(ctx, struct{}{})

	return nil
}
