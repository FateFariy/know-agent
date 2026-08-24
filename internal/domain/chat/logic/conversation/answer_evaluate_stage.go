package conversation

import (
	"context"
	"time"

	"github.com/swiftbit/know-agent/common/logx"
	"github.com/swiftbit/know-agent/internal/domain/callbacks"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/evaluate"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/enum"
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

	for _, evaluator := range a.evaluator {
		go func(evaluator evaluate.Evaluator) {
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
