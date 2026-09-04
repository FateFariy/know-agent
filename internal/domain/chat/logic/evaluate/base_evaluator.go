package evaluate

import (
	"context"
	"fmt"

	"github.com/swiftbit/know-agent/internal/domain/chat/adapter/model"
	"github.com/swiftbit/know-agent/internal/domain/document/adapter"
)

type evaluator interface {
	validate(input *EvaluationInput) error
	prepareVariables(input *EvaluationInput) map[string]any
	computeScore(input *EvaluationInput, llmOutput string) (float64, error)
}

// ========== 基础评估器 ==========
type baseEvaluator struct {
	templateName   string
	promptRenderer adapter.PromptRenderer
	llm            model.ChatModel
	evaluator
}

func (e *baseEvaluator) Evaluate(ctx context.Context, input *EvaluationInput) (float64, error) {
	// 验证输入
	if err := e.validate(input); err != nil {
		return 0, err
	}

	// 准备模板变量
	variables := e.prepareVariables(input)

	// 渲染提示词
	prompt, err := e.promptRenderer.Render(e.templateName, variables)
	if err != nil {
		return 0, fmt.Errorf("渲染提示词失败: %w", err)
	}

	// 调用LLM（仅做判断，分数由本地计算）
	response, err := e.llm.Generate(ctx, "", prompt, model.WithTemperature(0), model.WithFunction("judge"), model.WithThink(false))
	if err != nil {
		return 0, fmt.Errorf("LLM调用失败: %w", err)
	}

	// 本地根据 LLM 的判断结果计算分数
	return e.computeScore(input, response)
}
