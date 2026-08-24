package evaluate

import "context"

type EvaluationInput struct {
	Question    string   `json:"question"`     // 用户问题
	Contexts    []string `json:"contexts"`     // 检索到的上下文
	Answer      string   `json:"answer"`       // RAG系统的回答
	GroundTruth string   `json:"ground_truth"` // 标准参考答案
}

type EvaluationOutput struct {
	Metrics string
	Score   float64
}

type Evaluator interface {
	// Name 评估器名称
	Name() string

	// Evaluate 评估输入的上下文和回答, 返回评估分数
	Evaluate(ctx context.Context, input *EvaluationInput) (float64, error)
}
