package evaluate

import (
	"errors"
	"regexp"
	"strconv"
	"strings"

	"github.com/swiftbit/know-agent/internal/domain/chat/adapter/model"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/conversation"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/enum"
	"github.com/swiftbit/know-agent/internal/domain/document/adapter"
)

// FaithfulnessEvaluator 忠实度评估器
type FaithfulnessEvaluator struct {
	baseEvaluator
}

func NewFaithfulnessEvaluator(llm model.ChatModel, promptRenderer adapter.PromptRenderer) *FaithfulnessEvaluator {
	return &FaithfulnessEvaluator{
		baseEvaluator: baseEvaluator{
			templateName:   enum.AnswerFaithfulnessEvaluate,
			promptRenderer: promptRenderer,
			llm:            llm,
		},
	}
}

func (e *FaithfulnessEvaluator) Name() string {
	return enum.AnswerFaithfulness
}

func (e *FaithfulnessEvaluator) validate(input *conversation.EvaluationInput) error {
	if input.Question == "" {
		return errors.New("faithfulness评估需要question字段")
	}
	if len(input.Contexts) == 0 {
		return errors.New("faithfulness评估需要contexts字段")
	}
	if input.Answer == "" {
		return errors.New("faithfulness评估需要answer字段")
	}
	return nil
}

func (e *FaithfulnessEvaluator) prepareVariables(input *conversation.EvaluationInput) map[string]any {
	return map[string]any{
		"question": input.Question,
		"contexts": strings.Join(input.Contexts, "\n"),
		"answer":   input.Answer,
	}
}

func (e *FaithfulnessEvaluator) parseFaithfulnessScore(question, output string) (float64, error) {
	re := regexp.MustCompile(`FAITHFULNESS_SCORE:\s*(0\.\d+)`)
	matches := re.FindStringSubmatch(output)
	if len(matches) < 2 {
		return 0, errors.New("无法解析FAITHFULNESS_SCORE")
	}
	return strconv.ParseFloat(matches[1], 64)
}
