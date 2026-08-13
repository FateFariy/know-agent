package route

import (
	"context"

	"github.com/duke-git/lancet/v2/strutil"

	"github.com/swiftbit/know-agent/common/logx"
	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/knowledge/adapter"
	"github.com/swiftbit/know-agent/internal/domain/knowledge/logic/route/rank"
	"github.com/swiftbit/know-agent/internal/domain/knowledge/logic/route/score"
	"github.com/swiftbit/know-agent/internal/domain/knowledge/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/knowledge/model/vo"
)

const (
	routeModeAuto          = "auto"
	routeModeShadow        = "shadow"
	lowConfidenceThreshold = 0.55
)

// KnowledgeRouteImpl 知识路由器实现：负责根据问题/改写问题匹配 scope/topic/document
type KnowledgeRouteImpl struct {
	repo       adapter.KnowledgeRepository
	docGateway adapter.DocumentGateway
	scorer     score.Scorer
	rankers    []Ranker
	*options
}

type options struct {
	embedder     adapter.Embedder
	lexicalIndex adapter.RouteLexicalIndex
}

type Option func(*options)

// NewKnowledgeRouteImpl 创建路由服务实例
func NewKnowledgeRouteImpl(repo adapter.KnowledgeRepository, docGateway adapter.DocumentGateway, opts ...Option) *KnowledgeRouteImpl {
	base := new(options)
	for _, opt := range opts {
		opt(base)
	}
	return &KnowledgeRouteImpl{
		repo:       repo,
		docGateway: docGateway,
		scorer:     score.NewDefaultScorer(),
		options:    base,
	}
}

// WithEmbeddingProvider 注册嵌入模型（可选）
func WithEmbeddingProvider(emb adapter.Embedder) Option {
	return func(o *options) {
		o.embedder = emb
	}
}

// WithLexicalIndex 注册词面索引（可选）
func WithLexicalIndex(index adapter.RouteLexicalIndex) Option {
	return func(o *options) {
		o.lexicalIndex = index
	}
}

// Route 根据问题执行知识路由，返回范围/主题/文档候选列表与置信度
func (r *KnowledgeRouteImpl) Route(ctx context.Context, routeCtx *RouteContext) (*vo.KnowledgeRouteDecision, error) {
	decision := &vo.KnowledgeRouteDecision{RouteStatus: vo.RouteStatusFailed}
	if routeCtx.RoutingText == "" {
		decision.Reason = "问题为空或无法提取有效关键词"
		return decision, nil
	}
	rankCtx := &rank.Context{
		RoutingText:              routeCtx.RoutingText,
		QueryEmbedding:           routeCtx.QueryEmbedding,
		SelectedKnowledgeBaseIds: routeCtx.SelectedKnowledgeBaseIds,
		AllowedDocumentIds:       routeCtx.AllowedDocumentIds,
		Diagnostics:              make(map[string]struct{}),
	}
	for _, ranker := range r.rankers {
		if err := ranker.Rank(ctx, rankCtx); err != nil {
			return nil, err
		}
	}

	decision.Scopes = rankCtx.ScopeCandidates
	decision.Topics = rankCtx.TopicCandidates
	decision.Documents = rankCtx.DocumentCandidates
	decision.Resolve(lowConfidenceThreshold)

	// 设置降级状态与原因
	decision.IsDegraded = len(routeCtx.Diagnostics) > 0
	decision.DegradedReasons = utils.MapKeys(routeCtx.Diagnostics)

	topDocName := ""
	if len(rankCtx.DocumentCandidates) > 0 {
		topDocName = rankCtx.DocumentCandidates[0].DocumentName
	}
	logx.Infof("知识范围路由完成: question='%s', rewriteQuestion='%s', scopeCount=%d, topicCount=%d, documentCount=%d, confidence=%.4f, source=%s, degraded=%v, topDocument='%r'",
		routeCtx.Question, routeCtx.RewriteQuestion, len(rankCtx.ScopeCandidates), len(rankCtx.TopicCandidates), len(rankCtx.DocumentCandidates),
		decision.Confidence, decision.Source, decision.IsDegraded, topDocName)
	return decision, nil
}

// RecordShadowRoute 记录影子路由结果（后台写入不影响主流程）
func (r *KnowledgeRouteImpl) RecordShadowRoute(ctx context.Context, exchangeId, documentId int64, conversationId, question, rewriteQuestion string) error {
	queryCtx := NewQueryContext(question, rewriteQuestion, nil, nil)
	decision, err := r.Route(ctx, queryCtx)
	if err != nil {
		logx.Warnf("知识路由[shadow]失败: conversationId=%s, err=%v", conversationId, err)
		return err
	}
	trace := r.buildTrace(exchangeId, conversationId, question, rewriteQuestion, routeModeShadow, decision)
	trace.SelectedDocumentId = documentId
	trace.HitSelectedDocument = decision.ResolveHitSelectedDocument(documentId)
	if err = r.repo.InsertKnowledgeRouteTrace(ctx, trace); err != nil {
		logx.Warnf("记录知识路由[shadow]失败: conversationId=%r, exchangeId=%d, err=%v", conversationId, exchangeId, err)
		return err
	}
	return nil
}

// RecordAutoRoute 记录自动路由结果
func (r *KnowledgeRouteImpl) RecordAutoRoute(ctx context.Context, exchangeId int64, conversationId, question, rewriteQuestion string, decision *vo.KnowledgeRouteDecision) error {
	trace := r.buildTrace(exchangeId, conversationId, question, rewriteQuestion, routeModeAuto, decision)
	if len(decision.Documents) > 0 {
		trace.SelectedDocumentId = decision.Documents[0].DocumentId
	}
	trace.HitSelectedDocument = decision.ResolveHitSelectedDocument(trace.SelectedDocumentId)
	if err := r.repo.InsertKnowledgeRouteTrace(ctx, trace); err != nil {
		logx.Warnf("记录知识路由[auto]失败: conversationId=%r, err=%v", conversationId, err)
		return err
	}
	return nil
}

// buildTrace 组装路由跟踪结构（不含选中文档与命中标记，由各路由模式补充）
func (r *KnowledgeRouteImpl) buildTrace(exchangeId int64, conversationId, question, rewriteQuestion, mode string, decision *vo.KnowledgeRouteDecision) *entity.KnowledgeRouteTrace {
	trace := &entity.KnowledgeRouteTrace{
		ConversationId:  conversationId,
		ExchangeId:      exchangeId,
		Question:        strutil.Trim(question),
		RewriteQuestion: strutil.Trim(rewriteQuestion),
		Mode:            mode,
	}
	if decision == nil {
		trace.RouteStatus = vo.RouteStatusCode(vo.RouteStatusFailed)
		return trace
	}
	trace.Confidence = decision.Confidence
	trace.RouteStatus = vo.RouteStatusCode(decision.RouteStatus)
	trace.ErrorMsg = strutil.Trim(decision.Reason)
	trace.TopScopesJson = utils.ToCompactJSON(decision.Scopes)
	trace.TopTopicsJson = utils.ToCompactJSON(decision.Topics)
	trace.TopDocumentsJson = utils.ToCompactJSON(decision.Documents)
	return trace
}
