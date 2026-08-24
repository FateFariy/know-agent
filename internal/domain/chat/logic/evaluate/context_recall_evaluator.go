package evaluate

import (
	"errors"
	"fmt"
	"strings"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/chat/adapter"
	"github.com/swiftbit/know-agent/internal/domain/chat/adapter/model"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/conversation"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/enum"
)

// ContextRecallEvaluator 上下文召回率评估器
// 思路：调用 LLM 将标准参考答案（ground_truth）拆解为若干独立句子，
// 并逐句判定检索到的上下文是否包含能够支撑该句子的信息（JSON 输出）；
// 最终分数由本地计算：context_recall = 被支持的句子数 / 句子总数（RAGAS 定义）。
type ContextRecallEvaluator struct {
	baseEvaluator
}

func NewContextRecallEvaluator(llm model.ChatModel, promptRenderer adapter.PromptRenderer) *ContextRecallEvaluator {
	return &ContextRecallEvaluator{
		baseEvaluator: baseEvaluator{
			templateName:   enum.ContextRecallEvaluate,
			promptRenderer: promptRenderer,
			llm:            llm,
		},
	}
}

func (e *ContextRecallEvaluator) Name() string {
	return enum.ContextRecall
}

func (e *ContextRecallEvaluator) validate(input *conversation.EvaluationInput) error {
	if input.Question == "" {
		return errors.New("context recall评估需要question字段")
	}
	if len(input.Contexts) == 0 {
		return errors.New("context recall评估需要contexts字段")
	}
	if input.GroundTruth == "" {
		return errors.New("context recall评估需要ground_truth字段")
	}
	return nil
}

func (e *ContextRecallEvaluator) prepareVariables(input *conversation.EvaluationInput) map[string]any {
	return map[string]any{
		"question":     input.Question,
		"contexts":     strings.Join(input.Contexts, "\n"),
		"ground_truth": input.GroundTruth,
	}
}

// recallSentence 上下文召回率评估的 LLM 输出结构
type recallSentence struct {
	Sentence  string `json:"sentence"`
	Supported bool   `json:"supported"`
}

type contextRecallOutput struct {
	Sentences []recallSentence `json:"sentences"`
}

// computeScore 本地计算上下文召回率：被支持的句子数 / 句子总数
func (e *ContextRecallEvaluator) computeScore(_ *conversation.EvaluationInput, llmOutput string) (float64, error) {
	var out contextRecallOutput
	if err := utils.Unmarshal(llmOutput, &out); err != nil {
		return 0, fmt.Errorf("解析上下文召回率评估结果失败: %w", err)
	}
	total := len(out.Sentences)
	if total == 0 {
		// 参考答案无可拆解句子（如为空或极短）：视为完全召回
		return 1.0, nil
	}
	supported := 0
	for _, s := range out.Sentences {
		if s.Supported {
			supported++
		}
	}
	return float64(supported) / float64(total), nil
}
