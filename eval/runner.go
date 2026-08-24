package eval

import (
	"context"
	"sync"

	"github.com/swiftbit/know-agent/internal/domain/chat/adapter"
	"github.com/swiftbit/know-agent/internal/domain/chat/adapter/model"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/evaluate"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
	"github.com/swiftbit/know-agent/internal/infrastructure/port/emb"
)

// Evaluator 评测指标单元，直接复用 conversation.Evaluator 接口，
// 实现与线上"回答评估"阶段完全一致的计算逻辑（仅剥离了流式事件上报）。
type Evaluator = evaluate.Evaluator

// Runner 离线评测运行器。
// 仅保留"评估"这一必要阶段：按数据集逐条样本、并行执行各评估器，
// 汇总每个指标的平均分与逐样本明细。
type Runner struct {
	evaluators []Evaluator
}

// NewRunner 创建离线评测运行器。evaluators 一般来自
// conversation 包同款评估器（answer_faithfulness / answer_relevancy / context_precision / context_recall）
func NewRunner(evaluators ...Evaluator) *Runner {
	return &Runner{evaluators: evaluators}
}

// Run 对数据集执行评测，返回聚合报告
func (r *Runner) Run(ctx context.Context, dataset *Dataset) (*Report, error) {
	if len(dataset.Samples) == 0 {
		return nil, ErrEmptyDataset
	}
	if len(r.evaluators) == 0 {
		return nil, ErrNoEvaluator
	}

	// 初始化每个指标的得分收集器
	metricScores := make(map[string][]float64, len(r.evaluators))
	metricErrors := make(map[string]int, len(r.evaluators))
	for _, ev := range r.evaluators {
		metricScores[ev.Name()] = make([]float64, 0, len(dataset.Samples))
	}

	details := make([]SampleResult, len(dataset.Samples))

	var wg sync.WaitGroup
	for i := range dataset.Samples {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			input := dataset.Samples[idx].toInput()
			res := SampleResult{ID: dataset.Samples[idx].ID}

			for _, ev := range r.evaluators {
				score, err := ev.Evaluate(ctx, input)
				if err != nil {
					metricErrors[ev.Name()]++
					res.Scores = append(res.Scores, MetricScore{Name: ev.Name(), Score: 0, Err: err.Error()})
					metricScores[ev.Name()] = append(metricScores[ev.Name()], 0)
					continue
				}
				res.Scores = append(res.Scores, MetricScore{Name: ev.Name(), Score: score})
				metricScores[ev.Name()] = append(metricScores[ev.Name()], score)
			}
			details[idx] = res
		}(i)
	}
	wg.Wait()

	// 聚合每个指标的平均分
	metrics := make([]MetricSummary, 0, len(r.evaluators))
	for _, ev := range r.evaluators {
		name := ev.Name()
		scores := metricScores[name]
		metrics = append(metrics, MetricSummary{
			Name:        name,
			Mean:        mean(scores),
			ErrorCount:  metricErrors[name],
			SampleCount: len(scores),
		})
	}

	return &Report{
		DatasetName: dataset.Name,
		SampleCount: len(dataset.Samples),
		Metrics:     metrics,
		Details:     details,
	}, nil
}

func mean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func NewAnswerEvaluators(llm model.ChatModel, renderer adapter.PromptRenderer, emb *emb.Embedder) []evaluate.Evaluator {
	return []evaluate.Evaluator{
		evaluate.NewAnswerFaithfulnessEvaluator(llm, renderer),
		evaluate.NewAnswerRelevancyEvaluator(llm, renderer, emb),
		evaluate.NewContextPrecisionEvaluator(llm, renderer),
		evaluate.NewContextRecallEvaluator(llm, renderer),
	}
}

type MockSink struct {
}

func (m *MockSink) Text(content string, conversationId string, exchangeId int64) error {
	return nil
}

func (m *MockSink) Thinking(content string, conversationId string, exchangeId int64) error {
	return nil
}

func (m *MockSink) Status(content string, conversationId string, exchangeId int64) error {
	return nil
}

func (m *MockSink) Error(content string, conversationId string, exchangeId int64) error {
	return nil
}

func (m *MockSink) References(references []*vo.SearchReference, conversationId string, exchangeId int64) error {
	return nil
}

func (m *MockSink) Recommendations(recommendations []string, conversationId string, exchangeId int64) error {
	return nil
}

func (m *MockSink) Finish(conversationId string, exchangeId int64) error {
	return nil
}
