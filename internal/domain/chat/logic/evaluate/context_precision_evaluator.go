package evaluate

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/swiftbit/know-agent/internal/domain/chat/adapter/model"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/conversation"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/enum"
	"github.com/swiftbit/know-agent/internal/domain/document/adapter"
)

// ContextPrecisionEvaluator 上下文精确度评估器
type ContextPrecisionEvaluator struct {
	baseEvaluator
}

func NewContextPrecisionEvaluator(llm model.ChatModel, promptRenderer adapter.PromptRenderer) *ContextPrecisionEvaluator {
	return &ContextPrecisionEvaluator{
		baseEvaluator: baseEvaluator{
			templateName:   enum.ContextPrecisionEvaluate,
			promptRenderer: promptRenderer,
			llm:            llm,
			scoreParser:    parseContextPrecisionScore,
		},
	}
}

func (e *ContextPrecisionEvaluator) validate(input *conversation.EvaluationInput) error {
	if input.Question == "" {
		return errors.New("context precision评估需要question字段")
	}
	if len(input.Contexts) == 0 {
		return errors.New("context precision评估需要contexts字段")
	}
	if input.Answer == "" {
		return errors.New("context precision评估需要answer字段")
	}
	return nil
}

func (e *ContextPrecisionEvaluator) prepareVariables(input *conversation.EvaluationInput) map[string]any {
	// 构建带索引的上下文
	var contextsWithIndex strings.Builder
	for i, ctx := range input.Contexts {
		contextsWithIndex.WriteString(fmt.Sprintf("[位置%d] %s\n", i+1, ctx))
	}

	return map[string]any{
		"question":            input.Question,
		"contexts_with_index": contextsWithIndex.String(),
		"answer":              input.Answer,
	}
}

func parseContextPrecisionScore(output string) (float64, error) {
	re := regexp.MustCompile(`CONTEXT_PRECISION_SCORE:\s*(0\.\d+)`)
	matches := re.FindStringSubmatch(output)
	if len(matches) < 2 {
		return 0, errors.New("无法解析CONTEXT_PRECISION_SCORE")
	}
	return strconv.ParseFloat(matches[1], 64)
}
