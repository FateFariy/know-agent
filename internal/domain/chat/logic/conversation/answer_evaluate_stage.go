package conversation

import (
	"context"
	"time"

	"github.com/swiftbit/know-agent/common/logx"
	"github.com/swiftbit/know-agent/internal/domain/callbacks"
)

type AnswerEvaluateStage struct {
	evaluator []Evaluator
}

func NewAnswerEvaluateStage(evaluator []Evaluator) *AnswerEvaluateStage {
	return &AnswerEvaluateStage{
		evaluator: evaluator,
	}
}

func (a *AnswerEvaluateStage) Name() string {
	return "AnswerEvaluateStage"
}

func (a *AnswerEvaluateStage) Execute(ctx context.Context, convCtx *Context) error {
	execPlan := convCtx.ExecutionPlan.Load()
	var contexts []string
	if execPlan != nil {
		contexts = execPlan.RetrievalResult.RetrievalContexts()
	}
	input := &EvaluationInput{
		Question: convCtx.Question,
		Contexts: contexts,
		Answer:   convCtx.Answer(),
	}

	for _, evaluator := range a.evaluator {
		go func(evaluator Evaluator) {
			info := &callbacks.RunInfo{
				StartTime: time.Now(),
				Component: "rag_eval_metrics",
				Payload:   evaluator.Name(),
			}
			ctx = callbacks.EnsureRunInfo(ctx, info)
			ctx = callbacks.OnStart(ctx, struct{}{})
			score, err := evaluator.Evaluate(ctx, input)
			if err != nil {
				logx.Warnf("evaluate error: %v", err)
				callbacks.OnError(ctx, err)
				return
			}
			callbacks.OnEnd(ctx, score)
		}(evaluator)
	}
	return nil
}
