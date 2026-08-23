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
	scoreParser    func(string) (float64, error)
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

	// 调用LLM
	response, err := e.llm.Generate(ctx, "", prompt)
	if err != nil {
		return 0, fmt.Errorf("LLM调用失败: %w", err)
	}

	// 解析分数
	return e.scoreParser(response)
}

func (e *baseEvaluator) validate(input *conversation.EvaluationInput) error {
	return nil
}

func (e *baseEvaluator) prepareVariables(input *conversation.EvaluationInput) map[string]any {
	return nil
}
