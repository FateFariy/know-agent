package conversation

import (
	"context"
	"time"

	"github.com/swiftbit/know-agent/common/logx"
	"github.com/swiftbit/know-agent/internal/domain/callbacks"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/evaluate"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/enum"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
)

type AnswerEvaluateStage struct {
	evaluator []evaluate.Evaluator
}

func NewAnswerEvaluateStage(evaluator []evaluate.Evaluator) *AnswerEvaluateStage {
	return &AnswerEvaluateStage{
		evaluator: evaluator,
	}
}

func (a *AnswerEvaluateStage) Name() string {
	return enum.ConversationTraceStageAnswerEvaluate.Name
}

func (a *AnswerEvaluateStage) Order() int {
	return enum.ConversationTraceStageAnswerEvaluate.Order
}

// ShouldExecute 仅当非开放闲聊、执行计划与检索结果就绪且语义缓存未命中时执行
func (a *AnswerEvaluateStage) ShouldExecute(ctx context.Context, convCtx *Context) bool {
	if convCtx.ChatMode == enum.ChatQueryModeOpenChat {
		return false
	}
	execPlan := convCtx.ExecutionPlan.Load()
	if execPlan == nil || execPlan.RetrievalResult == nil {
		return false
	}
	return !convCtx.cache.IsCacheHit()
}

func (a *AnswerEvaluateStage) Execute(ctx context.Context, convCtx *Context) error {
	execPlan := convCtx.ExecutionPlan.Load()
	var contexts []string
	if execPlan != nil {
		contexts = execPlan.RetrievalResult.RetrievalContexts()
	}
	input := &evaluate.EvaluationInput{
		Question: convCtx.Question,
		Contexts: contexts,
		Answer:   convCtx.Answer(),
	}
	ctx = vo.WithTrace(context.Background(), convCtx.Trace)
	for _, evaluator := range a.evaluator {
		go func(evaluator evaluate.Evaluator) {
			info := &callbacks.RunInfo{
				StartTime: time.Now(),
				Component: "rag_eval_metrics",
				Payload:   evaluator.Name(),
			}
			valueCtx := callbacks.EnsureRunInfo(ctx, info)
			valueCtx = callbacks.OnStart(valueCtx, struct{}{})
			score, err := evaluator.Evaluate(valueCtx, input)
			if err != nil {
				logx.Warnf("evaluate error: %v", err)
				callbacks.OnError(valueCtx, err)
				return
			}
			callbacks.OnEnd(valueCtx, score)
		}(evaluator)
	}
	return nil
}
