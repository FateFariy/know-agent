package conversation

import (
	"context"

	"github.com/swiftbit/know-agent/internal/domain/chat/logic/intent"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/enum"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
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
	ctx = vo.OnStart(ctx, enum.ConversationTraceStageIntent,
		enum.ChatQueryModeName(convCtx.ChatMode), &vo.StageInput{SummaryText: "正在分析用户意图。"})

	input := &intent.RecognitionInput{
		OriginalQuestion:         execPlan.OriginalQuestion,
		RewrittenQuestion:        execPlan.RewriteQuestion,
		SubQuestions:             execPlan.RewriteSubQuestions,
		HistorySummary:           execPlan.HistorySummary,
		RecentQuestionTranscript: execPlan.RecentQuestionTranscript,
	}
	result, err := i.recognizer.Recognize(ctx, input)
	if err != nil {
		ctx = vo.OnError(ctx, "意图分析失败。", err)
		return err
	}
	execPlan.RecognitionResult = result

	ctx = vo.OnEnd(ctx, &vo.StageOutput{SummaryText: "意图分析结果：", Snapshot: result})

	return nil
}
