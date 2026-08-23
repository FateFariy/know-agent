package observability

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/swiftbit/know-agent/internal/domain/callbacks"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/enum"
)

// ragEvaluationScore 是一个带标签的直方图指标构造器，
// 用于按评估指标名称（faithfulness/answer_relevancy/context_recall/context_precision）
// 和状态（success/failed）分别记录得分分布。
var (
	RagEvaluationScore = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:      "rag_evaluation_score",
			Help:      "RAG 系统各评估指标的得分分布（0.0~1.0）",
			Namespace: "rag",
			Subsystem: "eval",
			Buckets:   prometheus.LinearBuckets(0.0, 0.05, 21), // 0.0,0.05,...,1.0
		},
		[]string{"metric", "status"},
	)

	// FaithfulnessScore 忠实度得分直方图（独立指标，便于单独告警/看板）
	FaithfulnessScore = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:      "rag_faithfulness_score",
			Help:      "RAG 回答忠实度（Faithfulness）得分分布",
			Namespace: "rag",
			Subsystem: "eval",
			Buckets:   prometheus.LinearBuckets(0.0, 0.05, 21),
		},
	)

	// AnswerRelevancyScore 答案相关性得分直方图
	AnswerRelevancyScore = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:      "rag_answer_relevancy_score",
			Help:      "RAG 答案相关性（Answer Relevancy）得分分布",
			Namespace: "rag",
			Subsystem: "eval",
			Buckets:   prometheus.LinearBuckets(0.0, 0.05, 21),
		},
	)

	// ContextPrecisionScore 上下文精确度得分直方图
	ContextPrecisionScore = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:      "rag_context_precision_score",
			Help:      "RAG 上下文精确度（Context Precision）得分分布",
			Namespace: "rag",
			Subsystem: "rag",
			Buckets:   prometheus.LinearBuckets(0.0, 0.05, 21),
		},
	)
)

func init() {
	registerRagEvalMetricsHandler()
}

// ragEvalMetricsHandler 实现 Handler 接口，用于记录 Prometheus 指标
type ragEvalMetricsHandler struct{}

func registerRagEvalMetricsHandler() {
	callbacks.AppendGlobalHandlers(&ragEvalMetricsHandler{})
	prometheus.MustRegister(
		RagEvaluationScore,
		FaithfulnessScore,
		AnswerRelevancyScore,
		ContextPrecisionScore,
	)
	return
}

// OnStart 评估开始时调用
func (h *ragEvalMetricsHandler) OnStart(ctx context.Context, info *callbacks.RunInfo, input any) context.Context {
	return ctx
}

// OnEnd 评估成功完成时调用
func (h *ragEvalMetricsHandler) OnEnd(ctx context.Context, info *callbacks.RunInfo, output any) context.Context {
	if info.Component != "rag_eval_metrics" {
		return ctx
	}
	metricName := info.Payload.(string)
	if metricName == "" {
		return ctx
	}
	score := output.(float64)
	observeScore(metricName, score, nil)

	return ctx
}

// OnError 评估发生错误时调用
func (h *ragEvalMetricsHandler) OnError(ctx context.Context, info *callbacks.RunInfo, err error) context.Context {
	if info.Component != "rag_eval_metrics" {
		return ctx
	}

	metricName := info.Payload.(string)
	if metricName == "" {
		return ctx
	}
	observeScore(metricName, 0, err)

	return ctx
}

// observeScore 根据指标名称向对应的独立直方图指标记录一次成功得分，
// 并向带标签的 rag_evaluation_score 直方图记录（含 success/failed 状态），
//
// 设计说明：独立得分直方图（rag_faithfulness_score 等）仅观测成功得分，
// 避免失败调用以 0 值污染得分分布；失败计数可通过聚合直方图的
// rag_evaluation_score{metric="...",status="failed"} 获取。
func observeScore(metricName string, score float64, err error) {
	status := "success"
	if err != nil {
		status = "failed"
	}

	// 带标签的聚合直方图（成功记实际得分，失败记 0 仅用于计数）
	failScore := score
	if err != nil {
		failScore = 0
	}
	RagEvaluationScore.WithLabelValues(metricName, status).Observe(failScore)

	// 独立得分直方图仅记录成功得分
	if err != nil {
		return
	}
	switch metricName {
	case enum.AnswerFaithfulness:
		FaithfulnessScore.Observe(score)
	case enum.AnswerRelevancy:
		AnswerRelevancyScore.Observe(score)
	case enum.ContextPrecision:
		ContextPrecisionScore.Observe(score)
	}
}
