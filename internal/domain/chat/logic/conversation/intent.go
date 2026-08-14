package conversation

import (
	"context"

	"github.com/swiftbit/know-agent/internal/domain/chat/logic/intent"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/enum"
)

type IntentRecognizeStage struct {
	recognizer intent.Recognizer
}

func NewIntentRecognizeStage(recognizer intent.Recognizer) *IntentRecognizeStage {
	return &IntentRecognizeStage{
		recognizer: recognizer,
	}
}

func (i *IntentRecognizeStage) Name() string {
	return enum.ConversationTraceStageIntent.Name
}

func (i *IntentRecognizeStage) Execute(ctx context.Context, convCtx *Context) error {
	execPlan := convCtx.ExecutionPlan.Load()
	if execPlan == nil {
		return nil
	}
	input := intent.RecognitionInput{
		OriginalQuestion:         execPlan.OriginalQuestion,
		RewrittenQuestion:        execPlan.RewriteQuestion,
		SubQuestions:             execPlan.RewriteSubQuestions,
		HistorySummary:           execPlan.HistorySummary,
		RecentQuestionTranscript: execPlan.RecentQuestionTranscript,
	}
	result, _ := i.recognizer.Recognize(ctx, &input)

	return nil
}
