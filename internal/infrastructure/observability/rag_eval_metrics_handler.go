package observability

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/swiftbit/know-agent/common/logx"
	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/callbacks"
	"github.com/swiftbit/know-agent/internal/domain/chat/adapter"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/enum"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
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

// ragEvalMetricsHandler 实现 Handler 接口，用于记录 Prometheus 指标，以及持久化评估结果
type ragEvalMetricsHandler struct {
	repo adapter.ChatRepository
}

func RegisterRagEvalMetricsHandler(repo adapter.ChatRepository) {
	callbacks.AppendGlobalHandlers(&ragEvalMetricsHandler{
		repo: repo,
	})
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

	// 在线持久化评估结果到 chat_exchange_eval
	h.persistEval(ctx, metricName, score, info.StartTime, nil)

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

	// 在线持久化评估失败结果到 chat_exchange_eval
	h.persistEval(ctx, metricName, 0, info.StartTime, err)

	return ctx
}

// persistEval 将单次评估结果直接写入仓储（best-effort，失败仅告警不阻塞主流程）
func (h *ragEvalMetricsHandler) persistEval(ctx context.Context, metricName string, score float64, startTime time.Time, err error) {
	trace := vo.TraceFromCtx(ctx)
	conversationId := trace.ConversationId()
	exchangeId := trace.ExchangeId()
	if conversationId == "" || exchangeId == 0 {
		return
	}
	errMsg, status := "", 0
	if err != nil {
		errMsg = err.Error()
		status = 1
	}
	eval := &entity.ChatExchangeEval{
		ConversationId: conversationId,
		ExchangeId:     exchangeId,
		MetricName:     metricName,
		MetricLabel:    metricLabel(metricName),
		Score:          score,
		LatencyMs:      time.Since(startTime).Milliseconds(),
		Status:         int8(status),
		ErrorMsg:       utils.Pointer(errMsg),
	}
	if err = h.repo.InsertExchangeEval(context.Background(), []*entity.ChatExchangeEval{eval}); err != nil {
		logx.Warnf("RAG 评估结果落库失败, conversationId=%s, exchangeId=%d, metric=%s, err=%v",
			conversationId, exchangeId, metricName, err)
	}
}

// metricLabel 指标编码转展示名
func metricLabel(name string) string {
	switch name {
	case enum.AnswerFaithfulness:
		return "答案忠实度"
	case enum.AnswerRelevancy:
		return "答案相关性"
	case enum.ContextPrecision:
		return "上下文精度"
	default:
		return name
	}
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
