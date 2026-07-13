package persistence

import (
	"context"
	"regexp"
	"strings"

	"github.com/duke-git/lancet/v2/strutil"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/convert"
	"github.com/swiftbit/know-agent/internal/domain/knowledge/adapter"
	"github.com/swiftbit/know-agent/internal/domain/knowledge/model/entity"
	"github.com/swiftbit/know-agent/internal/infrastructure/model"
	"github.com/swiftbit/know-agent/internal/svc"
)

// 正则常量（与 logic 层保持一致，避免跨模块循环依赖）
var (
	alnumPattern            = regexp.MustCompile(`[a-z0-9._-]{2,}`)
	chinesePattern          = regexp.MustCompile(`\p{Han}{2,}`)
	chineseNoisePhrasesRepo = []string{
		"请问", "帮我", "一下子", "一下", "如何", "怎么", "什么", "哪个", "这个", "那个", "是否", "关于", "可以", "需要", "想问", "看看",
	}
	chineseSegmentSplitPatternRepo = regexp.MustCompile(`[的和及与或]`)
	spacePatternRepo               = regexp.MustCompile(`\s+`)
)

// KnowledgeRepositoryImpl 文档知识仓储实现
// 其中向量检索接入 Milvus（当前以占位实现 + 过滤/排序注释说明），
// 关键词检索以 SQL/内存扫描形式实现（生产环境应接入 BM25/外部索引）。
type KnowledgeRepositoryImpl struct {
	*transactionManager
}

var _ adapter.KnowledgeRepository = (*KnowledgeRepositoryImpl)(nil)

// NewKnowledgeRepository 构造函数
func NewKnowledgeRepository(svcCtx *svc.ServiceContext) *KnowledgeRepositoryImpl {
	return &KnowledgeRepositoryImpl{
		transactionManager: &transactionManager{db: svcCtx.Db},
	}
}

// ============ 知识范围节点 ============

// SelectKnowledgeScopeNodes 查询所有知识范围节点
func (k *KnowledgeRepositoryImpl) SelectKnowledgeScopeNodes(ctx context.Context) ([]*entity.KnowledgeScopeNode, error) {
	var nodes []*entity.KnowledgeScopeNode
	if err := k.dbWithContext(ctx).Model(&model.KnowledgeScopeNode{}).
		Order("sort_order ASC").
		Find(&nodes).Error; err != nil {
		return nil, err
	}
	return nodes, nil
}

// UpsertKnowledgeScopeNode 插入或更新知识范围节点
func (k *KnowledgeRepositoryImpl) UpsertKnowledgeScopeNode(ctx context.Context, node *entity.KnowledgeScopeNode) error {
	nodeModel := convert.ToKnowledgeScopeNodeModel(node)
	if err := k.dbWithContext(ctx).Model(&model.KnowledgeScopeNode{}).Where("scope_code = ?", node.ScopeCode).First(node).Error; err != nil {
		return err
	}
	nodeModel.ID = node.ID
	return k.dbWithContext(ctx).Save(nodeModel).Error
}

// DeleteKnowledgeScopeNode 删除知识范围节点
func (k *KnowledgeRepositoryImpl) DeleteKnowledgeScopeNode(ctx context.Context, scopeCode string) error {
	if strutil.IsBlank(scopeCode) {
		return nil
	}
	return k.dbWithContext(ctx).Where("scope_code = ?", scopeCode).Delete(&model.KnowledgeScopeNode{}).Error
}

// ============ 主题节点 ============

// SelectKnowledgeTopicNodes 查询所有主题节点
func (k *KnowledgeRepositoryImpl) SelectKnowledgeTopicNodes(ctx context.Context) ([]*entity.KnowledgeTopicNode, error) {
	return k.SelectKnowledgeTopicNodesByScopeCode(ctx, "")
}

// SelectKnowledgeTopicNodesByScopeCode 根据知识范围节点查询所有主题节点
func (k *KnowledgeRepositoryImpl) SelectKnowledgeTopicNodesByScopeCode(ctx context.Context, scopeCode string) ([]*entity.KnowledgeTopicNode, error) {
	var nodes []*entity.KnowledgeTopicNode
	builder := k.dbWithContext(ctx).Model(&model.KnowledgeTopicNode{}).Order("sort_order ASC")
	if strutil.IsNotBlank(scopeCode) {
		builder = builder.Where("scope_code = ?", scopeCode)
	}
	if err := builder.Find(&nodes).Error; err != nil {
		return nil, err
	}
	return nodes, nil
}

// UpsertKnowledgeTopicNode 插入或更新主题节点
func (k *KnowledgeRepositoryImpl) UpsertKnowledgeTopicNode(ctx context.Context, node *entity.KnowledgeTopicNode) error {
	nodeModel := convert.ToKnowledgeTopicNodeModel(node)
	if err := k.dbWithContext(ctx).Model(&model.KnowledgeTopicNode{}).
		Where("topic_code = ?", node.TopicCode).
		First(node).Error; err != nil {
		return err
	}
	nodeModel.ID = node.ID
	return k.dbWithContext(ctx).Save(nodeModel).Error
}

// DeleteKnowledgeTopicNode 删除主题节点
func (k *KnowledgeRepositoryImpl) DeleteKnowledgeTopicNode(ctx context.Context, topicCode string) error {
	if strutil.IsBlank(topicCode) {
		return nil
	}
	return k.dbWithContext(ctx).Model(&model.KnowledgeTopicNode{}).Where("topic_code = ?", topicCode).Delete(nil).Error
}

// ============ 主题-文档关系 ============

// SelectTopicDocumentRelations 查询所有主题-文档关系
func (k *KnowledgeRepositoryImpl) SelectTopicDocumentRelations(ctx context.Context) ([]*entity.KnowledgeTopicDocumentRelation, error) {
	var relations []*entity.KnowledgeTopicDocumentRelation
	if err := k.dbWithContext(ctx).Model(&model.KnowledgeTopicDocumentRelation{}).Find(&relations).Error; err != nil {
		return nil, err
	}
	return relations, nil
}

// SelectTopicDocumentRelationsByTopicCode 根据主题编码查询主题-文档关系
func (k *KnowledgeRepositoryImpl) SelectTopicDocumentRelationsByTopicCode(ctx context.Context, topicCode string) ([]*entity.KnowledgeTopicDocumentRelation, error) {
	var relations []*entity.KnowledgeTopicDocumentRelation
	query := k.dbWithContext(ctx).Model(&model.KnowledgeTopicDocumentRelation{})
	if strutil.IsNotBlank(topicCode) {
		query = query.Where("topic_code = ?", topicCode)
	}
	if err := query.Find(&relations).Error; err != nil {
		return nil, err
	}
	return relations, nil
}

// UpsertTopicDocumentRelation 插入或更新主题-文档关系
func (k *KnowledgeRepositoryImpl) UpsertTopicDocumentRelation(ctx context.Context, relation *entity.KnowledgeTopicDocumentRelation) error {
	relModel := convert.ToKnowledgeTopicDocumentRelationModel(relation)
	if err := k.dbWithContext(ctx).Model(&model.KnowledgeTopicDocumentRelation{}).
		Where("topic_code = ? AND document_id = ?", relation.TopicCode, relation.DocumentId).
		First(relation).Error; err != nil {
		return err
	}
	relModel.ID = relation.ID
	return k.dbWithContext(ctx).Save(relModel).Error
}

// DeleteTopicDocumentRelation 删除主题-文档关系
func (k *KnowledgeRepositoryImpl) DeleteTopicDocumentRelation(ctx context.Context, topicCode string, documentId int64) error {
	return k.dbWithContext(ctx).Where("topic_code = ? AND document_id = ?", topicCode, documentId).
		Delete(&model.KnowledgeTopicDocumentRelation{}).Error
}

// ============ 路由跟踪 ============

// InsertKnowledgeRouteTrace 插入路由跟踪
func (k *KnowledgeRepositoryImpl) InsertKnowledgeRouteTrace(ctx context.Context, trace *entity.KnowledgeRouteTrace) error {
	return k.dbWithContext(ctx).Model(&model.KnowledgeRouteTrace{}).Create(convert.ToKnowledgeRouteTraceModel(trace)).Error
}

// SelectKnowledgeRouteTracePage 分页查询路由跟踪
func (k *KnowledgeRepositoryImpl) SelectKnowledgeRouteTracePage(ctx context.Context, conversationId, mode string, routeStatus, pageNo, pageSize int) ([]*entity.KnowledgeRouteTrace, int64, error) {
	builder := k.dbWithContext(ctx).Model(&model.KnowledgeRouteTrace{})
	if strutil.IsNotBlank(conversationId) {
		builder = builder.Where("conversation_id = ?", conversationId)
	}
	if strutil.IsNotBlank(mode) {
		builder = builder.Where("mode = ?", mode)
	}
	if routeStatus > 0 {
		builder = builder.Where("route_status = ?", routeStatus)
	}

	var total int64
	if err := builder.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []*entity.KnowledgeRouteTrace
	if err := builder.Scopes(utils.Paginate(pageNo, pageSize)).Order("id DESC").Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// ============ 内部辅助函数 ============

// // SearchByKeyword 关键词检索（按子串打分 + topK 排序）
// // 当前以"SQL + 简单关键词权重"的形式实现；生产环境建议替换为 BM25/外部索引。
// func (k *KnowledgeRepositoryImpl) SearchByKeyword(ctx context.Context, query string, documentIDs, taskIDs []int64, topK int, filters *vo2.DocumentRetrieveFilters) ([]*data.EmbeddingChunk, error) {
// 	if len(documentIDs) == 0 || len(taskIDs) == 0 || strutil.IsBlank(query) {
// 		return nil, nil
// 	}
//
// 	// 步骤 1：查询候选 chunk（仅返回必要字段）
// 	var candidateChunks []*data.EmbeddingChunk
// 	builder := k.dbWithContext(ctx).Model(&model.EmbeddingChunk{}).
// 		Where("status = ?", 1).
// 		Where("document_id IN ?", documentIDs).
// 		Where("task_id IN ?", taskIDs)
//
// 	// 步骤 2：应用结构/路径过滤器
// 	if filters != nil {
// 		if len(filters.SectionPathHints) > 0 {
// 			builder = builder.Where(buildLikeOrExpr("section_path", filters.SectionPathHints))
// 		}
// 		if len(filters.StructureNodeIdHints) > 0 {
// 			builder = builder.Where("structure_node_id IN ?", filters.StructureNodeIdHints)
// 		}
// 		if len(filters.CanonicalPathHints) > 0 {
// 			builder = builder.Where(buildLikeOrExpr("canonical_path", filters.CanonicalPathHints))
// 		}
// 		if len(filters.ItemIndexHints) > 0 {
// 			builder = builder.Where("item_index IN ?", filters.ItemIndexHints)
// 		}
// 	}
//
// 	if err := builder.Find(&candidateChunks).Error; err != nil {
// 		return nil, err
// 	}
// 	if len(candidateChunks) == 0 {
// 		return nil, nil
// 	}
//
// 	// 步骤 3：关键词分数计算
// 	terms := extractKeywordTerms(query)
// 	if len(terms) == 0 {
// 		return nil, nil
// 	}
//
// 	scoreChunks := make([]keywordScoreChunk, 0, len(candidateChunks))
// 	for _, chunk := range candidateChunks {
// 		if chunk == nil {
// 			continue
// 		}
// 		score := computeKeywordScore(terms, chunk.ChunkText, chunk.SectionPath)
// 		if score <= 0 {
// 			continue
// 		}
// 		scoreChunks = append(scoreChunks, keywordScoreChunk{chunk: chunk, score: score})
// 	}
//
// 	// 步骤 4：按分数降序取 topK
// 	sortKeywordChunksDesc(scoreChunks)
// 	limit := topK
// 	if limit <= 0 || limit > len(scoreChunks) {
// 		limit = len(scoreChunks)
// 	}
//
// 	result := make([]*data.EmbeddingChunk, 0, limit)
// 	for i := 0; i < limit; i++ {
// 		result = append(result, scoreChunks[i].chunk)
// 	}
// 	return result, nil
// }

func computeKeywordScore(terms []string, chunkText, sectionPath string) float64 {
	if len(terms) == 0 {
		return 0
	}
	lowerText := strings.ToLower(chunkText)
	lowerSection := strings.ToLower(safeText(sectionPath))
	var score float64
	for i, term := range terms {
		weight := keywordWeight(i)
		if strings.Contains(lowerText, strings.ToLower(term)) {
			score += float64(weight)
		}
		if lowerSection != "" && strings.Contains(lowerSection, strings.ToLower(term)) {
			score += float64(sectionKeywordWeight(i))
		}
	}
	return score
}

func buildLikeOrExpr(field string, patterns []string) string {
	// 例如：(LOWER(section_path) LIKE '%a%' OR LOWER(section_path) LIKE '%b%')
	if len(patterns) == 0 {
		return ""
	}
	parts := make([]string, 0, len(patterns))
	for _, p := range patterns {
		if strutil.IsBlank(p) {
			continue
		}
		parts = append(parts, "LOWER("+field+") LIKE '%"+strings.ToLower(strings.TrimSpace(p))+"%'")
	}
	if len(parts) == 0 {
		return ""
	}
	return "(" + strings.Join(parts, " OR ") + ")"
}

const maxKeywordTerms = 8

func extractKeywordTerms(question string) []string {
	normalized := normalizeQuestion(question)
	if strutil.IsBlank(normalized) {
		return nil
	}

	terms := make([]string, 0, maxKeywordTerms*2)
	seen := make(map[string]struct{}, maxKeywordTerms*2)

	// 步骤 1：字母/数字 token
	for _, t := range alnumPattern.FindAllString(normalized, -1) {
		if _, ok := seen[t]; ok {
			continue
		}
		seen[t] = struct{}{}
		terms = append(terms, t)
	}

	// 步骤 2：中文 token + 按分隔符切分 + n-gram 展开
	for _, raw := range chinesePattern.FindAllString(normalized, -1) {
		for _, segment := range splitChineseSegments(raw) {
			addChineseSegmentTerms(segment, &terms, seen)
			if len(terms) >= maxKeywordTerms*2 {
				break
			}
		}
		if len(terms) >= maxKeywordTerms*2 {
			break
		}
	}

	result := make([]string, 0, len(terms))
	for _, t := range terms {
		if len(t) >= 2 {
			result = append(result, t)
		}
		if len(result) >= maxKeywordTerms {
			break
		}
	}
	return result
}

func keywordWeight(index int) int {
	if v := 6 - index; v > 0 {
		return v
	}
	return 1
}

func sectionKeywordWeight(index int) int {
	return keywordWeight(index) + 2
}

func normalizeQuestion(question string) string {
	if strutil.IsBlank(question) {
		return ""
	}
	normalized := strings.ToLower(strutil.Trim(question))
	normalized = spacePatternRepo.ReplaceAllString(normalized, " ")
	return normalized
}

func safeText(text string) string {
	return text
}

func splitChineseSegments(chineseToken string) []string {
	cleaned := chineseToken
	if !strutil.IsBlank(cleaned) {
		normalized := strutil.Trim(cleaned)
		for _, phrase := range chineseNoisePhrasesRepo {
			normalized = strings.ReplaceAll(normalized, phrase, "")
		}
		cleaned = strutil.Trim(normalized)
	}
	if len(cleaned) < 2 {
		return nil
	}
	seen := make(map[string]struct{})
	var segments []string
	if _, ok := seen[cleaned]; !ok {
		seen[cleaned] = struct{}{}
		segments = append(segments, cleaned)
	}
	for _, part := range chineseSegmentSplitPatternRepo.Split(cleaned, -1) {
		normalized := strutil.Trim(part)
		if len(normalized) < 2 {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		segments = append(segments, normalized)
	}
	return segments
}

func addChineseSegmentTerms(segment string, terms *[]string, seen map[string]struct{}) {
	if strutil.IsBlank(segment) || len(segment) < 2 {
		return
	}

	runes := []rune(segment)
	addIfAbsent := func(t string) {
		if _, ok := seen[t]; ok {
			return
		}
		seen[t] = struct{}{}
		*terms = append(*terms, t)
	}
	if len(runes) <= 12 {
		addIfAbsent(segment)
	}
	addTailNgramsRepo(runes, addIfAbsent)
	addHeadNgramsRepo(runes, addIfAbsent)
	addSlidingNgramsRepo(runes, addIfAbsent)
}

func addTailNgramsRepo(runes []rune, add func(string)) {
	maxGram := 4
	if len(runes) < maxGram {
		maxGram = len(runes)
	}
	for size := maxGram; size >= 2; size-- {
		if len(runes)-size < 0 {
			continue
		}
		add(string(runes[len(runes)-size:]))
	}
}

func addHeadNgramsRepo(runes []rune, add func(string)) {
	maxGram := 4
	if len(runes) < maxGram {
		maxGram = len(runes)
	}
	for size := maxGram; size >= 2; size-- {
		add(string(runes[:size]))
	}
}

func addSlidingNgramsRepo(runes []rune, add func(string)) {
	maxGram := 4
	if len(runes) < maxGram {
		maxGram = len(runes)
	}
	for size := maxGram; size >= 2; size-- {
		for i := 0; i <= len(runes)-size; i++ {
			add(string(runes[i : i+size]))
		}
	}
}
