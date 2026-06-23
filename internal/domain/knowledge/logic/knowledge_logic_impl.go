package logic

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/duke-git/lancet/v2/strutil"

	"github.com/zeromicro/go-zero/core/logx"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/knowledge/adapter"
	"github.com/swiftbit/know-agent/internal/domain/knowledge/model/data"
	"github.com/swiftbit/know-agent/internal/domain/knowledge/model/vo"
)

const (
	BusinessStatusYes       = 1
	IndexStatusBuildSuccess = 2
	MaxKeywordTerms         = 8
)

var (
	alnumTokenPattern   = regexp.MustCompile(`[a-z0-9._-]{2,}`)
	chineseTokenPattern = regexp.MustCompile(`[\p{Han}]{2,}`)
	chineseNoisePhrases = []string{
		"请问", "帮我", "一下子", "一下", "如何", "怎么", "什么", "哪个", "这个", "那个", "是否", "关于", "可以", "需要", "想问", "看看",
	}
	chineseSegmentSplitPattern = regexp.MustCompile(`[的和及与或]`)
)

type DocumentKnowledgeLogicImpl struct {
	repo adapter.KnowledgeRepository
}

func NewDocumentKnowledgeService(repo adapter.KnowledgeRepository) *DocumentKnowledgeLogicImpl {
	return &DocumentKnowledgeLogicImpl{
		repo: repo,
	}
}

// ListRetrievableDocuments 列出可检索的文档
func (s *DocumentKnowledgeLogicImpl) ListRetrievableDocuments(ctx context.Context) ([]*vo.KnowledgeDocument, error) {
	return s.repo.SelectAllDocuments(ctx)
}

// VectorSearch 向量检索
func (s *DocumentKnowledgeLogicImpl) VectorSearch(ctx context.Context, retrieve *vo.DocumentRetrieve) ([]*vo.Document, error) {
	if !s.validSearchable(retrieve) {
		return nil, nil
	}

	documentIDs := retrieve.ResolvedDocumentIDs()
	taskIDs := retrieve.ResolvedTaskIDs()
	knowledgeMap, err := s.getDocumentsMap(ctx, documentIDs)
	if err != nil {
		return nil, err
	}

	chunks, err := s.repo.SearchByVector(ctx, retrieve.RetrievalQuery, documentIDs, taskIDs, s.resolveTopK(retrieve.TopK), retrieve.Filters)
	if err != nil {
		logx.Errorf("Vector search failed: %v", err)
		return nil, err
	}

	return s.buildSearchDocuments(chunks, knowledgeMap, "vector"), nil
}

// KeywordSearch 关键词检索
func (s *DocumentKnowledgeLogicImpl) KeywordSearch(ctx context.Context, retrieve *vo.DocumentRetrieve) ([]*vo.Document, error) {
	if !s.validSearchable(retrieve) {
		return nil, nil
	}

	documentIDs := retrieve.ResolvedDocumentIDs()
	taskIDs := retrieve.ResolvedTaskIDs()
	descriptorMap, err := s.getDocumentsMap(ctx, documentIDs)
	if err != nil {
		return nil, err
	}

	chunks, err := s.repo.SearchByKeyword(ctx, retrieve.RetrievalQuery, documentIDs, taskIDs, s.resolveTopK(retrieve.TopK), retrieve.Filters)
	if err != nil {
		logx.Errorf("Keyword search failed: %v", err)
		return nil, err
	}

	return s.buildSearchDocuments(chunks, descriptorMap, "keyword"), nil
}

// ElevateToParentBlocks 提升到父级块
func (s *DocumentKnowledgeLogicImpl) ElevateToParentBlocks(ctx context.Context, childDocuments []*vo.Document, maxChars int) ([]*vo.Document, error) {
	if len(childDocuments) == 0 {
		return []*vo.Document{}, nil
	}

	childGroupsByParent := make(map[int64][]*vo.Document)
	fallbackDocuments := make([]*vo.Document, 0)

	for _, childDocument := range childDocuments {
		if childDocument == nil {
			continue
		}
		parentBlockID := s.asInt64(childDocument.Meta[vo.MetaParentBlockID])
		if parentBlockID == nil || *parentBlockID == 0 {
			fallbackDocuments = append(fallbackDocuments, childDocument)
			continue
		}
		childGroupsByParent[*parentBlockID] = append(childGroupsByParent[*parentBlockID], childDocument)
	}

	if len(childGroupsByParent) == 0 {
		return fallbackDocuments, nil
	}

	parentBlockIDs := make([]int64, 0, len(childGroupsByParent))
	for id := range childGroupsByParent {
		parentBlockIDs = append(parentBlockIDs, id)
	}

	parentBlocks, err := s.repo.GetParentBlocks(ctx, parentBlockIDs)
	if err != nil {
		logx.Errorf("Get parent blocks failed: %v", err)
		return nil, err
	}

	parentBlockMap := make(map[int64]*data.SuperAgentDocumentParentBlock)
	for _, pb := range parentBlocks {
		parentBlockMap[pb.ID] = pb
	}

	elevatedDocuments := make([]*vo.Document, 0, len(childGroupsByParent)+len(fallbackDocuments))
	for parentID, children := range childGroupsByParent {
		parentBlock := parentBlockMap[parentID]
		if parentBlock == nil {
			elevatedDocuments = append(elevatedDocuments, children...)
			continue
		}
		elevatedDocuments = append(elevatedDocuments, s.buildParentEvidenceDocument(parentBlock, children, maxChars))
	}
	elevatedDocuments = append(elevatedDocuments, fallbackDocuments...)

	sort.Slice(elevatedDocuments, func(i, j int) bool {
		return s.compareEvidenceDocument(elevatedDocuments[i], elevatedDocuments[j]) < 0
	})

	return elevatedDocuments, nil
}

func (s *DocumentKnowledgeLogicImpl) validSearchable(retrieve *vo.DocumentRetrieve) bool {
	if retrieve == nil || strutil.IsBlank(retrieve.Question) || strutil.IsBlank(retrieve.RetrievalQuery) {
		return false
	}
	return len(retrieve.ResolvedDocumentIDs()) > 0 && len(retrieve.ResolvedTaskIDs()) > 0
}

func (s *DocumentKnowledgeLogicImpl) getDocumentsMap(ctx context.Context, documentIDs []int64) (map[int64]*vo.KnowledgeDocument, error) {
	documents, err := s.repo.SelectDocumentsByIDs(ctx, documentIDs)
	if err != nil {
		return nil, err
	}

	knowledgeMap := utils.SliceToMapBy(documents, func(item *vo.KnowledgeDocument) (int64, *vo.KnowledgeDocument) {
		return item.DocumentId, item
	})
	return knowledgeMap, nil
}

func (s *DocumentKnowledgeLogicImpl) buildSearchDocuments(chunks []*data.EmbeddingChunk, descriptorMap map[int64]*vo.KnowledgeDocument, channel string) []*vo.Document {
	result := make([]*vo.Document, 0, len(chunks))
	for _, chunk := range chunks {
		descriptor := descriptorMap[chunk.DocumentId]
		doc := s.buildRetrievedDocument(chunk, descriptor, channel)
		result = append(result, doc)
	}
	return result
}

func (s *DocumentKnowledgeLogicImpl) buildRetrievedDocument(chunk *data.EmbeddingChunk, descriptor *vo.KnowledgeDocument, channel string) *vo.Document {
	meta := make(map[string]interface{})

	meta[vo.MetaSourceType] = "DOCUMENT"
	meta[vo.MetaChannel] = channel
	meta[vo.MetaScore] = 0.0
	meta[vo.MetaChunkID] = chunk.ID
	meta[vo.MetaDocumentID] = chunk.DocumentId
	meta[vo.MetaTaskID] = chunk.TaskId
	meta[vo.MetaParentBlockID] = chunk.ParentBlockId
	meta[vo.MetaChunkNo] = chunk.ChunkNo
	meta[vo.MetaSectionPath] = chunk.SectionPath

	if chunk.StructureNodeId != 0 {
		meta[vo.MetaStructureNodeID] = chunk.StructureNodeId
	}
	if chunk.StructureNodeType != 0 {
		meta[vo.MetaStructureNodeType] = chunk.StructureNodeType
	}
	meta[vo.MetaCanonicalPath] = chunk.CanonicalPath
	if chunk.ItemIndex != 0 {
		meta[vo.MetaItemIndex] = chunk.ItemIndex
	}
	meta[vo.MetaOriginalSnippet] = chunk.ChunkText

	if descriptor != nil {
		meta[vo.MetaDocumentName] = descriptor.DocumentName
		meta[vo.MetaKnowledgeScopeCode] = descriptor.KnowledgeScopeCode
		meta[vo.MetaKnowledgeScopeName] = descriptor.KnowledgeScopeName
		meta[vo.MetaBusinessCategory] = descriptor.BusinessCategory
		meta[vo.MetaDocumentTags] = descriptor.DocumentTags
	}

	return &vo.Document{
		ID:      fmt.Sprintf("%d", chunk.ID),
		Content: chunk.ChunkText,
		Meta:    meta,
		Score:   0.0,
	}
}

func (s *DocumentKnowledgeLogicImpl) buildParentEvidenceDocument(parentBlock *data.SuperAgentDocumentParentBlock, childDocuments []*vo.Document, maxChars int) *vo.Document {
	bestChild := s.findBestChild(childDocuments)
	parentScore := s.aggregateParentScore(childDocuments)

	meta := make(map[string]interface{})
	if bestChild != nil {
		for k, v := range bestChild.Meta {
			meta[k] = v
		}
	}

	meta[vo.MetaParentBlockID] = parentBlock.ID
	meta[vo.MetaParentBlockNo] = parentBlock.ParentNo
	meta[vo.MetaSectionPath] = parentBlock.SectionPath
	if parentBlock.StructureNodeId != 0 {
		meta[vo.MetaStructureNodeID] = parentBlock.StructureNodeId
	}
	if parentBlock.StructureNodeType != 0 {
		meta[vo.MetaStructureNodeType] = parentBlock.StructureNodeType
	}
	meta[vo.MetaCanonicalPath] = parentBlock.CanonicalPath
	if parentBlock.ItemIndex != 0 {
		meta[vo.MetaItemIndex] = parentBlock.ItemIndex
	}
	meta[vo.MetaScore] = parentScore
	meta[vo.MetaOriginalSnippet] = parentBlock.ParentText

	channels := s.extractChannels(childDocuments)
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

func (s *DocumentKnowledgeLogicImpl) findBestChild(childDocuments []*vo.Document) *vo.Document {
	if len(childDocuments) == 0 {
		return nil
	}
	best := childDocuments[0]
	bestScore := s.resolveScore(best)
	for i := 1; i < len(childDocuments); i++ {
		score := s.resolveScore(childDocuments[i])
		if score > bestScore {
			bestScore = score
			best = childDocuments[i]
		}
	}
	return best
}

func (s *DocumentKnowledgeLogicImpl) aggregateParentScore(childDocuments []*vo.Document) float64 {
	bestChildScore := 0.0
	for _, doc := range childDocuments {
		score := s.resolveScore(doc)
		if score > bestChildScore {
			bestChildScore = score
		}
	}

	supportCount := max(0, len(childDocuments)-1)
	channels := s.extractChannels(childDocuments)

	supportWeight := min(0.36, float64(supportCount)*0.12)
	multiChannelWeight := 0.0
	if len(channels) > 1 {
		multiChannelWeight = 0.10
	}

	return bestChildScore * (1.0 + supportWeight + multiChannelWeight)
}

func (s *DocumentKnowledgeLogicImpl) extractChannels(childDocuments []*vo.Document) []string {
	channelSet := make(map[string]bool)
	for _, doc := range childDocuments {
		if doc == nil {
			continue
		}
		channel, ok := doc.Meta[vo.MetaChannel].(string)
		if ok && channel != "" {
			channelSet[channel] = true
		}
	}
	result := make([]string, 0, len(channelSet))
	for ch := range channelSet {
		result = append(result, ch)
	}
	return result
}

func (s *DocumentKnowledgeLogicImpl) renderParentEvidenceText(parentBlock *data.SuperAgentDocumentParentBlock, childDocuments []*vo.Document, maxChars int) string {
	parentText := parentBlock.ParentText
	if parentText == "" {
		if len(childDocuments) == 0 {
			return ""
		}
		return childDocuments[0].Content
	}

	var hitSummaryBuilder strings.Builder
	for i, childDocument := range childDocuments {
		if childDocument == nil {
			continue
		}
		if i > 0 {
			hitSummaryBuilder.WriteByte('\n')
		}
		chunkNo := s.asInt(childDocument.Meta[vo.MetaChunkNo])
		if chunkNo == nil {
			*chunkNo = 0
		}
		hitSummaryBuilder.WriteString(fmt.Sprintf("- child#%d：%s", *chunkNo, s.trimText((childDocument.Content), 140)))
	}

	var composed string
	if hitSummaryBuilder.Len() > 0 {
		composed = fmt.Sprintf("[父块内容]\n%s\n\n[命中子片段]\n%s", parentText, hitSummaryBuilder.String())
	} else {
		composed = fmt.Sprintf("[父块内容]\n%s", parentText)
	}

	return s.trimText(composed, max(maxChars, 1))
}

func (s *DocumentKnowledgeLogicImpl) resolveScore(document *vo.Document) float64 {
	if document == nil {
		return 0.0
	}
	if document.Score > 0 {
		return document.Score
	}
	metaScore, ok := document.Meta[vo.MetaScore].(float64)
	if ok {
		return metaScore
	}
	return 0.0
}

func (s *DocumentKnowledgeLogicImpl) compareEvidenceDocument(left, right *vo.Document) int {
	leftScore := s.resolveScore(left)
	rightScore := s.resolveScore(right)
	if rightScore > leftScore {
		return -1
	}
	if rightScore < leftScore {
		return 1
	}

	leftParentNo := s.asInt(left.Meta[vo.MetaParentBlockNo])
	rightParentNo := s.asInt(right.Meta[vo.MetaParentBlockNo])
	parentNoCompare := s.compareNullableInt(leftParentNo, rightParentNo)
	if parentNoCompare != 0 {
		return parentNoCompare
	}

	leftChunkNo := s.asInt(left.Meta[vo.MetaChunkNo])
	rightChunkNo := s.asInt(right.Meta[vo.MetaChunkNo])
	return s.compareNullableInt(leftChunkNo, rightChunkNo)
}

func (s *DocumentKnowledgeLogicImpl) compareNullableInt(left, right *int) int {
	if left == nil && right == nil {
		return 0
	}
	if left == nil {
		return 1
	}
	if right == nil {
		return -1
	}
	if *left < *right {
		return -1
	}
	if *left > *right {
		return 1
	}
	return 0
}

func (s *DocumentKnowledgeLogicImpl) trimText(text string, maxChars int) string {
	if text == "" || len(text) <= maxChars {
		return text
	}
	if maxChars <= 1 {
		return text[:1] + "…"
	}
	return text[:maxChars-1] + "…"
}

func (s *DocumentKnowledgeLogicImpl) asInt64(value interface{}) *int64 {
	if value == nil {
		return nil
	}
	switch v := value.(type) {
	case int64:
		return &v
	case int:
		iv := int64(v)
		return &iv
	case float64:
		iv := int64(v)
		return &iv
	default:
		return nil
	}
}

func (s *DocumentKnowledgeLogicImpl) asInt(value interface{}) *int {
	if value == nil {
		return nil
	}
	switch v := value.(type) {
	case int:
		return &v
	case int64:
		iv := int(v)
		return &iv
	case float64:
		iv := int(v)
		return &iv
	default:
		return nil
	}
}

func (s *DocumentKnowledgeLogicImpl) resolveTopK(topK int) int {
	if topK <= 0 {
		return 10
	}
	return min(topK, 50)
}
