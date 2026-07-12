package logic

import (
	"context"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/duke-git/lancet/v2/maputil"
	"github.com/duke-git/lancet/v2/stream"
	"github.com/duke-git/lancet/v2/strutil"
	"github.com/zeromicro/go-zero/core/logx"

	"github.com/swiftbit/know-agent/common/utils"
	documentLogic "github.com/swiftbit/know-agent/internal/domain/document/logic"
	den "github.com/swiftbit/know-agent/internal/domain/document/model/entity"
	dvo "github.com/swiftbit/know-agent/internal/domain/document/model/vo"
	"github.com/swiftbit/know-agent/internal/domain/knowledge/adapter"
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

// KnowledgeRouteLogicImpl 知识路由服务实现：负责根据问题/改写问题匹配 scope/topic/document
type KnowledgeRouteLogicImpl struct {
	repo           adapter.KnowledgeRepository
	lifecycleLogic documentLogic.LifecycleLogic
	profileLogic   documentLogic.ProfileLogic
	*options
}

type options struct {
	embedder     adapter.Embedder
	lexicalIndex adapter.RouteLexicalIndex
}

type Option func(*options)

// NewKnowledgeRouteLogicImpl 创建路由服务实例
func NewKnowledgeRouteLogicImpl(repo adapter.KnowledgeRepository, lifecycleLogic documentLogic.LifecycleLogic, profileLogic documentLogic.ProfileLogic, opts ...Option) *KnowledgeRouteLogicImpl {
	base := new(options)
	for _, opt := range opts {
		opt(base)
	}
	return &KnowledgeRouteLogicImpl{
		repo:           repo,
		lifecycleLogic: lifecycleLogic,
		profileLogic:   profileLogic,
		options:        base,
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
func (s *KnowledgeRouteLogicImpl) Route(ctx context.Context, question, rewriteQuestion string) (*vo.KnowledgeRouteDecision, error) {
	queryCtx := s.buildQueryContext(ctx, question, rewriteQuestion)
	decision := &vo.KnowledgeRouteDecision{RouteStatus: vo.RouteStatusFailed}
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
		decision.RouteStatus = vo.RouteStatusFailed
	case decision.Confidence < 0.55:
		decision.RouteStatus = vo.RouteStatusLowConfidence
	default:
		decision.RouteStatus = vo.RouteStatusSuccess
	}
	decision.Reason = s.resolveDecisionReason(documentCandidates, decision.Confidence)

	logx.Infof("知识范围路由完成: question='%s', rewriteQuestion='%s', scopeCount=%d, topicCount=%d, documentCount=%d, confidence=%.4f,topDocument='%s",
		strutil.Trim(question), strutil.Trim(rewriteQuestion), len(scopeCandidates), len(topicCandidates), len(documentCandidates), decision.Confidence, documentCandidates[0].DocumentName)
	return decision, nil
}

// RecordShadowRoute 记录影子路由结果（后台写入不影响主流程）
func (s *KnowledgeRouteLogicImpl) RecordShadowRoute(ctx context.Context, exchangeId, documentId int64, conversationId, question, rewriteQuestion string) error {
	decision, err := s.Route(ctx, question, rewriteQuestion)
	if err != nil {
		Warnf("知识路由[shadow]失败: conversationId=%s, err=%v", conversationId, err)
		return err
	}
	trace := s.buildTrace(exchangeId, conversationId, question, rewriteQuestion, routeModeShadow, decision)
	trace.SelectedDocumentId = documentId
	trace.HitSelectedDocument = decision.ResolveHitSelectedDocument(documentId)
	if err = s.repo.InsertKnowledgeRouteTrace(ctx, trace); err != nil {
		Warnf("记录知识路由[shadow]失败: conversationId=%s, exchangeId=%d, err=%v", conversationId, exchangeId, err)
		return err
	}
	return nil
}

// RecordAutoRoute 记录自动路由结果
func (s *KnowledgeRouteLogicImpl) RecordAutoRoute(ctx context.Context, exchangeId int64, conversationId, question, rewriteQuestion string, decision *vo.KnowledgeRouteDecision) error {
	trace := s.buildTrace(exchangeId, conversationId, question, rewriteQuestion, routeModeAuto, decision)
	if len(decision.Documents) > 0 {
		trace.SelectedDocumentId = decision.Documents[0].DocumentId
	}
	trace.HitSelectedDocument = decision.ResolveHitSelectedDocument(trace.SelectedDocumentId)
	if err := s.repo.InsertKnowledgeRouteTrace(ctx, trace); err != nil {
		Warnf("记录知识路由[auto]失败: conversationId=%s, err=%v", conversationId, err)
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

// buildQueryContext 组装路由上下文：拼接检索文本 + 分词 + 可选生成向量
func (s *KnowledgeRouteLogicImpl) buildQueryContext(ctx context.Context, question, rewriteQuestion string) *routeQueryContext {
	routingText := s.buildRoutingText(question, rewriteQuestion)
	terms := s.tokenize(routingText)

	// 若外部 embedder 未配置，则回退到纯词面
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

	terms := make(map[string]struct{})
	for _, part := range tokenSplitPattern.Split(cleaned, -1) {
		trimmed := strutil.Trim(part)
		if utf8.RuneCountInString(trimmed) > 1 {
			terms[trimmed] = struct{}{}
			s.expandChineseNgrams(trimmed, terms)
		}
	}

	// 限制最大关键词数量
	return utils.LimitSlice(maputil.Keys(terms), 40)
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
		routeTexts = append(routeTexts, utils.JoinNonBlank(" ", node.ScopeName, node.Description, node.Aliases, node.Examples))
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
		lexical := lexicalScores[node.ScopeCode]
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
	return utils.LimitSlice(candidates, 5)
}

// deriveScopesFromDocuments 当没有配置 scope 节点时，从文档元数据派生粗略的 scope 候选
func (s *KnowledgeRouteLogicImpl) deriveScopesFromDocuments(ctx context.Context, q *routeQueryContext) []*vo.ScopeRouteCandidate {
	docs, err := s.lifecycleLogic.ListRetrievableDocuments(ctx)
	if err != nil {
		Warnf("查询可检索文档失败: %v", err)
		return nil
	}
	best := make(map[string]*vo.ScopeRouteCandidate)
	for _, doc := range docs {
		if strutil.IsBlank(doc.KnowledgeScopeCode) && strutil.IsBlank(doc.KnowledgeScopeName) {
			continue
		}
		scopeCode := utils.BlankToDefault(doc.KnowledgeScopeCode, "general_document")
		scopeName := utils.BlankToDefault(doc.KnowledgeScopeName, "通用文档")
		routeText := utils.JoinNonBlank(" ", scopeCode, scopeName, doc.BusinessCategory, doc.DocumentTags)
		semanticScore := s.singleSemanticScore(ctx, q, routeText)
		finalScore := s.semanticMainScore(semanticScore) + s.keywordEntityMatchScore(q.QueryTerms, routeText)
		if finalScore <= 0 && len(q.QueryEmbedding) == 0 {
			continue
		}
		if existing := best[scopeCode]; existing.Score < finalScore {
			best[scopeCode] = &vo.ScopeRouteCandidate{
				ScopeCode: scopeCode,
				ScopeName: scopeName,
				Score:     finalScore,
				Reason:    s.buildReason(q.QueryTerms, routeText, semanticScore),
			}
		}
	}

	candidates := maputil.Values(best)
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].Score > candidates[j].Score })
	return utils.LimitSlice(candidates, 5)
}

// rankTopics 对主题节点打分：语义 + 词面 + 关键词 + 与当前 scope 命中的加分
func (s *KnowledgeRouteLogicImpl) rankTopics(ctx context.Context, q *routeQueryContext, scopeCandidates []*vo.ScopeRouteCandidate) []*vo.TopicRouteCandidate {
	nodes, err := s.repo.SelectKnowledgeTopicNodes(ctx)
	if err != nil {
		Warnf("查询 topic 节点失败: %v", err)
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
		routeTexts = append(routeTexts, utils.JoinNonBlank(" ", node.TopicName, node.Description, node.Aliases, node.Examples, node.AnswerShape, node.ExecutionPreference))
	}
	semanticScores := s.computeSemanticScores(ctx, q, routeTexts)
	lexicalScores := s.searchLexicalScores(ctx, q.RoutingText, "topic", 8)

	candidates := make([]*vo.TopicRouteCandidate, 0, len(nodes))
	for i, node := range nodes {
		semantic := 0.0
		if i < len(semanticScores) {
			semantic = semanticScores[i]
		}
		lexical := lexicalScores[node.TopicCode]
		score := s.semanticMainScore(semantic) + s.lexicalAssistScore(lexical) + s.keywordEntityMatchScore(q.QueryTerms, routeTexts[i])
		if _, preferred := preferredScopes[node.ScopeCode]; preferred {
			score += 8
		}
		if score > 0 || len(q.QueryEmbedding) > 0 {
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
	return utils.LimitSlice(candidates, 8)
}

// deriveTopicsFromProfiles 当 topic 节点未配置时，按文档画像的 CoreTopics 派生主题候选
func (s *KnowledgeRouteLogicImpl) deriveTopicsFromProfiles(ctx context.Context, q *routeQueryContext, preferredScopes map[string]struct{}) []*vo.TopicRouteCandidate {
	profiles, err := s.profileLogic.GetAllProfiles(ctx)
	if err != nil {
		Warnf("查询文档画像失败: %v", err)
		return nil
	}
	docs, err := s.lifecycleLogic.ListRetrievableDocuments(ctx)
	if err != nil {
		Warnf("查询可检索文档失败: %v", err)
	}
	scopeByDoc := make(map[int64]string, len(docs))
	for _, d := range docs {
		scopeByDoc[d.DocumentId] = strutil.Trim(d.KnowledgeScopeCode)
	}

	best := make(map[string]*vo.TopicRouteCandidate)
	for _, profile := range profiles {
		for _, topic := range parseJsonStringArray(profile.CoreTopics) {
			topic = strutil.Trim(topic)
			if utf8.RuneCountInString(topic) < 2 {
				continue
			}
			routeText := utils.JoinNonBlank(" ", topic, profile.DocumentSummary, profile.ExampleQuestions)
			keywordScore := s.keywordEntityMatchScore(q.QueryTerms, routeText)
			semanticScore := s.singleSemanticScore(ctx, q, routeText)
			scopeCode := scopeByDoc[profile.DocumentId]
			finalScore := keywordScore + s.semanticMainScore(semanticScore)
			if _, preferred := preferredScopes[scopeCode]; preferred {
				finalScore += 6
			}
			if existing := best[topic]; existing.Score < finalScore {
				best[topic] = &vo.TopicRouteCandidate{
					TopicCode: normalizeCode(topic),
					TopicName: topic,
					ScopeCode: scopeCode,
					Score:     finalScore,
					Reason:    s.buildReason(q.QueryTerms, routeText, semanticScore),
				}
			}
		}
	}

	candidates := maputil.Values(best)
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].Score > candidates[j].Score })
	return utils.LimitSlice(candidates, 8)
}

// rankDocuments 对文档进行打分，并将 top-score 的文档返回
func (s *KnowledgeRouteLogicImpl) rankDocuments(ctx context.Context, q *routeQueryContext, scopeCandidates []*vo.ScopeRouteCandidate, topicCandidates []*vo.TopicRouteCandidate) []*vo.DocumentRouteCandidate {
	documents, err := s.lifecycleLogic.ListRetrievableDocuments(ctx)
	if err != nil {
		Warnf("查询可检索文档失败: %v", err)
		return nil
	}
	if len(documents) == 0 {
		return nil
	}
	profiles, err := s.profileLogic.GetAllProfiles(ctx)
	if err != nil {
		Warnf("查询文档画像失败: %v", err)
	}
	profileByDoc := make(map[int64]*den.DocumentProfile, len(profiles))
	for _, p := range profiles {
		profileByDoc[p.DocumentId] = p
	}

	relations, err := s.repo.SelectTopicDocumentRelations(ctx)
	if err != nil {
		Warnf("查询 topic-document 关系失败: %v", err)
	}
	relationByTopic := make(map[string]map[int64]*entity.KnowledgeTopicDocumentRelation, len(relations))
	for _, rel := range relations {
		if _, ok := relationByTopic[rel.TopicCode]; !ok {
			relationByTopic[rel.TopicCode] = make(map[int64]*entity.KnowledgeTopicDocumentRelation)
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

		elems := &vo.DocumentRouteCandidate{
			DocumentId:         doc.DocumentId,
			DocumentName:       doc.DocumentName,
			LastIndexTaskId:    doc.LastIndexTaskId,
			KnowledgeScopeCode: strutil.Trim(doc.KnowledgeScopeCode),
			KnowledgeScopeName: strutil.Trim(doc.KnowledgeScopeName),
			BusinessCategory:   strutil.Trim(doc.BusinessCategory),
			DocumentTags:       strutil.Trim(doc.DocumentTags),
			Reason:             "未命中路由关键词",
		}
		if score <= 0 && len(q.QueryEmbedding) == 0 {
			matched = append(matched, elems)
			continue
		}
		elems.Score = score
		elems.Reason = s.buildReason(q.QueryTerms, routeText, semantic)
		matched = append(matched, elems)
	}

	sort.Slice(matched, func(i, j int) bool { return matched[i].Score > matched[j].Score })
	return utils.LimitSlice(matched, 5)
}

// buildDocumentRouteText 拼接文档元数据 + 画像作为路由文本
func (s *KnowledgeRouteLogicImpl) buildDocumentRouteText(doc *dvo.KnowledgeDocument, profile *den.DocumentProfile) string {
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
func (s *KnowledgeRouteLogicImpl) computeSemanticScores(ctx context.Context, q *routeQueryContext, routeTexts []string) []float64 {
	scores := make([]float64, len(routeTexts))
	if len(q.QueryEmbedding) == 0 || s.embedder == nil || len(routeTexts) == 0 {
		return scores
	}

	for start := 0; start < len(routeTexts); start += routeEmbeddingBatchSize {
		end := min(start+routeEmbeddingBatchSize, len(routeTexts))
		batch := routeTexts[start:end]
		embeddings, err := s.embedder.EmbedStrings(ctx, batch...)
		if err != nil || len(embeddings) != len(batch) {
			Warnf("知识路由批量向量计算失败: batchStart=%d, size=%d, err=%v", start, len(batch), err)
			return make([]float64, len(routeTexts))
		}
		for idx, emb := range embeddings {
			scores[start+idx] = cosineSimilarity(q.QueryEmbedding, emb)
		}
	}
	return scores
}

// singleSemanticScore 仅对单一文本做语义相似度（失败或未配置返回 0）
func (s *KnowledgeRouteLogicImpl) singleSemanticScore(ctx context.Context, q *routeQueryContext, text string) float64 {
	if len(q.QueryEmbedding) == 0 || s.embedder == nil || strutil.IsBlank(text) {
		return 0
	}
	vectors, err := s.embedder.EmbedStrings(ctx, text)
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
		result[hit.EntityCode] = hit.Score
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
		result[hit.DocumentId] = hit.Score
	}
	return result
}

// semanticMainScore 将余弦相似度映射为可累加分值（低于 0.2 视为无效）
func (s *KnowledgeRouteLogicImpl) semanticMainScore(semanticScore float64) float64 {
	return max(0, (semanticScore-0.20)*50)
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
		if !alpNumPattern.MatchString(term) && utf8.RuneCountInString(term) > 4 {
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
func cosineSimilarity(left, right []float64) float64 {
	if len(left) == 0 || len(right) == 0 || len(left) != len(right) {
		return 0
	}
	var dot, lNorm, rNorm float64
	for i := 0; i < len(left); i++ {
		dot += left[i] * right[i]
		lNorm += left[i] * left[i]
		rNorm += right[i] * right[i]
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
func (s *KnowledgeRouteLogicImpl) resolveConfidence(documents []*vo.DocumentRouteCandidate) float64 {
	if len(documents) == 0 {
		return 0
	}
	top1 := documents[0].Score
	top2 := 0.0
	if len(documents) > 1 {
		top2 = documents[1].Score
	}
	return top1 / max(10, top1+top2+5)
}

// resolveDecisionReason 根据候选与置信度生成决策原因
func (s *KnowledgeRouteLogicImpl) resolveDecisionReason(documents []*vo.DocumentRouteCandidate, confidence float64) string {
	if len(documents) == 0 {
		return "没有找到可用候选文档"
	}
	top := documents[0]
	if confidence >= 0.80 {
		return fmt.Sprintf("高置信度路由到《%s》，置信度 %.2f", top.DocumentName, confidence)
	} else if confidence >= 0.55 {
		return fmt.Sprintf("中等置信度路由到《%s》，置信度 %.2f", top.DocumentName, confidence)
	}
	return fmt.Sprintf("低置信度，前 %d 个候选得分接近，建议澄清", min(3, len(documents)))
}

// parseJsonStringArray 处理画像字段中的字符串数组：支持 ["a","b"]
func parseJsonStringArray(raw string) []string {
	cleaned := strutil.Trim(raw)
	if strutil.IsBlank(cleaned) || cleaned == "[]" {
		return nil
	}

	// 优先尝试 JSON 解析
	var items []string
	if err := utils.Unmarshal(cleaned, &items); err == nil {
		return items
	}

	// 回退到手工解析
	inner := strings.TrimPrefix(strings.TrimSuffix(cleaned, "]"), "[")
	return stream.FromSlice(strings.Split(inner, ",")).
		Map(func(item string) string { return strutil.Trim(strutil.Trim(item), "\"") }).
		Filter(func(item string) bool { return strutil.IsNotBlank(item) }).ToSlice()
}

// =====================================================
// Trace 写入
// =====================================================

// buildTrace 组装路由跟踪结构（不含选中文档与命中标记，由各路由模式补充）
func (s *KnowledgeRouteLogicImpl) buildTrace(exchangeId int64, conversationId, question, rewriteQuestion, mode string, decision *vo.KnowledgeRouteDecision) *entity.KnowledgeRouteTrace {
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
