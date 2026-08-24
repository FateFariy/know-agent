package evaluate

import (
	"context"
	"fmt"

	"github.com/swiftbit/know-agent/internal/domain/chat/adapter/model"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/conversation"
	"github.com/swiftbit/know-agent/internal/domain/document/adapter"
)

// ========== 基础评估器 ==========
type baseEvaluator struct {
	templateName   string
	promptRenderer adapter.PromptRenderer
	llm            model.ChatModel
}

// Name 评估器名称
func (e *baseEvaluator) Name() string {
	return ""
}

func (e *baseEvaluator) Evaluate(ctx context.Context, input *conversation.EvaluationInput) (float64, error) {
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
	response, err := e.llm.Generate(ctx, "", prompt, model.WithTemperature(0), model.WithFunction("judge"))
	if err != nil {
		return 0, fmt.Errorf("LLM调用失败: %w", err)
	}

	// 本地根据 LLM 的判断结果计算分数
	return e.computeScore(input, response)
}

// computeScore 由子类实现：基于 LLM 输出的判断结果在本地计算最终得分
func (e *baseEvaluator) computeScore(input *conversation.EvaluationInput, llmOutput string) (float64, error) {
	return 0, fmt.Errorf("computeScore 未实现")
}

func (e *baseEvaluator) validate(input *conversation.EvaluationInput) error {
	return nil
}

func (e *baseEvaluator) prepareVariables(input *conversation.EvaluationInput) map[string]any {
	return nil
}
