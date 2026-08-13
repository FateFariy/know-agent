package route

import (
	"context"
	"regexp"
	"strings"

	"github.com/duke-git/lancet/v2/strutil"

	"github.com/swiftbit/know-agent/common/logx"
	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/knowledge/adapter"
	"github.com/swiftbit/know-agent/internal/domain/knowledge/logic/route/rank"
	"github.com/swiftbit/know-agent/internal/domain/knowledge/logic/route/score"
	"github.com/swiftbit/know-agent/internal/domain/knowledge/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/knowledge/model/vo"
)

// 路由状态常量
const (
	routeEmbeddingBatchSize = 10
)

// 默认路由模式（用于跟踪）
const (
	routeModeAuto   = "auto"
	routeModeShadow = "shadow"
)

// 基础分隔符与规范化正则
var (
	alpNumPattern            = regexp.MustCompile(`[a-zA-Z0-9]`)
	tokenSplitPattern        = regexp.MustCompile(`[\s、，,；;：:（）()\-的和及与或]+`)
	normalizePattern         = regexp.MustCompile(`[\s>\x60*#_\-，,。；;：:（）()“”"'\\[]]+`)
	normalizeCodeInvalidChar = regexp.MustCompile(`[^a-z0-9]+`)
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
func (r *KnowledgeRouteImpl) Route(ctx context.Context, queryCtx *QueryContext) (*vo.KnowledgeRouteDecision, error) {
	decision := &vo.KnowledgeRouteDecision{RouteStatus: vo.RouteStatusFailed}
	if queryCtx.RoutingText == "" {
		decision.Reason = "问题为空或无法提取有效关键词"
		return decision, nil
	}
	rankCtx := &rank.Context{
		Question:                 queryCtx.Question,
		RewriteQuestion:          queryCtx.RewriteQuestion,
		RoutingText:              queryCtx.RoutingText,
		QueryEmbedding:           queryCtx.QueryEmbedding,
		SelectedKnowledgeBaseIds: nil,
		AllowedDocumentIds:       nil,
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
	decision.Confidence = decision.ResolveConfidence()
	decision.Source = decision.ResolveSource()

	// 设置降级状态与原因
	decision.IsDegraded = len(queryCtx.Diagnostics) > 0
	for reason := range queryCtx.Diagnostics {
		decision.DegradedReasons = append(decision.DegradedReasons, reason)
	}

	switch {
	case len(rankCtx.DocumentCandidates) == 0:
		decision.RouteStatus = vo.RouteStatusFailed
		decision.Reason = "没有找到可用候选文档"
	case decision.Confidence < lowConfidenceThreshold:
		decision.RouteStatus = vo.RouteStatusLowConfidence
	default:
		decision.RouteStatus = vo.RouteStatusSuccess
	}
	if decision.RouteStatus != vo.RouteStatusFailed {
		decision.Reason = r.resolveDecisionReason(rankCtx.DocumentCandidates, decision.Confidence)
	}

	topDocName := ""
	if len(rankCtx.DocumentCandidates) > 0 {
		topDocName = rankCtx.DocumentCandidates[0].DocumentName
	}
	logx.Infof("知识范围路由完成: question='%r', rewriteQuestion='%r', scopeCount=%d, topicCount=%d, documentCount=%d, confidence=%.4f, source=%r, degraded=%v, topDocument='%r'",
		strutil.Trim(queryCtx.Question), strutil.Trim(queryCtx.RewriteQuestion),
		len(rankCtx.ScopeCandidates), len(rankCtx.TopicCandidates), len(rankCtx.DocumentCandidates),
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

const (
	documentCandidateLimit = 5
	lowConfidenceThreshold = 0.55
)

// expandCandidatesWithAllowedScope 将允许范围内的文档作为零分候选补充
// 当置信度不足时，通过增加候选池提升召回率
func (r *KnowledgeRouteImpl) expandCandidatesWithAllowedScope(ranked []*vo.DocumentRouteCandidate, documents []*dvo.DocumentMetadata, q *QueryContext, limit int) []*vo.DocumentRouteCandidate {
	// 复制已排序的高分候选
	merged := make([]*vo.DocumentRouteCandidate, 0, limit)
	seenIds := make(map[int64]struct{}, limit)
	for _, c := range ranked {
		if len(merged) >= limit {
			break
		}
		if c != nil {
			merged = append(merged, c)
			seenIds[c.DocumentId] = struct{}{}
		}
	}

	// 允许文档ID过滤
	allowedIds := make(map[int64]struct{}, len(q.AllowedDocumentIds))
	for _, id := range q.AllowedDocumentIds {
		allowedIds[id] = struct{}{}
	}

	// 补充允许范围内的文档
	for _, doc := range documents {
		if len(merged) >= limit {
			break
		}
		if doc == nil || doc.DocumentId == 0 || doc.LastIndexTaskId == 0 {
			continue
		}
		if _, exists := seenIds[doc.DocumentId]; exists {
			continue
		}
		// 若有允许列表，仅补充允许范围内的文档
		if len(allowedIds) > 0 {
			if _, allowed := allowedIds[doc.DocumentId]; !allowed {
				continue
			}
		}
		merged = append(merged, &vo.DocumentRouteCandidate{
			DocumentId:      doc.DocumentId,
			DocumentName:    strutil.Trim(doc.DocumentName),
			LastIndexTaskId: doc.LastIndexTaskId,
			Score:           0,
			Reason:          "未形成有效路由特征，按允许文档范围有界扩展候选池",
			Source:          "ALLOWED_DOCUMENT_SCOPE",
			Features:        map[string]float64{"allowedScopeFallback": 1},
		})
		seenIds[doc.DocumentId] = struct{}{}
	}
	return merged
}

// singleSemanticScore 仅对单一文本做语义相似度（失败或未配置返回 0）
func (r *KnowledgeRouteImpl) singleSemanticScore(ctx context.Context, q *QueryContext, text string) float64 {
	if len(q.QueryEmbedding) == 0 || r.embedder == nil || utils.IsBlank(text) {
		return 0
	}
	vectors, _ := r.embedder.EmbedStrings(ctx, text)
	if len(vectors) == 0 {
		return 0
	}
	return cosineSimilarity(q.QueryEmbedding, vectors[0])
}

// searchDocumentLexicalScores 词面检索文档维度分数（同上）
func (r *KnowledgeRouteImpl) searchDocumentLexicalScores(ctx context.Context, routingText string, size int) map[int64]float64 {
	if r.lexicalIndex == nil {
		return nil
	}
	hits, err := r.lexicalIndex.Search(ctx, routingText, "document", size, nil)
	if err != nil || len(hits) == 0 {
		return nil
	}
	result := make(map[int64]float64, len(hits))
	for _, hit := range hits {
		result[hit.DocumentId] = hit.Score
	}
	return result
}

// normalize 归一化文本：去符号/空白/小写
func normalize(value string) string {
	if strutil.IsBlank(value) {
		return ""
	}
	return strings.ToLower(strutil.Trim(normalizePattern.ReplaceAllString(value, "")))
}

// buildReason 根据关键词命中和语义得分生成原因说明
func (r *KnowledgeRouteImpl) buildReason(queryTerms []string, content string, semanticScore float64) string {
	matched := make([]string, 0, 3)
	normalizedContent := normalize(content)
	for _, term := range queryTerms {
		if strings.Contains(normalizedContent, normalize(term)) {
			matched = append(matched, term)
			if len(matched) >= 3 {
				break
			}
		}
	}
	if len(matched) > 0 {
		return "命中关键词：" + strings.Join(matched, "、")
	}
	if semanticScore >= 0.55 {
		return "语义相似度高，基于文档画像与元数据召回"
	}
	if semanticScore >= 0.35 {
		return "语义相近，采用保守扩范围召回"
	}
	return "基于文档画像与元数据综合召回"
}

// resolveDecisionReason 根据候选与置信度生成决策原因
func (r *KnowledgeRouteImpl) resolveDecisionReason(documentCandidates []*vo.DocumentRouteCandidate, confidence float64) string {
	if len(documentCandidates) == 0 {
		return "没有找到可用候选文档"
	}
	top := documentCandidates[0]
	topReason := strutil.Trim(top.Reason)
	if topReason == "" {
		topReason = "未形成有效路由特征"
	}
	switch {
	case confidence >= 0.80:
		return topReason
	case confidence >= lowConfidenceThreshold:
		return topReason
	default:
		return topReason + "，低置信度，已进入保守扩范围候选"
	}
}

// =====================================================
// Trace 写入
// =====================================================

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
