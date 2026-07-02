package logic

import (
	"context"
	"encoding/json"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/duke-git/lancet/v2/maputil"
	"github.com/duke-git/lancet/v2/strutil"
	"github.com/zeromicro/go-zero/core/logx"

	"github.com/swiftbit/know-agent/common/utils"
	cvo "github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
	"github.com/swiftbit/know-agent/internal/domain/knowledge/adapter"
	"github.com/swiftbit/know-agent/internal/domain/knowledge/model/vo"
)

// 路由状态常量
const (
	RouteStatusSuccess       = "SUCCESS"
	RouteStatusLowConfidence = "LOW_CONFIDENCE"
	RouteStatusFailed        = "FAILED"
	routeEmbeddingBatchSize  = 10
)

// 默认路由模式（用于跟踪）
const (
	routeModeAuto   = "auto"
	routeModeShadow = "shadow"
)

// 基础分隔符与规范化正则
var (
	alnumPattern             = regexp.MustCompile(`[a-zA-Z0-9]`)
	tokenSplitPattern        = regexp.MustCompile(`[\s、，,；;：:（）()\-的和及与或]+`)
	normalizePattern         = regexp.MustCompile(`[\s>\x60*#_\-，,。；;：:（）()“”"'\\[]]+`)
	normalizeCodeInvalidChar = regexp.MustCompile(`[^a-z0-9]+`)
)

// KnowledgeRouteLogicImpl 知识路由服务实现：负责根据问题/改写问题匹配 scope/topic/document
type KnowledgeRouteLogicImpl struct {
	repo         adapter.KnowledgeRepository
	embedder     adapter.Embedder
	lexicalIndex adapter.RouteLexicalIndex
}

// NewKnowledgeRouteLogic 创建路由服务实例
func NewKnowledgeRouteLogic(repo adapter.KnowledgeRepository) *KnowledgeRouteLogicImpl {
	return &KnowledgeRouteLogicImpl{repo: repo}
}

// WithEmbeddingProvider 注册嵌入模型（可选）
func (s *KnowledgeRouteLogicImpl) WithEmbeddingProvider(emb adapter.Embedder) *KnowledgeRouteLogicImpl {
	s.embedder = emb
	return s
}

// WithLexicalIndex 注册词面索引（可选）
func (s *KnowledgeRouteLogicImpl) WithLexicalIndex(index adapter.RouteLexicalIndex) *KnowledgeRouteLogicImpl {
	s.lexicalIndex = index
	return s
}

// Route 根据问题执行知识路由，返回范围/主题/文档候选列表与置信度
func (s *KnowledgeRouteLogicImpl) Route(ctx context.Context, question, rewriteQuestion string) (*vo.KnowledgeRouteDecision, error) {
	queryCtx := s.buildQueryContext(ctx, question, rewriteQuestion)
	decision := &vo.KnowledgeRouteDecision{RouteStatus: RouteStatusFailed}
	if len(queryCtx.QueryTerms) == 0 {
		decision.Reason = "问题为空或无法提取有效关键词"
		return decision, nil
	}

	scopeCandidates := s.rankScopes(ctx, queryCtx)
	topicCandidates := s.rankTopics(ctx, queryCtx, scopeCandidates)
	documentCandidates := s.rankDocuments(ctx, queryCtx, scopeCandidates, topicCandidates)

	decision.Scopes = scopeCandidates
	decision.Topics = topicCandidates
	decision.Documents = documentCandidates
	decision.Confidence = s.resolveConfidence(documentCandidates)

	switch {
	case len(documentCandidates) == 0:
		decision.RouteStatus = RouteStatusFailed
	case decision.Confidence < 0.55:
		decision.RouteStatus = RouteStatusLowConfidence
	default:
		decision.RouteStatus = RouteStatusSuccess
	}
	decision.Reason = s.resolveDecisionReason(documentCandidates, decision.Confidence)

	logx.Infof("知识范围路由完成: question='%s', scopeCount=%d, topicCount=%d, documentCount=%d, confidence=%.4f",
		strutil.Trim(question), len(scopeCandidates), len(topicCandidates), len(documentCandidates))
	return decision, nil
}

// RecordShadowRoute 记录影子路由结果（后台写入不影响主流程）
func (s *KnowledgeRouteLogicImpl) RecordShadowRoute(ctx context.Context, conversationId, exchangeId string, documentId int64, question, rewriteQuestion string) error {
	decision, err := s.Route(ctx, question, rewriteQuestion)
	if err != nil {
		Warnf("知识路由[shadow]失败: conversationId=%s, err=%v", conversationId, err)
		return err
	}
	trace := s.buildTrace(conversationId, exchangeId, question, rewriteQuestion, routeModeShadow, decision)
	trace.SelectedDocumentId = documentId
	trace.HitSelectedDocument = s.resolveHitSelectedDocument(documentId, decision)
	if err := s.repo.SaveKnowledgeRouteTrace(ctx, trace); err != nil {
		Warnf("记录知识路由[shadow]失败: conversationId=%s, err=%v", conversationId, err)
		return err
	}
	return nil
}

// RecordAutoRoute 记录自动路由结果
func (s *KnowledgeRouteLogicImpl) RecordAutoRoute(ctx context.Context, conversationId, exchangeId string, question, rewriteQuestion string, decision *vo.KnowledgeRouteDecision) error {
	trace := s.buildTrace(conversationId, exchangeId, question, rewriteQuestion, routeModeAuto, decision)
	if len(decision.Documents) > 0 && decision.Documents[0].DocumentId != "" {
		if id, parseErr := strconv.ParseInt(decision.Documents[0].DocumentId, 10, 64); parseErr == nil {
			trace.SelectedDocumentId = id
		}
	}
	trace.HitSelectedDocument = s.resolveHitSelectedDocument(trace.SelectedDocumentId, decision)
	if err := s.repo.SaveKnowledgeRouteTrace(ctx, trace); err != nil {
		logx.Warnf("记录知识路由[auto]失败: conversationId=%s, err=%v", conversationId, err)
		return err
	}
	return nil
}

// =====================================================
// 上下文构造
// =====================================================

// routeQueryContext 聚合路由上下文
type routeQueryContext struct {
	OriginalQuestion string
	RewriteQuestion  string
	RoutingText      string
	QueryTerms       []string
	QueryEmbedding   []float64
}

func (q *routeQueryContext) semanticEnabled() bool {
	return q != nil && len(q.QueryEmbedding) > 0
}

// buildQueryContext 组装路由上下文：拼接检索文本 + 分词 + 可选生成向量
func (s *KnowledgeRouteLogicImpl) buildQueryContext(ctx context.Context, question, rewriteQuestion string) *routeQueryContext {
	routingText := s.buildRoutingText(question, rewriteQuestion)
	terms := s.tokenize(routingText)

	// 若外部 embedding 未配置，则回退到纯词面
	var queryEmbedding []float64
	if s.embedder != nil && strutil.IsNotBlank(routingText) {
		if vectors, err := s.embedder.EmbedStrings(ctx, routingText); err == nil && len(vectors) > 0 {
			queryEmbedding = vectors[0]
		} else if err != nil {
			Warnf("知识路由生成向量失败，退回词面匹配: err=%v", err)
		}
	}

	return &routeQueryContext{
		OriginalQuestion: strutil.Trim(question),
		RewriteQuestion:  strutil.Trim(rewriteQuestion),
		RoutingText:      routingText,
		QueryTerms:       terms,
		QueryEmbedding:   queryEmbedding,
	}
}

// buildRoutingText 将原始问题与改写文本拼接；两文本相同则返回其一
func (s *KnowledgeRouteLogicImpl) buildRoutingText(question, rewriteQuestion string) string {
	original := strutil.Trim(question)
	rewritten := strutil.Trim(rewriteQuestion)
	if strutil.IsBlank(original) {
		return rewritten
	}
	if strutil.IsBlank(rewritten) || original == rewritten {
		return original
	}
	return original + " " + rewritten
}

// tokenize 分词：按中英文常见分隔符分割，再对长度 >=4 的中文片段进行 n-gram 扩展
func (s *KnowledgeRouteLogicImpl) tokenize(text string) []string {
	cleaned := strutil.Trim(text)
	if strutil.IsBlank(cleaned) {
		return nil
	}

	seen := make(map[string]struct{})
	for _, part := range tokenSplitPattern.Split(cleaned, -1) {
		trimmed := strutil.Trim(part)
		if utf8.RuneCountInString(trimmed) > 1 {
			seen[trimmed] = struct{}{}
			s.expandChineseNgrams(trimmed, seen)
		}
	}

	// 限制最大关键词数量
	terms := maputil.Keys(seen)
	return terms[:min(40, len(terms))]
}

// expandChineseNgrams 对中文短片段做 2~maxGram 的滑动窗口扩展（用于提高短实体召回）
func (s *KnowledgeRouteLogicImpl) expandChineseNgrams(segment string, seen map[string]struct{}) {
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

// =====================================================
// 评分与候选构造
// =====================================================

// rankScopes 对 scope 节点打分：语义分 + 词面辅助 + 关键词命中
func (s *KnowledgeRouteLogicImpl) rankScopes(ctx context.Context, q *routeQueryContext) []*vo.ScopeRouteCandidate {
	nodes, err := s.repo.SelectKnowledgeScopeNodes(ctx)
	if err != nil {
		Warnf("查询 scope 节点失败: %v", err)
	}
	if len(nodes) == 0 {
		return s.deriveScopesFromDocuments(ctx, q)
	}

	// 生成 routeText 列表
	routeTexts := make([]string, 0, len(nodes))
	for _, node := range nodes {
		routeTexts = append(routeTexts, s.joinNonBlank(node.ScopeName, node.Description, node.Aliases, node.Examples))
	}
	semanticScores := s.computeSemanticScores(ctx, q, routeTexts)
	lexicalScores := s.searchLexicalScores(ctx, q.RoutingText, "scope", 5)

	// 组装候选
	candidates := make([]*vo.ScopeRouteCandidate, 0, len(nodes))
	for i, node := range nodes {
		semantic := 0.0
		if i < len(semanticScores) {
			semantic = semanticScores[i]
		}
		lexical := 0.0
		if v, ok := lexicalScores[node.ScopeCode]; ok {
			lexical = v
		}
		finalScore := s.semanticMainScore(semantic) + s.lexicalAssistScore(lexical) +
			s.keywordEntityMatchScore(q.QueryTerms, routeTexts[i])
		if finalScore > 0 || semantic > 0 {
			candidates = append(candidates, &vo.ScopeRouteCandidate{
				ScopeCode: node.ScopeCode,
				ScopeName: node.ScopeName,
				Score:     finalScore,
				Reason:    s.buildReason(q.QueryTerms, routeTexts[i], semantic),
			})
		}
	}

	sort.Slice(candidates, func(i, j int) bool { return candidates[i].Score > candidates[j].Score })
	limit := 5
	if len(candidates) > limit {
		candidates = candidates[:limit]
	}
	return candidates
}

// deriveScopesFromDocuments 当没有配置 scope 节点时，从文档元数据派生粗略的 scope 候选
func (s *KnowledgeRouteLogicImpl) deriveScopesFromDocuments(ctx context.Context, q *routeQueryContext) []*vo.ScopeRouteCandidate {
	docs, err := s.repo.SelectRetrievableDocuments(ctx)
	if err != nil {
		Warnf("查询可检索文档失败: %v", err)
		return nil
	}
	best := make(map[string]*vo.ScopeRouteCandidate)
	for _, doc := range docs {
		scopeCode := utils.BlankToDefault(doc.KnowledgeScopeCode, "general_document")
		scopeName := utils.BlankToDefault(doc.KnowledgeScopeName, "通用文档")
		routeText := s.joinNonBlank(scopeCode, scopeName, doc.BusinessCategory, doc.DocumentTags)
		keywordScore := s.keywordEntityMatchScore(q.QueryTerms, routeText)
		semanticScore := s.singleSemanticScore(ctx, q, routeText)
		total := s.semanticMainScore(semanticScore) + keywordScore
		if total <= 0 && !q.semanticEnabled() {
			continue
		}
		if existing, ok := best[scopeCode]; ok && existing.Score >= total {
			continue
		}
		best[scopeCode] = &vo.ScopeRouteCandidate{
			ScopeCode: scopeCode,
			ScopeName: scopeName,
			Score:     total,
			Reason:    s.buildReason(q.QueryTerms, routeText, semanticScore),
		}
	}

	candidates := make([]*vo.ScopeRouteCandidate, 0, len(best))
	for _, c := range best {
		candidates = append(candidates, c)
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].Score > candidates[j].Score })
	limit := 5
	if len(candidates) > limit {
		candidates = candidates[:limit]
	}
	return candidates
}

// rankTopics 对主题节点打分：语义 + 词面 + 关键词 + 与当前 scope 命中的加分
func (s *KnowledgeRouteLogicImpl) rankTopics(ctx context.Context, q *routeQueryContext, scopeCandidates []*vo.ScopeRouteCandidate) []*vo.TopicRouteCandidate {
	nodes, err := s.repo.SelectKnowledgeTopicNodes(ctx)
	if err != nil {
		logx.Warnf("查询 topic 节点失败: %v", err)
	}
	preferredScopes := make(map[string]struct{}, len(scopeCandidates))
	for _, sc := range scopeCandidates {
		preferredScopes[sc.ScopeCode] = struct{}{}
	}

	if len(nodes) == 0 {
		return s.deriveTopicsFromProfiles(ctx, q, preferredScopes)
	}

	routeTexts := make([]string, 0, len(nodes))
	for _, node := range nodes {
		routeTexts = append(routeTexts, s.joinNonBlank(node.TopicName, node.Description, node.Aliases, node.Examples, node.AnswerShape, node.ExecutionPreference))
	}
	semanticScores := s.computeSemanticScores(ctx, q, routeTexts)
	lexicalScores := s.searchLexicalScores(ctx, q.RoutingText, "topic", 8)

	candidates := make([]*vo.TopicRouteCandidate, 0, len(nodes))
	for i, node := range nodes {
		semantic := 0.0
		if i < len(semanticScores) {
			semantic = semanticScores[i]
		}
		lexical := 0.0
		if v, ok := lexicalScores[node.TopicCode]; ok {
			lexical = v
		}
		score := s.semanticMainScore(semantic) + s.lexicalAssistScore(lexical) + s.keywordEntityMatchScore(q.QueryTerms, routeTexts[i])
		if _, preferred := preferredScopes[node.ScopeCode]; preferred {
			score += 8
		}
		if score > 0 || q.semanticEnabled() {
			candidates = append(candidates, &vo.TopicRouteCandidate{
				TopicCode: node.TopicCode,
				TopicName: node.TopicName,
				ScopeCode: node.ScopeCode,
				Score:     score,
				Reason:    s.buildReason(q.QueryTerms, routeTexts[i], semantic),
			})
		}
	}

	sort.Slice(candidates, func(i, j int) bool { return candidates[i].Score > candidates[j].Score })
	limit := 8
	if len(candidates) > limit {
		candidates = candidates[:limit]
	}
	return candidates
}

// deriveTopicsFromProfiles 当 topic 节点未配置时，按文档画像的 CoreTopics 派生主题候选
func (s *KnowledgeRouteLogicImpl) deriveTopicsFromProfiles(ctx context.Context, q *routeQueryContext, preferredScopes map[string]struct{}) []*vo.TopicRouteCandidate {
	profiles, err := s.repo.SelectDocumentProfiles(ctx)
	if err != nil {
		logx.Warnf("查询文档画像失败: %v", err)
		return nil
	}
	docs, err := s.repo.SelectRetrievableDocuments(ctx)
	if err != nil {
		logx.Warnf("查询可检索文档失败: %v", err)
	}
	scopeByDoc := make(map[int64]string, len(docs))
	for _, d := range docs {
		if d == nil {
			continue
		}
		scopeByDoc[d.DocumentId] = strutil.Trim(d.KnowledgeScopeCode)
	}

	best := make(map[string]*vo.TopicRouteCandidate)
	for _, profile := range profiles {
		if profile == nil {
			continue
		}
		for _, topic := range parseJsonStringArray(profile.CoreTopics) {
			topic = strutil.Trim(topic)
			if len(topic) < 2 {
				continue
			}
			routeText := s.joinNonBlank(topic, profile.DocumentSummary, profile.ExampleQuestions)
			keywordScore := s.keywordEntityMatchScore(q.QueryTerms, routeText)
			semanticScore := s.singleSemanticScore(ctx, q, routeText)
			scopeCode := scopeByDoc[profile.DocumentId]
			total := keywordScore + s.semanticMainScore(semanticScore)
			if _, preferred := preferredScopes[scopeCode]; preferred {
				total += 6
			}
			if existing, ok := best[topic]; ok && existing.Score >= total {
				continue
			}
			best[topic] = &vo.TopicRouteCandidate{
				TopicCode: normalizeCode(topic),
				TopicName: topic,
				ScopeCode: scopeCode,
				Score:     total,
				Reason:    s.buildReason(q.QueryTerms, routeText, semanticScore),
			}
		}
	}

	candidates := make([]*vo.TopicRouteCandidate, 0, len(best))
	for _, c := range best {
		candidates = append(candidates, c)
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].Score > candidates[j].Score })
	limit := 8
	if len(candidates) > limit {
		candidates = candidates[:limit]
	}
	return candidates
}

// rankDocuments 对文档进行打分，并将 top-score 的文档返回
func (s *KnowledgeRouteLogicImpl) rankDocuments(ctx context.Context, q *routeQueryContext, scopeCandidates []*vo.ScopeRouteCandidate, topicCandidates []*vo.TopicRouteCandidate) []*vo.DocumentRouteCandidate {
	documents, err := s.repo.SelectRetrievableDocuments(ctx)
	if err != nil {
		logx.Warnf("查询可检索文档失败: %v", err)
		return nil
	}
	if len(documents) == 0 {
		return nil
	}
	profiles, err := s.repo.SelectDocumentProfiles(ctx)
	if err != nil {
		Warnf("查询文档画像失败: %v", err)
	}
	profileByDoc := make(map[int64]*kvo.KnowledgeDocumentProfile, len(profiles))
	for _, p := range profiles {
		if p == nil {
			continue
		}
		profileByDoc[p.DocumentId] = p
	}

	relations, err := s.repo.SelectTopicDocumentRelations(ctx)
	if err != nil {
		Warnf("查询 topic-document 关系失败: %v", err)
	}
	relationByTopic := make(map[string]map[int64]*kvo.KnowledgeTopicDocumentRelation, len(relations))
	for _, rel := range relations {
		if rel == nil {
			continue
		}
		if _, ok := relationByTopic[rel.TopicCode]; !ok {
			relationByTopic[rel.TopicCode] = make(map[int64]*kvo.KnowledgeTopicDocumentRelation)
		}
		relationByTopic[rel.TopicCode][rel.DocumentId] = rel
	}

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
		routeTexts = append(routeTexts, s.buildDocumentRouteText(doc, profileByDoc[doc.DocumentId]))
	}
	semanticScores := s.computeSemanticScores(ctx, q, routeTexts)
	lexicalScores := s.searchDocumentLexicalScores(ctx, q.RoutingText, 5)

	// 打分
	matched := make([]*vo.DocumentRouteCandidate, 0, len(documents))
	for i, doc := range documents {
		if doc == nil {
			continue
		}
		routeText := routeTexts[i]
		semantic := 0.0
		if i < len(semanticScores) {
			semantic = semanticScores[i]
		}
		score := s.semanticMainScore(semantic) + s.lexicalAssistScore(lexicalScores[doc.DocumentId]) + s.keywordEntityMatchScore(q.QueryTerms, routeText)
		if strutil.IsNotBlank(topScopeCode) && topScopeCode == doc.KnowledgeScopeCode {
			score += 15
		}
		if strutil.IsNotBlank(topTopicCode) {
			if relMap, ok := relationByTopic[topTopicCode]; ok {
				if rel, ok := relMap[doc.DocumentId]; ok && rel.RelationScore > 0 {
					score += rel.RelationScore * 20
				}
			}
		}

		if score <= 0 && !q.semanticEnabled() {
			matched = append(matched, &vo.DocumentRouteCandidate{
				DocumentId:         strconv.FormatInt(doc.DocumentId, 10),
				DocumentName:       doc.DocumentName,
				LastIndexTaskId:    strconv.FormatInt(doc.LastIndexTaskId, 10),
				KnowledgeScopeCode: strutil.Trim(doc.KnowledgeScopeCode),
				KnowledgeScopeName: strutil.Trim(doc.KnowledgeScopeName),
				BusinessCategory:   strutil.Trim(doc.BusinessCategory),
				DocumentTags:       strutil.Trim(doc.DocumentTags),
				Score:              0,
				Reason:             "未命中路由关键词",
			})
			continue
		}
		matched = append(matched, &vo.DocumentRouteCandidate{
			DocumentId:         strconv.FormatInt(doc.DocumentId, 10),
			DocumentName:       doc.DocumentName,
			LastIndexTaskId:    strconv.FormatInt(doc.LastIndexTaskId, 10),
			KnowledgeScopeCode: strutil.Trim(doc.KnowledgeScopeCode),
			KnowledgeScopeName: strutil.Trim(doc.KnowledgeScopeName),
			BusinessCategory:   strutil.Trim(doc.BusinessCategory),
			DocumentTags:       strutil.Trim(doc.DocumentTags),
			Score:              score,
			Reason:             s.buildReason(q.QueryTerms, routeText, semantic),
		})
	}

	sort.Slice(matched, func(i, j int) bool { return matched[i].Score > matched[j].Score })
	limit := 5
	if len(matched) > limit {
		matched = matched[:limit]
	}
	return matched
}

// buildDocumentRouteText 拼接文档元数据 + 画像作为路由文本
func (s *KnowledgeRouteLogicImpl) buildDocumentRouteText(doc *kvo.KnowledgeDocument, profile *kvo.KnowledgeDocumentProfile) string {
	if profile == nil {
		return s.joinNonBlank(doc.DocumentName, doc.KnowledgeScopeName, doc.KnowledgeScopeCode, doc.BusinessCategory, doc.DocumentTags)
	}
	return s.joinNonBlank(doc.DocumentName, doc.KnowledgeScopeName, doc.KnowledgeScopeCode, doc.BusinessCategory, doc.DocumentTags,
		profile.DocumentSummary, profile.CoreTopics, profile.ExampleQuestions, profile.DocumentType)
}

// =====================================================
// 语义与词面打分辅助
// =====================================================

// computeSemanticScores 批量计算 routingText 与每个候选文本的余弦相似度；embedding 未配置时返回全 0 长度相同
func (s *KnowledgeRouteLogicImpl) computeSemanticScores(ctx context.Context, q *routeQueryContext, routeTexts []string) []float64 {
	scores := make([]float64, len(routeTexts))
	if !q.semanticEnabled() || s.embedding == nil || len(routeTexts) == 0 {
		return scores
	}

	total := len(routeTexts)
	for start := 0; start < total; start += routeEmbeddingBatchSize {
		end := start + routeEmbeddingBatchSize
		if end > total {
			end = total
		}
		batch := routeTexts[start:end]
		embeddings, err := s.embedding.EmbedBatch(ctx, batch)
		if err != nil || len(embeddings) != len(batch) {
			logx.Warnf("知识路由批量向量计算失败: batchStart=%d, size=%d, err=%v", start, len(batch), err)
			return make([]float64, total)
		}
		for idx, emb := range embeddings {
			scores[start+idx] = cosineSimilarity(q.QueryEmbedding, emb)
		}
	}
	return scores
}

// singleSemanticScore 仅对单一文本做语义相似度（失败或未配置返回 0）
func (s *KnowledgeRouteLogicImpl) singleSemanticScore(ctx context.Context, q *routeQueryContext, text string) float64 {
	if !q.semanticEnabled() || s.embedding == nil || strutil.IsBlank(text) {
		return 0
	}
	vectors, err := s.embedding.EmbedBatch(ctx, []string{text})
	if err != nil || len(vectors) == 0 {
		return 0
	}
	return cosineSimilarity(q.QueryEmbedding, vectors[0])
}

// searchLexicalScores 调用外部词面索引；未配置或失败时回退到本地计算
func (s *KnowledgeRouteLogicImpl) searchLexicalScores(ctx context.Context, routingText, entityType string, size int) map[string]float64 {
	if s.lexicalIndex == nil {
		return nil
	}
	hits, err := s.lexicalIndex.Search(ctx, routingText, entityType, size)
	if err != nil || len(hits) == 0 {
		return nil
	}
	result := make(map[string]float64, len(hits))
	for _, hit := range hits {
		if hit == nil {
			continue
		}
		if strutil.IsNotBlank(hit.EntityCode) {
			result[hit.EntityCode] = hit.Score
		}
	}
	return result
}

// searchDocumentLexicalScores 词面检索文档维度分数（同上）
func (s *KnowledgeRouteLogicImpl) searchDocumentLexicalScores(ctx context.Context, routingText string, size int) map[int64]float64 {
	if s.lexicalIndex == nil {
		return nil
	}
	hits, err := s.lexicalIndex.Search(ctx, routingText, "document", size)
	if err != nil || len(hits) == 0 {
		return nil
	}
	result := make(map[int64]float64, len(hits))
	for _, hit := range hits {
		if hit == nil || hit.DocumentId <= 0 {
			continue
		}
		result[hit.DocumentId] = hit.Score
	}
	return result
}

// semanticMainScore 将余弦相似度映射为可累加分值（低于 0.2 视为无效）
func (s *KnowledgeRouteLogicImpl) semanticMainScore(semanticScore float64) float64 {
	if semanticScore <= 0.20 {
		return 0
	}
	return (semanticScore - 0.20) * 50
}

// lexicalAssistScore 词面辅助分的软化
func (s *KnowledgeRouteLogicImpl) lexicalAssistScore(lexicalScore float64) float64 {
	if lexicalScore <= 0 {
		return 0
	}
	return math.Min(10, lexicalScore*1.6)
}

// keywordEntityMatchScore 对具有“实体感”的关键词加分：包含字母/数字或较短的中文短语
func (s *KnowledgeRouteLogicImpl) keywordEntityMatchScore(queryTerms []string, routeText string) float64 {
	if len(queryTerms) == 0 {
		return 0
	}
	normalizedContent := normalize(routeText)
	var score float64
	for _, term := range queryTerms {
		if !(alnumPattern.MatchString(term) || utf8.RuneCountInString(term) <= 4) {
			continue
		}
		termNorm := normalize(term)
		if utf8.RuneCountInString(termNorm) < 2 {
			continue
		}
		if strings.Contains(normalizedContent, termNorm) {
			score += 6
		}
	}
	return score
}

// =====================================================
// 通用工具
// =====================================================

// cosineSimilarity 计算两个等长向量的余弦相似度
func cosineSimilarity(left, right []float32) float64 {
	if len(left) == 0 || len(right) == 0 || len(left) != len(right) {
		return 0
	}
	var dot, lNorm, rNorm float64
	for i := 0; i < len(left); i++ {
		dot += float64(left[i]) * float64(right[i])
		lNorm += float64(left[i]) * float64(left[i])
		rNorm += float64(right[i]) * float64(right[i])
	}
	if lNorm <= 0 || rNorm <= 0 {
		return 0
	}
	return dot / (math.Sqrt(lNorm) * math.Sqrt(rNorm))
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

// joinNonBlank 用空格连接非空字符串列表
func (s *KnowledgeRouteLogicImpl) joinNonBlank(values ...string) string {
	nonBlank := make([]string, 0, len(values))
	for _, v := range values {
		if strutil.IsNotBlank(v) {
			nonBlank = append(nonBlank, strutil.Trim(v))
		}
	}
	return strings.Join(nonBlank, " ")
}

// buildReason 根据关键词命中和语义得分生成原因说明
func (s *KnowledgeRouteLogicImpl) buildReason(queryTerms []string, content string, semanticScore float64) string {
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

// resolveConfidence 计算整体置信度：以 top1 分数/(top1+top2+5) 归一化
func (s *KnowledgeRouteLogicImpl) resolveConfidence(documents []*cvo.DocumentRouteCandidate) float64 {
	if len(documents) == 0 {
		return 0
	}
	top1 := documents[0].Score
	top2 := 0.0
	if len(documents) > 1 {
		top2 = documents[1].Score
	}
	return top1 / math.Max(10, top1+top2+5)
}

// resolveDecisionReason 根据候选与置信度生成决策原因
func (s *KnowledgeRouteLogicImpl) resolveDecisionReason(documents []*cvo.DocumentRouteCandidate, confidence float64) string {
	if len(documents) == 0 {
		return "没有找到可用候选文档"
	}
	topReason := strutil.Trim(documents[0].Reason)
	if confidence < 0.55 {
		return firstNonBlank(topReason, "低置信度，已进入保守扩范围候选")
	}
	return topReason
}

// firstNonBlank 返回第一个非空字符串；全部为空则回退到 defaultValue
func firstNonBlank(primary, defaultValue string) string {
	if strutil.IsNotBlank(primary) {
		return strutil.Trim(primary)
	}
	return strutil.Trim(defaultValue)
}

// parseJsonStringArray 处理画像字段中的字符串数组：支持 ["a","b"] 或 a,b 或 a|b
func parseJsonStringArray(raw string) []string {
	cleaned := strutil.Trim(raw)
	if strutil.IsBlank(cleaned) || cleaned == "[]" {
		return nil
	}
	// 优先尝试 JSON 解析
	if strings.HasPrefix(cleaned, "[") && strings.HasSuffix(cleaned, "]") {
		var items []string
		if err := json.Unmarshal([]byte(cleaned), &items); err == nil {
			return items
		}
		// 回退到手工解析
		inner := strings.TrimPrefix(strings.TrimSuffix(cleaned, "]"), "[")
		items = splitAndTrim(inner, ",")
		return items
	}
	// 退化为按 , | 空白分割
	return splitAndTrim(cleaned, ",")
}

func splitAndTrim(s, sep string) []string {
	parts := strings.Split(s, sep)
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.Trim(strings.TrimSpace(strutil.Trim(p)), "\"")
		if strutil.IsNotBlank(trimmed) {
			out = append(out, trimmed)
		}
	}
	return out
}

// =====================================================
// Trace 写入
// =====================================================

// buildTrace 组装路由跟踪结构（不含选中文档与命中标记，由各路由模式补充）
func (s *KnowledgeRouteLogicImpl) buildTrace(conversationId, exchangeId, question, rewriteQuestion, mode string, decision *cvo.KnowledgeRouteDecision) *kvo.KnowledgeRouteTrace {
	trace := &kvo.KnowledgeRouteTrace{
		ConversationId:  conversationId,
		ExchangeId:      exchangeId,
		Question:        strutil.Trim(question),
		RewriteQuestion: strutil.Trim(rewriteQuestion),
		Mode:            mode,
		Status:          1,
	}
	if decision == nil {
		trace.RouteStatus = RouteStatusFailed
		return trace
	}
	trace.Confidence = decision.Confidence
	trace.RouteStatus = decision.RouteStatus
	trace.ErrorMsg = strutil.Trim(decision.Reason)
	trace.TopScopesJson = toCompactJSON(decision.Scopes, maxItems(5))
	trace.TopTopicsJson = toCompactJSON(decision.Topics, maxItems(5))
	trace.TopDocumentsJson = toCompactJSON(decision.Documents, maxItems(5))
	return trace
}

// resolveHitSelectedDocument 当 selectedDocumentId 有效时，判断其是否在候选前三
func (s *KnowledgeRouteLogicImpl) resolveHitSelectedDocument(selectedDocumentId int64, decision *cvo.KnowledgeRouteDecision) *int {
	if selectedDocumentId == 0 || decision == nil || len(decision.Documents) == 0 {
		return nil
	}
	hit := 0
	for idx, doc := range decision.Documents {
		if idx >= 3 {
			break
		}
		id, err := strconv.ParseInt(doc.DocumentId, 10, 64)
		if err == nil && id == selectedDocumentId {
			hit = 1
			break
		}
	}
	return &hit
}

// maxItems 返回限制数量（不修改结果只用于序列化）
func maxItems(n int) int { return n }

// toCompactJSON 将任意切片序列化为紧凑 JSON；超过 limit 则截断
func toCompactJSON(v any, limit int) string {
	if v == nil {
		return "[]"
	}
	// 简单起见：我们仅对几种候选类型进行序列化
	var items []map[string]any
	switch list := v.(type) {
	case []*cvo.ScopeRouteCandidate:
		for i, it := range list {
			if i >= limit {
				break
			}
			if it == nil {
				continue
			}
			items = append(items, map[string]any{
				"scopeCode": it.ScopeCode,
				"scopeName": it.ScopeName,
				"score":     round4(it.Score),
				"reason":    strutil.Trim(it.Reason),
			})
		}
	case []*cvo.TopicRouteCandidate:
		for i, it := range list {
			if i >= limit {
				break
			}
			if it == nil {
				continue
			}
			items = append(items, map[string]any{
				"topicCode": it.TopicCode,
				"topicName": it.TopicName,
				"scopeCode": it.ScopeCode,
				"score":     round4(it.Score),
				"reason":    strutil.Trim(it.Reason),
			})
		}
	case []*cvo.DocumentRouteCandidate:
		for i, it := range list {
			if i >= limit {
				break
			}
			if it == nil {
				continue
			}
			items = append(items, map[string]any{
				"documentId":      it.DocumentId,
				"documentName":    strutil.Trim(it.DocumentName),
				"lastIndexTaskId": it.LastIndexTaskId,
				"score":           round4(it.Score),
				"reason":          strutil.Trim(it.Reason),
			})
		}
	}
	if len(items) == 0 {
		return "[]"
	}
	data, err := json.Marshal(items)
	if err != nil {
		return "[]"
	}
	return string(data)
}

// round4 保留四位小数（与 Java BigDecimal#setScale(4, HALF_UP) 对齐）
func round4(v float64) float64 {
	return math.Round(v*10000) / 10000
}
