package evaluate

import (
	"context"
	"errors"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/chat/adapter"
	"github.com/swiftbit/know-agent/internal/domain/chat/adapter/model"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/conversation"
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
	return &AnswerRelevancyEvaluator{
		baseEvaluator: baseEvaluator{
			templateName:   enum.AnswerRelevancyEvaluate,
			promptRenderer: promptRenderer,
			llm:            llm,
		},
		emb: emb,
	}
}

func (e *AnswerRelevancyEvaluator) Name() string {
	return enum.AnswerRelevancy
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

func (e *AnswerRelevancyEvaluator) parseAnswerRelevancyScore(question, output string) (float64, error) {
	var wrapper struct {
		GenerateQuestions []string `json:"generate_questions"`
	}
	if err := utils.Unmarshal(output, &wrapper); err != nil {
		return 0, err
	}
	totalScore := 0.0
	embeddings, err := e.emb.EmbedStrings(context.Background(), append([]string{question}, wrapper.GenerateQuestions...)...)
	if err != nil {
		return 0, err
	}
	for i := 1; i < len(embeddings); i++ {
		totalScore += utils.CosineSimilarity(embeddings[0], embeddings[i])
	}

	return totalScore / float64(len(wrapper.GenerateQuestions)), nil
}
