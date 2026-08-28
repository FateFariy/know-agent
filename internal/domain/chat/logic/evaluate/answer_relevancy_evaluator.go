package evaluate

import (
	"context"
	"errors"
	"fmt"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/chat/adapter"
	"github.com/swiftbit/know-agent/internal/domain/chat/adapter/model"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/enum"
)

// Embedder 文本嵌入模型
type Embedder interface {
	// EmbedStrings 文本向量化
	EmbedStrings(ctx context.Context, texts ...string) ([][]float64, error)
}

// AnswerRelevancyEvaluator 答案相关性评估器
type AnswerRelevancyEvaluator struct {
	emb Embedder
	baseEvaluator
}

func NewAnswerRelevancyEvaluator(llm model.ChatModel, promptRenderer adapter.PromptRenderer, emb Embedder) *AnswerRelevancyEvaluator {
	a := &AnswerRelevancyEvaluator{
		emb: emb,
	}
	a.baseEvaluator = baseEvaluator{
		templateName:   enum.AnswerRelevancyEvaluate,
		promptRenderer: promptRenderer,
		llm:            llm,
		evaluator:      a,
	}
	return a
}

func (e *AnswerRelevancyEvaluator) Name() string {
	return enum.AnswerRelevancy
}

func (e *AnswerRelevancyEvaluator) validate(input *EvaluationInput) error {
	if input.Question == "" {
		return errors.New("answer relevancy评估需要question字段")
	}
	if input.Answer == "" {
		return errors.New("answer relevancy评估需要answer字段")
	}
	return nil
}

func (e *AnswerRelevancyEvaluator) prepareVariables(input *EvaluationInput) map[string]any {
	return map[string]any{
		"question": input.Question,
		"answer":   input.Answer,
	}
}

// computeScore 本地计算答案相关性：LLM 基于答案反推若干问题，再与原始问题做向量余弦相似度平均
func (e *AnswerRelevancyEvaluator) computeScore(input *EvaluationInput, llmOutput string) (float64, error) {
	var wrapper struct {
		GenerateQuestions []string `json:"generate_questions"`
	}
	if err := utils.Unmarshal(llmOutput, &wrapper); err != nil {
		return 0, err
	}
	if len(wrapper.GenerateQuestions) == 0 {
		return 0, fmt.Errorf("LLM 未生成反推问题")
	}
	embeddings, err := e.emb.EmbedStrings(context.Background(), append([]string{input.Question}, wrapper.GenerateQuestions...)...)
	if err != nil {
		return 0, err
	}
	totalScore := 0.0
	for i := 1; i < len(embeddings); i++ {
		totalScore += utils.CosineSimilarity(embeddings[0], embeddings[i])
	}
	return totalScore / float64(len(wrapper.GenerateQuestions)), nil
}
