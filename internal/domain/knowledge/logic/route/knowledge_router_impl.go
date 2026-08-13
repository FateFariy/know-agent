package route

import (
	"context"
	"regexp"
	"sort"
	"strings"

	"github.com/duke-git/lancet/v2/maputil"
	"github.com/duke-git/lancet/v2/stream"
	"github.com/duke-git/lancet/v2/strutil"

	"github.com/swiftbit/know-agent/common/logx"
	"github.com/swiftbit/know-agent/common/utils"
	den "github.com/swiftbit/know-agent/internal/domain/document/model/entity"
	dvo "github.com/swiftbit/know-agent/internal/domain/document/model/vo"
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
		_ = ranker.Rank(ctx, rankCtx)
	}

	decision.Scopes = rankCtx.ScopeCandidates
	decision.Topics = rankCtx.TopicCandidates
	decision.Documents = rankCtx.DocumentCandidates
	decision.Confidence = r.computeConfidence(rankCtx.DocumentCandidates)

	// 设置决策来源（从文档候选中解析）
	decision.Source = r.resolveDecisionSource(rankCtx.DocumentCandidates)

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
		logx.Warnf("知识路由[shadow]失败: conversationId=%r, err=%v", conversationId, err)
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

// expandChineseNgrams 对中文短片段做 2~maxGram 的滑动窗口扩展（用于提高短实体召回）
func (r *KnowledgeRouteImpl) expandChineseNgrams(segment string, seen map[string]struct{}) {
	runes := []rune(segment)
	if len(runes) < 4 {
		return
	}
	maxGram := min(6, len(runes))
	for gram := 2; gram <= maxGram; gram++ {
		for start := 0; start+gram <= len(runes); start++ {
			g := string(runes[start : start+gram])
			seen[g] = struct{}{}
		}
	}
}

const (
	documentCandidateLimit = 5
	lowConfidenceThreshold = 0.55
)

// rankDocuments 对文档进行打分，并将 top-score 的文档返回
//
// 打分策略：语义分 + 词面辅助分 + 关键词命中分 + 与 topScope 匹配加分 + 主题关系加分
// 低置信度扩展：当置信度 < 阈值时，将允许范围内的文档作为零分候选补充
func (r *KnowledgeRouteImpl) rankDocuments(ctx context.Context, q *QueryContext) {
	documents, err := r.docGateway.FindRetrieveDocumentByIds(ctx, q.AllowedDocumentIds)
	if err != nil {
		logx.Warnf("查询可检索文档失败: %v", err)
		return
	}
	if len(documents) == 0 {
		return
	}

	// 加载文档画像
	profiles, err := r.profileLogic.GetAllProfiles(ctx)
	if err != nil {
		logx.Warnf("查询文档画像失败: %v", err)
	}
	profileByDoc := make(map[int64]*den.DocumentProfile, len(profiles))
	for _, p := range profiles {
		profileByDoc[p.DocumentId] = p
	}

	// 加载主题-文档关系
	relations, err := r.repo.SelectTopicDocumentRelations(ctx)
	if err != nil {
		logx.Warnf("查询 topic-document 关系失败: %v", err)
	}
	relationByTopic := make(map[string]map[int64]*entity.KnowledgeTopicDocumentRelation, len(relations))
	for _, rel := range relations {
		if _, ok := relationByTopic[rel.TopicCode]; !ok {
			relationByTopic[rel.TopicCode] = make(map[int64]*entity.KnowledgeTopicDocumentRelation)
		}
		relationByTopic[rel.TopicCode][rel.DocumentId] = rel
	}

	// 获取 top scope/topic 用于加分
	topScopeCode := ""
	if len(scopeCandidates) > 0 {
		topScopeCode = scopeCandidates[0].ScopeCode
	}
	topTopicCode := ""
	if len(topicCandidates) > 0 {
		topTopicCode = topicCandidates[0].TopicCode
	}

	// 为每个文档准备文本与语义分
	routeTexts := make([]string, 0, len(documents))
	for _, doc := range documents {
		routeTexts = append(routeTexts, r.buildDocumentRouteText(doc, profileByDoc[doc.DocumentId]))
	}
	semanticScores := r.computeSemanticScores(ctx, q, routeTexts)
	lexicalScores := r.searchDocumentLexicalScores(ctx, q.RoutingText, documentCandidateLimit)

	// 打分：语义分 + 词面辅助分 + 关键词命中分
	matched := make([]*vo.DocumentRouteCandidate, 0, len(documents))
	for i, doc := range documents {
		routeText := routeTexts[i]
		semantic := 0.0
		if i < len(semanticScores) {
			semantic = semanticScores[i]
		}
		lexical := lexicalScores[doc.DocumentId]
		score := r.semanticMainScore(semantic) + r.lexicalAssistScore(lexical) + r.keywordEntityMatchScore(q.QueryTerms, routeText)

		// 与 top scope 匹配加分
		if strutil.IsNotBlank(topScopeCode) && topScopeCode == doc.KnowledgeScopeCode {
			score += 15
		}
		// 主题关系加分
		topicRelationScore := 0.0
		if strutil.IsNotBlank(topTopicCode) {
			if relMap, ok := relationByTopic[topTopicCode]; ok {
				if rel, ok := relMap[doc.DocumentId]; ok && rel.RelationScore > 0 {
					topicRelationScore = rel.RelationScore
					score += topicRelationScore * 20
				}
			}
		}

		// 仅保留正分候选（零分候选在低置信度扩展时补充）
		if score <= 0 {
			continue
		}
		matched = append(matched, &vo.DocumentRouteCandidate{
			DocumentId:      doc.DocumentId,
			DocumentName:    doc.DocumentName,
			LastIndexTaskId: doc.LastIndexTaskId,
			Score:           score,
			Reason:          r.buildCandidateReason(semantic, lexical, topicRelationScore),
			Source:          r.resolveCandidateSource(semantic, lexical, topicRelationScore),
			Features:        r.buildFeatures(semantic, lexical, topicRelationScore),
		})
	}

	// 按得分降序排列
	sort.Slice(matched, func(i, j int) bool { return matched[i].Score > matched[j].Score })
	candidates := utils.Limit(matched, documentCandidateLimit)

	// 低置信度扩展：当无候选或置信度低时，补充允许范围内的文档
	if len(candidates) == 0 || (len(candidates) > 0 && r.computeConfidence(candidates) < lowConfidenceThreshold) {
		expanded := r.expandCandidatesWithAllowedScope(candidates, documents, q, documentCandidateLimit)
		if len(expanded) > len(candidates) {
			q.Diagnostics["LOW_CONFIDENCE_ALLOWED_SCOPE_EXPANSION"] = struct{}{}
		}
		candidates = expanded
	}

	return
}

// computeConfidence 从候选列表计算置信度：top1/(top1+top2+5) 归一化
func (r *KnowledgeRouteImpl) computeConfidence(candidates []*vo.DocumentRouteCandidate) float64 {
	if len(candidates) == 0 {
		return 0
	}
	top := candidates[0].Score
	second := 0.0
	if len(candidates) > 1 {
		second = candidates[1].Score
	}
	return top / max(10, top+second+5)
}

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
			DocumentId:         doc.DocumentId,
			DocumentName:       strutil.Trim(doc.DocumentName),
			LastIndexTaskId:    doc.LastIndexTaskId,
			KnowledgeScopeCode: strutil.Trim(doc.KnowledgeScopeCode),
			KnowledgeScopeName: strutil.Trim(doc.KnowledgeScopeName),
			BusinessCategory:   strutil.Trim(doc.BusinessCategory),
			DocumentTags:       strutil.Trim(doc.DocumentTags),
			Score:              0,
			Reason:             "未形成有效路由特征，按允许文档范围有界扩展候选池",
			Source:             "ALLOWED_DOCUMENT_SCOPE",
			Features:           map[string]float64{"allowedScopeFallback": 1},
		})
		seenIds[doc.DocumentId] = struct{}{}
	}
	return merged
}

// buildDocumentRouteText 拼接文档元数据 + 画像作为路由文本
func (r *KnowledgeRouteImpl) buildDocumentRouteText(doc *dvo.DocumentMetadata, profile *den.DocumentProfile) string {
	if profile == nil {
		return utils.JoinNonBlank(" ", doc.DocumentName, doc.KnowledgeScopeName, doc.KnowledgeScopeCode, doc.BusinessCategory, doc.DocumentTags)
	}
	return utils.JoinNonBlank(" ", doc.DocumentName, doc.KnowledgeScopeName, doc.KnowledgeScopeCode, doc.BusinessCategory, doc.DocumentTags,
		profile.DocumentSummary, profile.CoreTopics, profile.ExampleQuestions, profile.DocumentType)
}

// =====================================================
// 语义与词面打分辅助
// =====================================================

// computeSemanticScores 批量计算 routingText 与每个候选文本的余弦相似度；embedder 未配置时返回全 0 长度相同
func (r *KnowledgeRouteImpl) computeSemanticScores(ctx context.Context, q *QueryContext, routeTexts []string) []float64 {
	scores := make([]float64, len(routeTexts))
	if len(q.QueryEmbedding) == 0 || len(routeTexts) == 0 {
		return scores
	}
	if r.embedder == nil {
		q.Diagnostics["SEMANTIC_ROUTE_NOT_CONFIGURED"] = struct{}{}
		return scores
	}

	for start := 0; start < len(routeTexts); start += routeEmbeddingBatchSize {
		end := min(start+routeEmbeddingBatchSize, len(routeTexts))
		batch := routeTexts[start:end]
		embeddings, err := r.embedder.EmbedStrings(ctx, batch...)
		if err != nil {
			logx.Warnf("知识路由批量向量计算失败: batchStart=%d, size=%d, err=%v", start, len(batch), err)
			q.Diagnostics["SEMANTIC_CANDIDATE_EMBEDDING_UNAVAILABLE"] = struct{}{}
			return make([]float64, len(routeTexts))
		}
		if len(embeddings) != len(batch) {
			q.Diagnostics["SEMANTIC_CANDIDATE_EMBEDDING_INVALID"] = struct{}{}
			return make([]float64, len(routeTexts))
		}
		for idx, emb := range embeddings {
			scores[start+idx] = cosineSimilarity(q.QueryEmbedding, emb)
		}
	}
	return scores
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
	hits, err := r.lexicalIndex.Search(ctx, routingText, "document", size)
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

// normalizeCode 将字符串规范化为可做 code 的形式：非字母数字统一替换为下划线
func normalizeCode(value string) string {
	cleaned := normalize(value)
	if strutil.IsBlank(cleaned) {
		return ""
	}
	return normalizeCodeInvalidChar.ReplaceAllString(cleaned, "_")
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

// resolveDecisionSource 从文档候选中解析决策来源
func (r *KnowledgeRouteImpl) resolveDecisionSource(documentCandidates []*vo.DocumentRouteCandidate) string {
	if len(documentCandidates) == 0 {
		return "NONE"
	}
	sourceSet := make(map[string]struct{})
	for _, c := range documentCandidates {
		if c != nil && strutil.IsNotBlank(c.Source) {
			sourceSet[c.Source] = struct{}{}
		}
	}
	if len(sourceSet) == 1 {
		for s := range sourceSet {
			return s
		}
	}
	if len(sourceSet) > 1 {
		return "COMPOSITE"
	}
	return "NONE"
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
