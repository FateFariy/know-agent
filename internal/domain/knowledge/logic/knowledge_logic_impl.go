package logic

import (
	"context"
	"fmt"
	"maps"
	"math"
	"regexp"
	"slices"
	"sort"
	"strings"

	"github.com/duke-git/lancet/v2/convertor"
	"github.com/duke-git/lancet/v2/slice"
	"github.com/duke-git/lancet/v2/stream"
	"github.com/duke-git/lancet/v2/strutil"
	"github.com/zeromicro/go-zero/core/logx"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/knowledge/adapter"
	"github.com/swiftbit/know-agent/internal/domain/knowledge/model/data"
	"github.com/swiftbit/know-agent/internal/domain/knowledge/model/vo"
)

// ====== 常量与正则（与 Java DocumentKnowledgeServiceImpl 对齐） ======
const (
	BusinessStatusYes       = 1
	IndexStatusBuildSuccess = 2
	MaxKeywordTerms         = 8
)

var (
	alnumTokenPattern   = regexp.MustCompile(`[a-z0-9._-]{2,}`)
	chineseTokenPattern = regexp.MustCompile(`\p{Han}{2,}`)
	chineseNoisePhrases = []string{
		"请问", "帮我", "一下子", "一下", "如何", "怎么", "什么", "哪个", "这个", "那个", "是否", "关于", "可以", "需要", "想问", "看看",
	}
	chineseSegmentSplitPattern = regexp.MustCompile(`[的和及与或]`)
	spacePattern               = regexp.MustCompile(`\s+`)
)

// DocumentKnowledgeLogicImpl 文档知识服务实现
type DocumentKnowledgeLogicImpl struct {
	repo adapter.KnowledgeRepository
}

// NewDocumentKnowledgeService 构造函数
func NewDocumentKnowledgeService(repo adapter.KnowledgeRepository) *DocumentKnowledgeLogicImpl {
	return &DocumentKnowledgeLogicImpl{repo: repo}
}

// =====================================================
// 公开 API
// =====================================================

// ListRetrievableDocuments 列出可检索的文档
func (s *DocumentKnowledgeLogicImpl) ListRetrievableDocuments(ctx context.Context) ([]*vo.KnowledgeDocument, error) {
	return s.repo.SelectRetrievableDocuments(ctx)
}

// VectorSearch 向量检索（Milvus 后端 + 过滤器）
// 流程：参数校验 → 构建描述符 map → 调用 Milvus 向量相似度查询（topK + 过滤）→ 组装 vo.Document
func (s *DocumentKnowledgeLogicImpl) VectorSearch(ctx context.Context, retrieve *vo.DocumentRetrieve) ([]*vo.Document, error) {
	if !s.validSearchable(retrieve) {
		return nil, nil
	}

	documentIDs := retrieve.ResolvedDocumentIds()
	taskIDs := retrieve.ResolvedTaskIds()
	descriptorMap, err := s.getDocumentsMap(ctx, documentIDs)
	if err != nil {
		return nil, err
	}

	chunks, err := s.repo.SearchByVector(ctx, retrieve.RetrievalQuery, documentIDs, taskIDs, s.resolveTopK(retrieve.TopK), retrieve.Filters)
	if err != nil {
		logx.Errorf("VectorSearch failed: query=%s, err=%v", retrieve.RetrievalQuery, err)
		return nil, err
	}

	return s.buildSearchDocuments(chunks, descriptorMap, "vector"), nil
}

// KeywordSearch 关键词检索
// 流程：参数校验 → 提取关键词项 → 调用仓储（SQL 或外部索引）→ 按分数降序组装结果
func (s *DocumentKnowledgeLogicImpl) KeywordSearch(ctx context.Context, retrieve *vo.DocumentRetrieve) ([]*vo.Document, error) {
	if !s.validSearchable(retrieve) {
		return nil, nil
	}

	documentIDs := retrieve.ResolvedDocumentIds()
	taskIDs := retrieve.ResolvedTaskIds()
	descriptorMap, err := s.getDocumentsMap(ctx, documentIDs)
	if err != nil {
		return nil, err
	}

	chunks, err := s.repo.SearchByKeyword(ctx, retrieve.RetrievalQuery, documentIDs, taskIDs, s.resolveTopK(retrieve.TopK), retrieve.Filters)
	if err != nil {
		logx.Errorf("KeywordSearch failed: query=%s, err=%v", retrieve.RetrievalQuery, err)
		return nil, err
	}

	return s.buildSearchDocuments(chunks, descriptorMap, "keyword"), nil
}

// ElevateToParentBlocks 将子文档提升到父块级别，聚合出更完整的证据
// 流程：按 parentBlockId 分组 → 查询父块 → 聚合分数/通道 → 按分数排序
func (s *DocumentKnowledgeLogicImpl) ElevateToParentBlocks(ctx context.Context, childDocuments []*vo.Document, maxChars int) ([]*vo.Document, error) {
	if len(childDocuments) == 0 {
		return nil, nil
	}

	// 步骤 1：按 parentBlockId 分组，并收集无法被归类的 childDocument 作为 fallback
	childGroupsByParent := make(map[int64][]*vo.Document, len(childDocuments))
	fallbackDocuments := make([]*vo.Document, 0, len(childDocuments))
	parentBlockIDs := make([]int64, 0, len(childDocuments))
	for _, childDocument := range childDocuments {
		if childDocument == nil {
			continue
		}
		parentBlockID, err := convertor.ToInt(childDocument.Meta[vo.MetaParentBlockID])
		if err != nil || parentBlockID == 0 {
			fallbackDocuments = append(fallbackDocuments, childDocument)
			continue
		}
		if _, exists := childGroupsByParent[parentBlockID]; !exists {
			parentBlockIDs = append(parentBlockIDs, parentBlockID)
		}
		childGroupsByParent[parentBlockID] = append(childGroupsByParent[parentBlockID], childDocument)
	}

	if len(childGroupsByParent) == 0 {
		return fallbackDocuments, nil
	}

	// 步骤 2：查询父块
	parentBlocks, err := s.repo.SelectParentBlocks(ctx, parentBlockIDs)
	if err != nil {
		return nil, err
	}
	parentBlockMap := utils.SliceToMapBy(parentBlocks, func(item *entity.DocumentParentBlock) (int64, *entity.DocumentParentBlock) {
		return item.ID, item
	})

	// 步骤 3：构建父级证据文档，或当父块未找到时直接保留子文档
	elevatedDocuments := make([]*vo.Document, 0, len(childGroupsByParent)+len(fallbackDocuments))
	for parentID, children := range childGroupsByParent {
		parentBlock, ok := parentBlockMap[parentID]
		if !ok {
			elevatedDocuments = append(elevatedDocuments, children...)
			continue
		}
		elevatedDocuments = append(elevatedDocuments, s.buildParentEvidenceDocument(parentBlock, children, maxChars))
	}
	elevatedDocuments = append(elevatedDocuments, fallbackDocuments...)

	// 步骤 4：排序（分数降序 → 父块编号升序 → chunkNo 升序）
	sort.Slice(elevatedDocuments, func(i, j int) bool {
		return s.compareEvidenceDocument(elevatedDocuments[i], elevatedDocuments[j]) < 0
	})

	return elevatedDocuments, nil
}

// =====================================================
// 内部辅助：校验与映射
// =====================================================

func (s *DocumentKnowledgeLogicImpl) validSearchable(retrieve *vo.DocumentRetrieve) bool {
	if retrieve == nil || strutil.IsBlank(retrieve.Question) || strutil.IsBlank(retrieve.RetrievalQuery) {
		return false
	}
	return len(retrieve.ResolvedDocumentIds()) > 0 && len(retrieve.ResolvedTaskIds()) > 0
}

// getDocumentsMap 获取文档描述符到 documentId 的映射
func (s *DocumentKnowledgeLogicImpl) getDocumentsMap(ctx context.Context, documentIDs []int64) (map[int64]*vo.KnowledgeDocument, error) {
	documents, err := s.repo.SelectRetrievableDocuments(ctx, documentIDs...)
	if err != nil {
		return nil, err
	}
	descriptorMap := make(map[int64]*vo.KnowledgeDocument, len(documents))
	for _, doc := range documents {
		if doc == nil {
			continue
		}
		descriptorMap[doc.DocumentId] = doc
	}
	return descriptorMap, nil
}

// =====================================================
// 内部辅助：构建 vo.Document
// =====================================================

// buildSearchDocuments 将 EmbeddingChunk 切片转换为 vo.Document 切片
func (s *DocumentKnowledgeLogicImpl) buildSearchDocuments(chunks []*data.EmbeddingChunk, descriptorMap map[int64]*vo.KnowledgeDocument, channel string) []*vo.Document {
	result := make([]*vo.Document, 0, len(chunks))
	for _, chunk := range chunks {
		descriptor := descriptorMap[chunk.DocumentId]
		result = append(result, s.buildRetrievedDocument(chunk, descriptor, channel))
	}
	return result
}

// buildRetrievedDocument 构建单条 vo.Document（对应 Java 的 buildRetrievedDocument）
func (s *DocumentKnowledgeLogicImpl) buildRetrievedDocument(chunk *data.EmbeddingChunk, descriptor *vo.KnowledgeDocument, channel string) *vo.Document {
	if chunk == nil {
		return nil
	}

	meta := make(map[string]any, 16)

	meta[vo.MetaSourceType] = "DOCUMENT"
	meta[vo.MetaChannel] = channel
	meta[vo.MetaScore] = 0.0
	meta[vo.MetaChunkID] = chunk.ID
	meta[vo.MetaDocumentID] = chunk.DocumentId
	meta[vo.MetaTaskID] = chunk.TaskId
	meta[vo.MetaParentBlockID] = chunk.ParentBlockId
	meta[vo.MetaChunkNo] = chunk.ChunkNo
	meta[vo.MetaSectionPath] = safeText(chunk.SectionPath)
	if chunk.StructureNodeId != 0 {
		meta[vo.MetaStructureNodeID] = chunk.StructureNodeId
	}
	if chunk.StructureNodeType != 0 {
		meta[vo.MetaStructureNodeType] = chunk.StructureNodeType
	}
	meta[vo.MetaCanonicalPath] = safeText(chunk.CanonicalPath)
	if chunk.ItemIndex != 0 {
		meta[vo.MetaItemIndex] = chunk.ItemIndex
	}
	meta[vo.MetaOriginalSnippet] = chunk.ChunkText

	if descriptor != nil {
		meta[vo.MetaDocumentName] = safeText(descriptor.DocumentName)
		meta[vo.MetaKnowledgeScopeCode] = safeText(descriptor.KnowledgeScopeCode)
		meta[vo.MetaKnowledgeScopeName] = safeText(descriptor.KnowledgeScopeName)
		meta[vo.MetaBusinessCategory] = safeText(descriptor.BusinessCategory)
		meta[vo.MetaDocumentTags] = safeText(descriptor.DocumentTags)
	}

	return &vo.Document{
		ID:      fmt.Sprintf("%d", chunk.ID),
		Content: chunk.ChunkText,
		Meta:    meta,
		Score:   0.0,
	}
}

// =====================================================
// 内部辅助：父级证据构建
// =====================================================

// buildParentEvidenceDocument 构建父级证据文档（聚合 score/channel）
func (s *DocumentKnowledgeLogicImpl) buildParentEvidenceDocument(parentBlock *entity.DocumentParentBlock, childDocuments []*vo.Document, maxChars int) *vo.Document {
	if parentBlock == nil || len(childDocuments) == 0 {
		return nil
	}

	// 选出 score 最高的子文档，作为元数据的基础
	bestChild := slices.MaxFunc(childDocuments, func(a, b *vo.Document) int {
		scoreA := resolveScoreOrZero(a)
		scoreB := resolveScoreOrZero(b)
		if math.Abs(scoreA-scoreB) < 1e-9 {
			return 0
		}
		if scoreA > scoreB {
			return 1
		}
		return -1
	})

	supportCount := max(0, len(childDocuments)-1)
	channels := s.extractChannels(childDocuments)
	supportWeight := min(0.36, float64(supportCount)*0.12)
	multiChannelWeight := utils.Ternary(len(channels) > 1, 0.10, 0.0)
	parentScore := resolveScoreOrZero(bestChild) * (1.0 + supportWeight + multiChannelWeight)

	// 复制 bestChild 的元数据，再覆盖父块专属字段
	meta := make(map[string]any, len(bestChild.Meta)+8)
	maps.Copy(meta, bestChild.Meta)

	meta[vo.MetaParentBlockID] = parentBlock.ID
	meta[vo.MetaParentBlockNo] = parentBlock.ParentNo
	meta[vo.MetaSectionPath] = safeText(parentBlock.SectionPath)
	if parentBlock.StructureNodeId != 0 {
		meta[vo.MetaStructureNodeID] = parentBlock.StructureNodeId
	}
	if parentBlock.StructureNodeType != 0 {
		meta[vo.MetaStructureNodeType] = parentBlock.StructureNodeType
	}
	meta[vo.MetaCanonicalPath] = safeText(parentBlock.CanonicalPath)
	if parentBlock.ItemIndex != 0 {
		meta[vo.MetaItemIndex] = parentBlock.ItemIndex
	}
	meta[vo.MetaScore] = parentScore
	meta[vo.MetaOriginalSnippet] = safeText(parentBlock.ParentText)

	if len(channels) > 1 {
		meta[vo.MetaChannel] = "hybrid"
	} else if len(channels) == 1 {
		meta[vo.MetaChannel] = channels[0]
	} else {
		meta[vo.MetaChannel] = "vector"
	}

	return &vo.Document{
		ID:      fmt.Sprintf("parent-%d", parentBlock.ID),
		Content: s.renderParentEvidenceText(parentBlock, childDocuments, maxChars),
		Meta:    meta,
		Score:   parentScore,
	}
}

// extractChannels 提取子文档的渠道集合（去重、非空）
func (s *DocumentKnowledgeLogicImpl) extractChannels(childDocuments []*vo.Document) []string {
	channels := slice.Map(childDocuments, func(_ int, item *vo.Document) string {
		if item == nil || item.Meta == nil {
			return ""
		}
		return asText(item.Meta[vo.MetaChannel])
	})
	return stream.FromSlice(channels).
		Filter(func(item string) bool { return item != "" }).
		Distinct().ToSlice()
}

// renderParentEvidenceText 渲染父级证据文本：[父块内容] + [命中子片段]
func (s *DocumentKnowledgeLogicImpl) renderParentEvidenceText(parentBlock *entity.DocumentParentBlock, childDocuments []*vo.Document, maxChars int) string {
	parentText := strutil.Trim(safeText(parentBlock.ParentText))

	// 当父块无内容时，使用首条子文档的内容作为回退
	if strutil.IsBlank(parentText) {
		if len(childDocuments) > 0 && childDocuments[0] != nil {
			return childDocuments[0].Content
		}
		return ""
	}

	var childSummaryBuilder strings.Builder
	for i, childDocument := range childDocuments {
		if childDocument == nil {
			continue
		}
		if i > 0 {
			childSummaryBuilder.WriteByte('\n')
		}
		chunkNo, _ := convertor.ToInt(childDocument.Meta[vo.MetaChunkNo])
		childSummaryBuilder.WriteString(fmt.Sprintf("- child#%d：%s", chunkNo, trimText(safeText(childDocument.Content), 140)))
	}

	var composed string
	if childSummaryBuilder.Len() > 0 {
		composed = fmt.Sprintf("[父块内容]\n%s\n\n[命中子片段]\n%s", parentText, childSummaryBuilder.String())
	} else {
		composed = fmt.Sprintf("[父块内容]\n%s", parentText)
	}

	return trimText(composed, max(maxChars, 1))
}

// compareEvidenceDocument 比较两条证据文档（分数降序 → parentNo 升序 → chunkNo 升序）
func (s *DocumentKnowledgeLogicImpl) compareEvidenceDocument(left, right *vo.Document) int {
	leftScore := resolveScoreOrZero(left)
	rightScore := resolveScoreOrZero(right)
	if diff := rightScore - leftScore; math.Abs(diff) > 1e-9 {
		if diff > 0 {
			return -1
		}
		return 1
	}

	leftParentNo := asIntegerOrZero(safeMetaValue(left, vo.MetaParentBlockNo))
	rightParentNo := asIntegerOrZero(safeMetaValue(right, vo.MetaParentBlockNo))
	if leftParentNo != rightParentNo {
		return int(leftParentNo - rightParentNo)
	}

	leftChunkNo := asIntegerOrZero(safeMetaValue(left, vo.MetaChunkNo))
	rightChunkNo := asIntegerOrZero(safeMetaValue(right, vo.MetaChunkNo))
	return int(leftChunkNo - rightChunkNo)
}

// =====================================================
// 关键词提取（对应 Java extractKeywordTerms / splitChineseSegments / addChineseSegmentTerms 等）
// =====================================================

// ExtractKeywordTerms 从查询句中提取最多 MaxKeywordTerms 个关键词项
func (s *DocumentKnowledgeLogicImpl) ExtractKeywordTerms(question string) []string {
	normalized := normalizeQuestion(question)
	if strutil.IsBlank(normalized) {
		return nil
	}

	terms := make([]string, 0, MaxKeywordTerms*2)
	seen := make(map[string]struct{}, MaxKeywordTerms*2)

	// 步骤 1：提取字母/数字 token
	for _, t := range alnumTokenPattern.FindAllString(normalized, -1) {
		if _, ok := seen[t]; ok {
			continue
		}
		seen[t] = struct{}{}
		terms = append(terms, t)
	}

	// 步骤 2：提取中文 token → 按分割符拆分 → 再补充 n-gram 子段
	for _, raw := range chineseTokenPattern.FindAllString(normalized, -1) {
		for _, segment := range splitChineseSegments(raw) {
			addChineseSegmentTerms(segment, &terms, seen)
			if len(terms) >= MaxKeywordTerms*2 {
				break
			}
		}
		if len(terms) >= MaxKeywordTerms*2 {
			break
		}
	}

	// 步骤 3：长度 >=2 过滤 + 限制数量
	result := make([]string, 0, len(terms))
	for _, t := range terms {
		if len(t) >= 2 {
			result = append(result, t)
		}
		if len(result) >= MaxKeywordTerms {
			break
		}
	}
	return result
}

// splitChineseSegments 去除噪音短语后再按分隔符切分，所有片段按长度 >=2 保留
func splitChineseSegments(chineseToken string) []string {
	cleaned := removeChineseNoisePhrases(chineseToken)
	if len(cleaned) < 2 {
		return nil
	}

	seen := make(map[string]struct{})
	var segments []string
	if _, ok := seen[cleaned]; !ok {
		seen[cleaned] = struct{}{}
		segments = append(segments, cleaned)
	}
	for _, part := range chineseSegmentSplitPattern.Split(cleaned, -1) {
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

// addChineseSegmentTerms 为中文段补充：原词 + head n-gram + tail n-gram + sliding n-gram
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

	addTailNgrams(runes, addIfAbsent)
	addHeadNgrams(runes, addIfAbsent)
	addSlidingNgrams(runes, addIfAbsent)
}

func addTailNgrams(runes []rune, add func(string)) {
	maxGram := min(4, len(runes))
	for size := maxGram; size >= 2; size-- {
		if len(runes)-size < 0 {
			continue
		}
		add(string(runes[len(runes)-size:]))
	}
}

func addHeadNgrams(runes []rune, add func(string)) {
	maxGram := min(4, len(runes))
	for size := maxGram; size >= 2; size-- {
		add(string(runes[:size]))
	}
}

func addSlidingNgrams(runes []rune, add func(string)) {
	maxGram := min(4, len(runes))
	for size := maxGram; size >= 2; size-- {
		for i := 0; i <= len(runes)-size; i++ {
			add(string(runes[i : i+size]))
		}
	}
}

// normalizeQuestion 标准化查询：去除换行/制表符、合并空白、转为小写
func normalizeQuestion(question string) string {
	if strutil.IsBlank(question) {
		return ""
	}
	normalized := strings.ToLower(strutil.Trim(question))
	normalized = spacePattern.ReplaceAllString(normalized, " ")
	return normalized
}

// removeChineseNoisePhrases 从文本中去除常见的中文噪音短语
func removeChineseNoisePhrases(text string) string {
	if strutil.IsBlank(text) {
		return ""
	}
	normalized := strutil.Trim(text)
	for _, phrase := range chineseNoisePhrases {
		normalized = strings.ReplaceAll(normalized, phrase, "")
	}
	return strutil.Trim(normalized)
}

// =====================================================
// 关键词权重（与 Java keywordWeight / sectionKeywordWeight 对齐）
// =====================================================

// KeywordWeight 关键词命中在 chunk_text 时的基础权重（索引越靠前权重越高）
func KeywordWeight(index int) int {
	return max(1, 6-index)
}

// SectionKeywordWeight 关键词命中在 section_path 时的额外权重（基础权重 + 2）
func SectionKeywordWeight(index int) int {
	return KeywordWeight(index) + 2
}

// =====================================================
// 小型工具函数
// =====================================================

func resolveTopK(topK int) int {
	if topK <= 0 {
		return 10
	}
	return min(topK, 50)
}

func trimText(text string, maxChars int) string {
	if len(text) <= maxChars {
		return text
	}
	maxChars = max(maxChars, 1)
	if maxChars-1 >= len(text) {
		return text
	}
	return text[:maxChars-1] + "…"
}

func safeText(text string) string {
	return text
}

func asText(value any) string {
	if value == nil {
		return ""
	}
	switch v := value.(type) {
	case string:
		return v
	case fmt.Stringer:
		return v.String()
	default:
		return fmt.Sprintf("%v", v)
	}
}

func asIntegerOrZero(value any) int64 {
	if value == nil {
		return 0
	}
	switch v := value.(type) {
	case int:
		return int64(v)
	case int32:
		return int64(v)
	case int64:
		return v
	case float64:
		return int64(v)
	case float32:
		return int64(v)
	default:
		n, err := convertor.ToInt(value)
		if err != nil {
			return 0
		}
		return n
	}
}

func resolveScoreOrZero(doc *vo.Document) float64 {
	if doc == nil {
		return 0
	}
	if doc.Meta != nil {
		if v, ok := doc.Meta[vo.MetaScore]; ok {
			switch n := v.(type) {
			case float64:
				return n
			case float32:
				return float64(n)
			case int:
				return float64(n)
			case int64:
				return float64(n)
			}
		}
	}
	return doc.Score
}

func safeMetaValue(doc *vo.Document, key string) any {
	if doc == nil || doc.Meta == nil {
		return nil
	}
	return doc.Meta[key]
}
