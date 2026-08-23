package evaluate

import (
	"errors"
	"regexp"
	"strconv"

	"github.com/swiftbit/know-agent/internal/domain/chat/adapter"
	"github.com/swiftbit/know-agent/internal/domain/chat/adapter/model"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/conversation"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/enum"
)

// AnswerRelevancyEvaluator 答案相关性评估器
type AnswerRelevancyEvaluator struct {
	baseEvaluator
}

func NewAnswerRelevancyEvaluator(promptRenderer adapter.PromptRenderer, llm model.ChatModel) *AnswerRelevancyEvaluator {
	return &AnswerRelevancyEvaluator{
		baseEvaluator: baseEvaluator{
			templateName:   enum.AnswerRelevancyEvaluate,
			promptRenderer: promptRenderer,
			llm:            llm,
			scoreParser:    parseAnswerRelevancyScore,
		},
	}
}

func (e *AnswerRelevancyEvaluator) validate(input *conversation.EvaluationInput) error {
	if input.Question == "" {
		return errors.New("answer relevancy评估需要question字段")
	}
	if input.Answer == "" {
		return errors.New("answer relevancy评估需要answer字段")
	}
	return nil
}

func (e *AnswerRelevancyEvaluator) prepareVariables(input *conversation.EvaluationInput) map[string]any {
	return map[string]any{
		"question": input.Question,
		"answer":   input.Answer,
	}
}

func parseAnswerRelevancyScore(output string) (float64, error) {
	re := regexp.MustCompile(`ANSWER_RELEVANCY_SCORE:\s*(0\.\d+)`)
	matches := re.FindStringSubmatch(output)
	if len(matches) < 2 {
		return 0, errors.New("无法解析ANSWER_RELEVANCY_SCORE")
	}
	return strconv.ParseFloat(matches[1], 64)
}
