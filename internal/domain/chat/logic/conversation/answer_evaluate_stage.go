package conversation

import (
	"context"
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

	//TODO implement me
	panic("implement me")
}
