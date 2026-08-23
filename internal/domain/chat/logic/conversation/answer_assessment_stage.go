package conversation

import (
	"context"

	"github.com/swiftbit/know-agent/internal/domain/chat/adapter/model"
	"github.com/swiftbit/know-agent/internal/domain/document/adapter"
)

type AnswerAssessmentStage struct {
	judge    model.ChatModel
	renderer adapter.PromptRenderer
}

func NewAnswerAssessmentStage(judge model.ChatModel, renderer adapter.PromptRenderer) *AnswerAssessmentStage {
	return &AnswerAssessmentStage{
		judge:    judge,
		renderer: renderer,
	}
}

func (a *AnswerAssessmentStage) Name() string {
	//TODO implement me
	panic("implement me")
}

func (a *AnswerAssessmentStage) Execute(ctx context.Context, convCtx *Context) error {

	//TODO implement me
	panic("implement me")
}
