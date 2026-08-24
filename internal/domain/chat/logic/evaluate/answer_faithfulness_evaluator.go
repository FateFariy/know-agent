package evaluate

import (
	"errors"
	"fmt"
	"strings"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/chat/adapter"
	"github.com/swiftbit/know-agent/internal/domain/chat/adapter/model"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/enum"
)

// AnswerFaithfulnessEvaluator 答案忠实度评估器
// 思路：调用 LLM 将回答拆解为若干"声明(claim)"并逐条判定是否受检索上下文支持（JSON 输出）；
// 最终分数由本地计算：faithfulness = 受支持的声明数 / 声明总数。
type AnswerFaithfulnessEvaluator struct {
	baseEvaluator
}

func NewAnswerFaithfulnessEvaluator(llm model.ChatModel, promptRenderer adapter.PromptRenderer) *AnswerFaithfulnessEvaluator {
	return &AnswerFaithfulnessEvaluator{
		baseEvaluator: baseEvaluator{
			templateName:   enum.AnswerFaithfulnessEvaluate,
			promptRenderer: promptRenderer,
			llm:            llm,
		},
	}
}

func (e *AnswerFaithfulnessEvaluator) Name() string {
	return enum.AnswerFaithfulness
}

func (e *AnswerFaithfulnessEvaluator) validate(input *EvaluationInput) error {
	if input.Answer == "" {
		return errors.New("answer faithfulness评估需要answer字段")
	}
	return nil
}

func (e *AnswerFaithfulnessEvaluator) prepareVariables(input *EvaluationInput) map[string]any {
	return map[string]any{
		"question": input.Question,
		"contexts": strings.Join(input.Contexts, "\n"),
	}
}

// claimItem 忠实度评估的 LLM 输出结构
type claimItem struct {
	Claim     string `json:"claim"`
	Supported bool   `json:"supported"`
}

type faithfulnessOutput struct {
	Claims []claimItem `json:"claims"`
}

// computeScore 本地计算忠实度：受支持声明 / 总声明
func (e *AnswerFaithfulnessEvaluator) computeScore(_ *EvaluationInput, llmOutput string) (float64, error) {
	var out faithfulnessOutput
	if err := utils.Unmarshal(llmOutput, &out); err != nil {
		return 0, fmt.Errorf("解析忠实度评估结果失败: %w", err)
	}
	total := len(out.Claims)
	if total == 0 {
		// 无声明可判定（如回答极短）：视为完全忠实
		return 1.0, nil
	}
	supported := 0
	for _, c := range out.Claims {
		if c.Supported {
			supported++
		}
	}
	return float64(supported) / float64(total), nil
}
