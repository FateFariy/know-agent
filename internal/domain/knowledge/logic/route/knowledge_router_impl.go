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
	documentLogic "github.com/swiftbit/know-agent/internal/domain/document/logic"
	den "github.com/swiftbit/know-agent/internal/domain/document/model/entity"
	dvo "github.com/swiftbit/know-agent/internal/domain/document/model/vo"
	"github.com/swiftbit/know-agent/internal/domain/knowledge/adapter"
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
	repo         adapter.KnowledgeRepository
	docGateway   adapter.DocumentGateway
	profileLogic documentLogic.ProfileLogic
	scorer       score.Scorer
	*options
}

type options struct {
	embedder     adapter.Embedder
	lexicalIndex adapter.RouteLexicalIndex
}

type Option func(*options)

// NewKnowledgeRouteImpl 创建路由服务实例
func NewKnowledgeRouteImpl(repo adapter.KnowledgeRepository, docGateway adapter.DocumentGateway, profileLogic documentLogic.ProfileLogic, opts ...Option) *KnowledgeRouteImpl {
	base := new(options)
	for _, opt := range opts {
		opt(base)
	}
	return &KnowledgeRouteImpl{
		repo:         repo,
		docGateway:   docGateway,
		profileLogic: profileLogic,
		scorer:       score.NewDefaultScorer(),
		options:      base,
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
func (s *KnowledgeRouteImpl) Route(ctx context.Context, queryCtx *QueryContext) (*vo.KnowledgeRouteDecision, error) {
	decision := &vo.KnowledgeRouteDecision{RouteStatus: vo.RouteStatusFailed}
	if queryCtx.RoutingText == "" {
		decision.Reason = "问题为空或无法提取有效关键词"
		return decision, nil
	}

	scopeCandidates := s.rankScopes(ctx, queryCtx)
	topicCandidates := s.rankTopics(ctx, queryCtx, scopeCandidates)
	documentCandidates := s.rankDocuments(ctx, queryCtx, scopeCandidates, topicCandidates)

	decision.Scopes = scopeCandidates
	decision.Topics = topicCandidates
	decision.Documents = documentCandidates
	decision.Confidence = s.computeConfidence(documentCandidates)

	// 设置决策来源（从文档候选中解析）
	decision.Source = s.resolveDecisionSource(documentCandidates)

	// 设置降级状态与原因
	decision.IsDegraded = len(queryCtx.Diagnostics) > 0
	for reason := range queryCtx.Diagnostics {
		decision.DegradedReasons = append(decision.DegradedReasons, reason)
	}

	switch {
	case len(documentCandidates) == 0:
		decision.RouteStatus = vo.RouteStatusFailed
		decision.Reason = "没有找到可用候选文档"
	case decision.Confidence < lowConfidenceThreshold:
		decision.RouteStatus = vo.RouteStatusLowConfidence
	default:
		decision.RouteStatus = vo.RouteStatusSuccess
	}
	if decision.RouteStatus != vo.RouteStatusFailed {
		decision.Reason = s.resolveDecisionReason(documentCandidates, decision.Confidence)
	}

	topDocName := ""
	if len(documentCandidates) > 0 {
		topDocName = documentCandidates[0].DocumentName
	}
	logx.Infof("知识范围路由完成: question='%s', rewriteQuestion='%s', scopeCount=%d, topicCount=%d, documentCount=%d, confidence=%.4f, source=%s, degraded=%v, topDocument='%s'",
		strutil.Trim(queryCtx.Question), strutil.Trim(queryCtx.RewriteQuestion),
		len(scopeCandidates), len(topicCandidates), len(documentCandidates),
		decision.Confidence, decision.Source, decision.IsDegraded, topDocName)
	return decision, nil
}

// RecordShadowRoute 记录影子路由结果（后台写入不影响主流程）
func (s *KnowledgeRouteImpl) RecordShadowRoute(ctx context.Context, exchangeId, documentId int64, conversationId, question, rewriteQuestion string) error {
	queryCtx := NewQueryContext(question, rewriteQuestion, nil, nil)
	decision, err := s.Route(ctx, queryCtx)
	if err != nil {
		logx.Warnf("知识路由[shadow]失败: conversationId=%s, err=%v", conversationId, err)
		return err
	}
	trace := s.buildTrace(exchangeId, conversationId, question, rewriteQuestion, routeModeShadow, decision)
	trace.SelectedDocumentId = documentId
	trace.HitSelectedDocument = decision.ResolveHitSelectedDocument(documentId)
	if err = s.repo.InsertKnowledgeRouteTrace(ctx, trace); err != nil {
		logx.Warnf("记录知识路由[shadow]失败: conversationId=%s, exchangeId=%d, err=%v", conversationId, exchangeId, err)
		return err
	}
	return nil
}

// RecordAutoRoute 记录自动路由结果
func (s *KnowledgeRouteImpl) RecordAutoRoute(ctx context.Context, exchangeId int64, conversationId, question, rewriteQuestion string, decision *vo.KnowledgeRouteDecision) error {
	trace := s.buildTrace(exchangeId, conversationId, question, rewriteQuestion, routeModeAuto, decision)
	if len(decision.Documents) > 0 {
		trace.SelectedDocumentId = decision.Documents[0].DocumentId
	}
	trace.HitSelectedDocument = decision.ResolveHitSelectedDocument(trace.SelectedDocumentId)
	if err := s.repo.InsertKnowledgeRouteTrace(ctx, trace); err != nil {
		logx.Warnf("记录知识路由[auto]失败: conversationId=%s, err=%v", conversationId, err)
		return err
	}
	return nil
}

// tokenize 分词：按中英文常见分隔符分割，再对长度 >=4 的中文片段进行 n-gram 扩展
func (s *KnowledgeRouteImpl) tokenize(text string) []string {
	cleaned := strutil.Trim(text)
	if strutil.IsBlank(cleaned) {
		return nil
	}

	terms := make(map[string]struct{})
	for _, part := range tokenSplitPattern.Split(cleaned, -1) {
		trimmed := strutil.Trim(part)
		if utils.Len(trimmed) > 1 {
			terms[trimmed] = struct{}{}
			s.expandChineseNgrams(trimmed, terms)
		}
	}

	// 限制最大关键词数量
	return utils.Limit(maputil.Keys(terms), 40)
}

// expandChineseNgrams 对中文短片段做 2~maxGram 的滑动窗口扩展（用于提高短实体召回）
func (s *KnowledgeRouteImpl) expandChineseNgrams(segment string, seen map[string]struct{}) {
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
func (s *KnowledgeRouteImpl) rankScopes(ctx context.Context, q *QueryContext) []*vo.ScopeRouteCandidate {
	nodes, err := s.repo.SelectKnowledgeScopeNodesByKbIds(ctx, q.SelectedKnowledgeBaseIds)
	if err != nil {
		logx.Warnf("查询 scope 节点失败: %v", err)
	}
	if len(nodes) == 0 {
		return nil
	}

	// 生成 routeText 列表
	routeTexts := utils.Map(nodes, func(node *entity.KnowledgeScopeNode) string {
		return utils.JoinNonBlank(" ", node.ScopeName, node.Description, node.Aliases, node.Examples)
	})
	semanticScores := s.computeSemanticScores(ctx, q, routeTexts)
	lexicalScores := s.searchLexicalScores(ctx, q.RoutingText, "scope", 5)

	// 组装候选
	candidates := make([]*vo.ScopeRouteCandidate, 0, len(nodes))
	for i, node := range nodes {
		scoreResult := s.scorer.Score(&score.Features{
			SemanticScore: semanticScores[i],
			LexicalScore:  lexicalScores[node.ID],
		})

		if scoreResult.TotalScore > 0 {
			candidates = append(candidates, &vo.ScopeRouteCandidate{
				ScopeId:   node.ID,
				ScopeName: node.ScopeName,
				Score:     scoreResult.TotalScore,
				Reason:    scoreResult.Reason,
				Source:    scoreResult.Source,
				Features:  scoreResult.Features,
			})
		}
	}

	sort.Slice(candidates, func(i, j int) bool { return candidates[i].Score > candidates[j].Score })
	return utils.Limit(candidates, 5)
}

// rankTopics 对主题节点打分：语义 + 词面 + 关键词 + 与当前 scope 命中的加分
func (s *KnowledgeRouteImpl) rankTopics(ctx context.Context, q *QueryContext, scopeCandidates []*vo.ScopeRouteCandidate) []*vo.TopicRouteCandidate {
	nodes, err := s.repo.SelectKnowledgeTopicNodesByKbIds(ctx, nil)
	if err != nil {
		logx.Warnf("查询 topic 节点失败: %v", err)
	}
	preferredScopes := utils.MapBy(nodes, func(node *entity.KnowledgeTopicNode) (int64, *entity.KnowledgeTopicNode) {
		return node.ScopeId, node
	})

	if len(nodes) == 0 {
		return s.deriveTopicsFromProfiles(ctx, q)
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
		score := semanticMainScore(semantic) + lexicalAssistScore(lexical)

		// 与预选 scope 匹配加分
		scopeRelationScore := 0.0
		if _, preferred := preferredScopes[node.ScopeCode]; preferred {
			scopeRelationScore = 8
			score += scopeRelationScore
		}

		if score > 0 || len(q.QueryEmbedding) > 0 {
			candidates = append(candidates, &vo.TopicRouteCandidate{
				TopicName: node.TopicName,
				Score:     score,
				Reason:    s.buildCandidateReason(semantic, lexical, scopeRelationScore),
				Source:    s.resolveCandidateSource(semantic, lexical, scopeRelationScore),
				Features: map[string]float64{
					"semanticScore":      max(0, semantic),
					"routeIndexScore":    max(0, lexical),
					"scopeRelationScore": scopeRelationScore,
				},
			})
		}
	}

	sort.Slice(candidates, func(i, j int) bool { return candidates[i].Score > candidates[j].Score })
	return utils.Limit(candidates, 8)
}

// deriveTopicsFromProfiles 当 topic 节点未配置时，按文档画像的 CoreTopics 派生主题候选
func (s *KnowledgeRouteImpl) deriveTopicsFromProfiles(ctx context.Context, q *QueryContext) []*vo.TopicRouteCandidate {
	docs, _ := s.listRetrievableDocuments(ctx, q)
	if len(docs) == 0 {
		return nil
	}

	docIds := utils.Map(docs, func(d *vo.DocumentMetadata) int64 { return d.DocumentId })
	profiles, err := s.docGateway.FindDocumentProfileByDocIds(ctx, docIds)
	if err != nil {
		logx.Warnf("查询文档画像失败: %v", err)
		return nil
	}

	best := make(map[string]*vo.TopicRouteCandidate)
	for _, profile := range profiles {
		for _, topic := range parseJsonStringArray(profile.CoreTopics) {
			routeText := utils.JoinNonBlank(" ", topic, profile.DocumentSummary, profile.ExampleQuestions)
			semanticScore := s.singleSemanticScore(ctx, q, routeText)
			scopeRelationScore := 0.0
			if _, preferred := preferredScopes[scopeCode]; preferred {
				scopeRelationScore = 6
			}
			finalScore := keywordScore + s.semanticMainScore(semanticScore) + scopeRelationScore
			if existing, ok := best[topic]; !ok || existing.Score < finalScore {
				best[topic] = &vo.TopicRouteCandidate{
					TopicCode: normalizeCode(topic),
					TopicName: topic,
					ScopeCode: scopeCode,
					Score:     finalScore,
					Reason:    s.buildCandidateReason(semanticScore, 0, scopeRelationScore),
					Source:    s.resolveCandidateSource(semanticScore, 0, scopeRelationScore),
					Features: map[string]float64{
						"semanticScore":      max(0, semanticScore),
						"scopeRelationScore": scopeRelationScore,
					},
				}
			}
		}
	}

	candidates := maputil.Values(best)
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].Score > candidates[j].Score })
	return utils.Limit(candidates, 8)
}

// listRetrievableDocuments 查询可检索的文档
func (s *KnowledgeRouteImpl) listRetrievableDocuments(ctx context.Context, q *QueryContext) ([]*vo.DocumentMetadata, error) {
	var docs []*vo.DocumentMetadata
	var err error
	if len(q.AllowedDocumentIds) != 0 {
		docs, err = s.docGateway.FindRetrieveDocumentByIds(ctx, q.AllowedDocumentIds)
		if err != nil {
			logx.Warnf("查询可检索文档失败: %v", err)
		}
	}
	if len(docs) == 0 {
		docs, err = s.docGateway.FindRetrievableByKbIds(ctx, q.SelectedKnowledgeBaseIds)
		if err != nil {
			logx.Warnf("查询可检索文档失败: %v", err)
		}
	}
	return docs, err
}

const (
	documentCandidateLimit = 5
	lowConfidenceThreshold = 0.55
)

// rankDocuments 对文档进行打分，并将 top-score 的文档返回
//
// 打分策略：语义分 + 词面辅助分 + 关键词命中分 + 与 topScope 匹配加分 + 主题关系加分
// 低置信度扩展：当置信度 < 阈值时，将允许范围内的文档作为零分候选补充
func (s *KnowledgeRouteImpl) rankDocuments(ctx context.Context, q *QueryContext, scopeCandidates []*vo.ScopeRouteCandidate, topicCandidates []*vo.TopicRouteCandidate) []*vo.DocumentRouteCandidate {
	documents, err := s.lifecycleLogic.ListRetrievableDocuments(ctx)
	if err != nil {
		logx.Warnf("查询可检索文档失败: %v", err)
		return nil
	}
	if len(documents) == 0 {
		return nil
	}

	// 加载文档画像
	profiles, err := s.profileLogic.GetAllProfiles(ctx)
	if err != nil {
		logx.Warnf("查询文档画像失败: %v", err)
	}
	profileByDoc := make(map[int64]*den.DocumentProfile, len(profiles))
	for _, p := range profiles {
		profileByDoc[p.DocumentId] = p
	}

	// 加载主题-文档关系
	relations, err := s.repo.SelectTopicDocumentRelations(ctx)
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
		routeTexts = append(routeTexts, s.buildDocumentRouteText(doc, profileByDoc[doc.DocumentId]))
	}
	semanticScores := s.computeSemanticScores(ctx, q, routeTexts)
	lexicalScores := s.searchDocumentLexicalScores(ctx, q.RoutingText, documentCandidateLimit)

	// 打分：语义分 + 词面辅助分 + 关键词命中分
	matched := make([]*vo.DocumentRouteCandidate, 0, len(documents))
	for i, doc := range documents {
		routeText := routeTexts[i]
		semantic := 0.0
		if i < len(semanticScores) {
			semantic = semanticScores[i]
		}
		lexical := lexicalScores[doc.DocumentId]
		score := s.semanticMainScore(semantic) + s.lexicalAssistScore(lexical) + s.keywordEntityMatchScore(q.QueryTerms, routeText)

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
			DocumentId:         doc.DocumentId,
			DocumentName:       doc.DocumentName,
			LastIndexTaskId:    doc.LastIndexTaskId,
			KnowledgeScopeCode: strutil.Trim(doc.KnowledgeScopeCode),
			KnowledgeScopeName: strutil.Trim(doc.KnowledgeScopeName),
			BusinessCategory:   strutil.Trim(doc.BusinessCategory),
			DocumentTags:       strutil.Trim(doc.DocumentTags),
			Score:              score,
			Reason:             s.buildCandidateReason(semantic, lexical, topicRelationScore),
			Source:             s.resolveCandidateSource(semantic, lexical, topicRelationScore),
			Features:           s.buildFeatures(semantic, lexical, topicRelationScore),
		})
	}

	// 按得分降序排列
	sort.Slice(matched, func(i, j int) bool { return matched[i].Score > matched[j].Score })
	candidates := utils.Limit(matched, documentCandidateLimit)

	// 低置信度扩展：当无候选或置信度低时，补充允许范围内的文档
	if len(candidates) == 0 || (len(candidates) > 0 && s.computeConfidence(candidates) < lowConfidenceThreshold) {
		expanded := s.expandCandidatesWithAllowedScope(candidates, documents, q, documentCandidateLimit)
		if len(expanded) > len(candidates) {
			q.Diagnostics["LOW_CONFIDENCE_ALLOWED_SCOPE_EXPANSION"] = struct{}{}
		}
		candidates = expanded
	}

	return candidates
}

// computeConfidence 从候选列表计算置信度：top1/(top1+top2+5) 归一化
func (s *KnowledgeRouteImpl) computeConfidence(candidates []*vo.DocumentRouteCandidate) float64 {
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
func (s *KnowledgeRouteImpl) expandCandidatesWithAllowedScope(ranked []*vo.DocumentRouteCandidate, documents []*dvo.DocumentMetadata, q *QueryContext, limit int) []*vo.DocumentRouteCandidate {
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
func (s *KnowledgeRouteImpl) buildDocumentRouteText(doc *dvo.DocumentMetadata, profile *den.DocumentProfile) string {
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
func (s *KnowledgeRouteImpl) computeSemanticScores(ctx context.Context, q *QueryContext, routeTexts []string) []float64 {
	scores := make([]float64, len(routeTexts))
	if len(q.QueryEmbedding) == 0 || len(routeTexts) == 0 {
		return scores
	}
	if s.embedder == nil {
		q.Diagnostics["SEMANTIC_ROUTE_NOT_CONFIGURED"] = struct{}{}
		return scores
	}

	for start := 0; start < len(routeTexts); start += routeEmbeddingBatchSize {
		end := min(start+routeEmbeddingBatchSize, len(routeTexts))
		batch := routeTexts[start:end]
		embeddings, err := s.embedder.EmbedStrings(ctx, batch...)
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
func (s *KnowledgeRouteImpl) singleSemanticScore(ctx context.Context, q *QueryContext, text string) float64 {
	if len(q.QueryEmbedding) == 0 || s.embedder == nil || utils.IsBlank(text) {
		return 0
	}
	vectors, _ := s.embedder.EmbedStrings(ctx, text)
	if len(vectors) == 0 {
		return 0
	}
	return cosineSimilarity(q.QueryEmbedding, vectors[0])
}

// searchLexicalScores 调用外部词面索引；未配置或失败时回退到本地计算
func (s *KnowledgeRouteImpl) searchLexicalScores(ctx context.Context, routingText, entityType string, size int) map[int64]float64 {
	if s.lexicalIndex == nil {
		return nil
	}
	hits, err := s.lexicalIndex.Search(ctx, routingText, entityType, size)
	if err != nil || len(hits) == 0 {
		return nil
	}
	result := make(map[int64]float64, len(hits))
	for _, hit := range hits {
		result[hit.EntityCode] = hit.Score
	}
	return result
}

// searchDocumentLexicalScores 词面检索文档维度分数（同上）
func (s *KnowledgeRouteImpl) searchDocumentLexicalScores(ctx context.Context, routingText string, size int) map[int64]float64 {
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

// keywordEntityMatchScore 对具有“实体感”的关键词加分：包含字母/数字或较短的中文短语
func (s *KnowledgeRouteImpl) keywordEntityMatchScore(queryTerms []string, routeText string) float64 {
	if len(queryTerms) == 0 {
		return 0
	}
	normalizedContent := normalize(routeText)
	var score float64
	for _, term := range queryTerms {
		if !alpNumPattern.MatchString(term) && utils.Len(term) > 4 {
			continue
		}
		termNorm := normalize(term)
		if utils.Len(termNorm) < 2 {
			continue
		}
		if strings.Contains(normalizedContent, termNorm) {
			score += 6
		}
	}
	return score
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
func (s *KnowledgeRouteImpl) buildReason(queryTerms []string, content string, semanticScore float64) string {
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

// buildCandidateReason 根据语义分、词面分、关系分构建候选原因
func (s *KnowledgeRouteImpl) buildCandidateReason(semanticScore, lexicalScore, relationScore float64) string {
	semanticMatched := semanticMainScore(semanticScore) > 0
	lexicalMatched := lexicalScore > 0
	relationMatched := relationScore > 0

	switch {
	case semanticMatched && lexicalMatched && relationMatched:
		return "语义、路由索引与持久化关系特征共同召回"
	case semanticMatched && lexicalMatched:
		return "语义与路由索引特征共同召回"
	case lexicalMatched && relationMatched:
		return "路由索引与持久化关系特征共同召回"
	case semanticMatched && relationMatched:
		return "语义与持久化关系特征共同召回"
	case lexicalMatched:
		return "由路由索引 BM25 特征召回"
	case semanticMatched:
		return "由持久化画像的语义相似度召回"
	case relationMatched:
		return "由持久化关系特征召回"
	default:
		return "没有形成有效路由特征"
	}
}

// buildFeatures 构建候选特征Map
func (s *KnowledgeRouteImpl) buildFeatures(semanticScore, lexicalScore, relationScore float64) map[string]float64 {
	features := make(map[string]float64, 3)
	features["semanticScore"] = max(0, semanticScore)
	features["routeIndexScore"] = max(0, lexicalScore)
	features["topicRelationScore"] = max(0, relationScore)
	return features
}

// resolveDecisionSource 从文档候选中解析决策来源
func (s *KnowledgeRouteImpl) resolveDecisionSource(documentCandidates []*vo.DocumentRouteCandidate) string {
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
func (s *KnowledgeRouteImpl) resolveDecisionReason(documentCandidates []*vo.DocumentRouteCandidate, confidence float64) string {
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
func (s *KnowledgeRouteImpl) buildTrace(exchangeId int64, conversationId, question, rewriteQuestion, mode string, decision *vo.KnowledgeRouteDecision) *entity.KnowledgeRouteTrace {
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
