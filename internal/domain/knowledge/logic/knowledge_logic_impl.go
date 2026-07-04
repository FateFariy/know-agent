package logic

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/duke-git/lancet/v2/maputil"
	"github.com/duke-git/lancet/v2/strutil"
	"github.com/zeromicro/go-zero/core/logx"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
	vo2 "github.com/swiftbit/know-agent/internal/domain/document/model/vo"
	"github.com/swiftbit/know-agent/internal/domain/knowledge/adapter"
)

const (
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
	port *adapter.KnowledgePort
}

// NewDocumentKnowledgeService 构造函数
func NewDocumentKnowledgeService(repo adapter.KnowledgeRepository, port *adapter.KnowledgePort) *DocumentKnowledgeLogicImpl {
	return &DocumentKnowledgeLogicImpl{
		repo: repo,
		port: port,
	}
}

// ListRetrievableDocuments 列出可检索的文档
func (s *DocumentKnowledgeLogicImpl) ListRetrievableDocuments(ctx context.Context) ([]*vo2.KnowledgeDocument, error) {
	return s.repo.SelectRetrievableDocuments(ctx)
}

// VectorSearch 向量检索（Milvus 后端 + 过滤器）
// 流程：参数校验 → 构建描述符 map → 调用 Milvus 向量相似度查询（topK + 过滤）→ 组装 vo.DocumentChunk
func (s *DocumentKnowledgeLogicImpl) VectorSearch(ctx context.Context, retrieve *vo.DocumentRetrieve) ([]*vo.DocumentChunk, error) {
	if !retrieve.ValidSearchable() {
		return nil, nil
	}

	knowledgeMap, err := s.getDocumentsMap(ctx, retrieve.DocumentIds)
	if err != nil {
		return nil, err
	}

	documents, err := s.port.SearchByVector(ctx, retrieve.RetrievalQuery, retrieve.DocumentIds, retrieve.TaskIds, resolveTopK(retrieve.TopK), retrieve.Filters)
	if err != nil {
		logx.Errorf("VectorSearch failed: query=%s, err=%v", retrieve.RetrievalQuery, err)
		return nil, err
	}
	for _, document := range documents {
		document.FillKnowledge(knowledgeMap[document.DocumentId])
	}

	return documents, nil
}

// KeywordSearch 关键词检索
// 流程：参数校验 → 提取关键词项 → 调用仓储（SQL 或外部索引）→ 按分数降序组装结果
func (s *DocumentKnowledgeLogicImpl) KeywordSearch(ctx context.Context, retrieve *vo.DocumentRetrieve) ([]*vo.DocumentChunk, error) {
	if !retrieve.ValidSearchable() {
		return nil, nil
	}

	knowledgeMap, err := s.getDocumentsMap(ctx, retrieve.DocumentIds)
	if err != nil {
		return nil, err
	}

	documents, err := s.port.SearchByKeyword(ctx, retrieve.RetrievalQuery, retrieve.DocumentIds, retrieve.TaskIds, resolveTopK(retrieve.TopK), retrieve.Filters)
	if err != nil {
		logx.Errorf("KeywordDB failed: query=%s, err=%v", retrieve.RetrievalQuery, err)
		return nil, err
	}

	for _, document := range documents {
		document.FillKnowledge(knowledgeMap[document.DocumentId])
	}

	return documents, nil
}

// ElevateToParentBlocks 将子文档提升到父块级别，聚合出更完整的证据
// 流程：按 parentBlockId 分组 → 查询父块 → 聚合分数/通道 → 按分数排序
func (s *DocumentKnowledgeLogicImpl) ElevateToParentBlocks(ctx context.Context, childDocuments []*vo.DocumentChunk, maxChars int) ([]*vo.DocumentChunk, error) {
	if len(childDocuments) == 0 {
		return nil, nil
	}

	// 按 parentBlockId 分组，并收集无法被归类的 childDocument 作为 fallback
	childGroupsByParent := make(map[int64][]*vo.DocumentChunk, len(childDocuments))
	fallbackDocuments := make([]*vo.DocumentChunk, 0, len(childDocuments))
	parentBlockIds := make([]int64, 0, len(childDocuments))
	for _, childDocument := range childDocuments {
		parentBlockId := childDocument.ParentBlockId
		if parentBlockId == 0 {
			fallbackDocuments = append(fallbackDocuments, childDocument)
			continue
		}
		childGroupsByParent[parentBlockId] = append(childGroupsByParent[parentBlockId], childDocument)
		if _, exists := childGroupsByParent[parentBlockId]; exists {
			parentBlockIds = append(parentBlockIds, parentBlockId)
		}
	}

	if len(childGroupsByParent) == 0 {
		return fallbackDocuments, nil
	}

	// 查询父块
	parentBlocks, err := s.repo.SelectParentBlocks(ctx, parentBlockIds)
	if err != nil {
		return nil, err
	}
	parentBlockMap := utils.SliceToMapBy(parentBlocks, func(item *entity.DocumentParentBlock) (int64, *entity.DocumentParentBlock) {
		return item.ID, item
	})

	// 构建父级证据文档，或当父块未找到时直接保留子文档
	elevatedDocuments := make([]*vo.DocumentChunk, 0, len(childGroupsByParent)+len(fallbackDocuments))
	for parentId, children := range childGroupsByParent {
		parentBlock, ok := parentBlockMap[parentId]
		if !ok {
			elevatedDocuments = append(elevatedDocuments, children...)
			continue
		}
		elevatedDocuments = append(elevatedDocuments, s.buildParentEvidenceDocument(parentBlock, children, maxChars))
	}
	elevatedDocuments = append(elevatedDocuments, fallbackDocuments...)

	// 排序（分数降序 → 父块编号升序 → chunkNo 升序）
	slices.SortFunc(elevatedDocuments, func(a, b *vo.DocumentChunk) int {
		if a.Score != b.Score {
			return int(b.Score - a.Score)
		} else if a.ParentBlockNo != b.ParentBlockNo {
			return a.ParentBlockNo - b.ParentBlockNo
		}
		return a.ChunkNo - b.ChunkNo
	})

	return elevatedDocuments, nil
}

// getDocumentsMap 获取文档描述符到 documentId 的映射
func (s *DocumentKnowledgeLogicImpl) getDocumentsMap(ctx context.Context, documentIDs []int64) (map[int64]*vo2.KnowledgeDocument, error) {
	documents, err := s.repo.SelectRetrievableDocuments(ctx, documentIDs...)
	if err != nil {
		return nil, err
	}
	descriptorMap := utils.SliceToMapBy(documents, func(t *vo2.KnowledgeDocument) (int64, *vo2.KnowledgeDocument) {
		return t.DocumentId, t
	})
	return descriptorMap, nil
}

// buildParentEvidenceDocument 构建父级证据文档
func (s *DocumentKnowledgeLogicImpl) buildParentEvidenceDocument(parentBlock *entity.DocumentParentBlock, childDocuments []*vo.DocumentChunk, maxChars int) *vo.DocumentChunk {
	if parentBlock == nil || len(childDocuments) == 0 {
		return nil
	}

	// 选出 score 最高的子文档，作为元数据的基础
	bestChild := childDocuments[0]
	for i := 1; i < len(childDocuments); i++ {
		if bestChild.Score < childDocuments[i].Score {
			bestChild = childDocuments[i]
		}
	}

	channelMap := make(map[string]struct{})
	for _, childDocument := range childDocuments {
		channelMap[childDocument.Channel] = struct{}{}
	}
	channels := maputil.Keys(channelMap)

	// 计算父级证据分数
	supportCount := max(0, len(childDocuments)-1)
	supportWeight := min(0.36, float64(supportCount)*0.12)
	multiChannelWeight := utils.Ternary(len(channels) > 1, 0.10, 0.0)
	parentScore := bestChild.Score * (1.0 + supportWeight + multiChannelWeight)

	return &vo.DocumentChunk{
		ID:                fmt.Sprintf("parent-%d", parentBlock.ID),
		Content:           s.renderParentEvidenceText(parentBlock, childDocuments, maxChars),
		ParentBlockId:     parentBlock.ID,
		ParentBlockNo:     parentBlock.ParentNo,
		SectionPath:       parentBlock.SectionPath,
		StructureNodeId:   parentBlock.StructureNodeId,
		StructureNodeType: parentBlock.StructureNodeType,
		CanonicalPath:     parentBlock.CanonicalPath,
		ItemIndex:         parentBlock.ItemIndex,
		OriginalSnippet:   parentBlock.ParentText,
		Score:             parentScore,
		Channel:           utils.Ternary(len(channels) > 1, "hybrid", channels[0]),
	}
}

// renderParentEvidenceText 渲染父级证据文本：[父块内容] + [命中子片段]
func (s *DocumentKnowledgeLogicImpl) renderParentEvidenceText(parentBlock *entity.DocumentParentBlock, childDocuments []*vo.DocumentChunk, maxChars int) string {
	parentText := strutil.Trim(parentBlock.ParentText)

	// 当父块无内容时，使用首条子文档的内容作为回退
	if strutil.IsBlank(parentText) {
		return utils.Ternary(len(childDocuments) > 0, childDocuments[0].OriginalSnippet, "")
	}

	var childSummaryBuilder strings.Builder
	for i, childDocument := range childDocuments {
		if i > 0 {
			childSummaryBuilder.WriteByte('\n')
		}
		childSummaryBuilder.WriteString("- child#")
		childSummaryBuilder.WriteString(strconv.Itoa(childDocument.ChunkNo))
		childSummaryBuilder.WriteString("：")
		childSummaryBuilder.WriteString(trimText(childDocument.OriginalSnippet, 140))
	}

	var composed string
	if childSummaryBuilder.Len() > 0 {
		composed = fmt.Sprintf("[父块内容]\n%s\n\n[命中子片段]\n%s", parentText, childSummaryBuilder.String())
	} else {
		composed = fmt.Sprintf("[父块内容]\n%s", parentText)
	}

	return trimText(composed, max(maxChars, 1))
}

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

// KeywordWeight 关键词命中在 chunk_text 时的基础权重（索引越靠前权重越高）
func KeywordWeight(index int) int {
	return max(1, 6-index)
}

// SectionKeywordWeight 关键词命中在 section_path 时的额外权重（基础权重 + 2）
func SectionKeywordWeight(index int) int {
	return KeywordWeight(index) + 2
}

func resolveTopK(topK int) int {
	if topK <= 0 {
		return 10
	}
	return min(topK, 50)
}

func trimText(text string, maxChars int) string {
	charLen := utf8.RuneCountInString(text)
	if charLen <= maxChars-1 {
		return text
	}
	return text[:max(maxChars-1, 0)] + "…"
}
