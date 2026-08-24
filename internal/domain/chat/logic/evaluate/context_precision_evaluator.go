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

// ContextPrecisionEvaluator 上下文精度评估器
// 思路：调用 LLM 对检索到的若干上下文片段逐一判定是否与问题相关（JSON 输出）；
// 最终分数由本地按"排名加权精度"计算（RAGAS 定义），越靠前的相关片段得分越高。
type ContextPrecisionEvaluator struct {
	baseEvaluator
}

func NewContextPrecisionEvaluator(llm model.ChatModel, promptRenderer adapter.PromptRenderer) *ContextPrecisionEvaluator {
	return &ContextPrecisionEvaluator{
		baseEvaluator: baseEvaluator{
			templateName:   enum.ContextPrecisionEvaluate,
			promptRenderer: promptRenderer,
			llm:            llm,
		},
	}
}

func (e *ContextPrecisionEvaluator) Name() string {
	return enum.ContextPrecision
}

func (e *ContextPrecisionEvaluator) validate(input *EvaluationInput) error {
	if input.Question == "" {
		return errors.New("context precision评估需要question字段")
	}
	if len(input.Contexts) == 0 {
		return errors.New("context precision评估需要contexts字段")
	}
	return nil
}

func (e *ContextPrecisionEvaluator) prepareVariables(input *EvaluationInput) map[string]any {
	// 构建带索引的上下文
	var contextsWithIndex strings.Builder
	for i, ctx := range input.Contexts {
		contextsWithIndex.WriteString(fmt.Sprintf("[%d] %s\n", i+1, ctx))
	}

	return map[string]any{
		"question": input.Question,
		"contexts": contextsWithIndex.String(),
	}
}

// relevanceItem 上下文精度评估的 LLM 输出结构
type relevanceItem struct {
	Index    int  `json:"index"`
	Relevant bool `json:"relevant"`
}

type contextPrecisionOutput struct {
	Judgements []relevanceItem `json:"judgements"`
}

// computeScore 本地计算上下文精度（排名加权累积精度，RAGAS 定义）
// precision@k = (Σ_{i<=k} relevant@i) / k ；score = Σ_k(precision@k × relevant@k) / 总相关数
func (e *ContextPrecisionEvaluator) computeScore(input *EvaluationInput, llmOutput string) (float64, error) {
	var out contextPrecisionOutput
	if err := utils.Unmarshal(llmOutput, &out); err != nil {
		return 0, fmt.Errorf("解析上下文精度评估结果失败: %w", err)
	}
	judgements := make([]bool, 0, len(out.Judgements))
	for _, j := range out.Judgements {
		judgements = append(judgements, j.Relevant)
	}
	total := len(judgements)
	if total == 0 {
		return 0, fmt.Errorf("LLM 未输出任何相关性判定")
	}
	relevant := 0
	for _, ok := range judgements {
		if ok {
			relevant++
		}
	}
	if relevant == 0 {
		return 0, nil
	}

	var cumulative int
	var scoreSum float64
	for k, ok := range judgements {
		if ok {
			cumulative++
			precisionAtK := float64(cumulative) / float64(k+1)
			scoreSum += precisionAtK
		}
	}
	return scoreSum / float64(relevant), nil
}
