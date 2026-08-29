package retrieval

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/chat/adapter"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
)

// ParentElevationStage 父块提升阶段，将子文档提升到父块级别，聚合出更完整的证据
type ParentElevationStage struct {
	docGateway adapter.DocumentGateway
	maxChars   int
}

// NewParentElevationStage 创建父块提升阶段
func NewParentElevationStage(docGateway adapter.DocumentGateway, maxChars int) *ParentElevationStage {
	return &ParentElevationStage{
		docGateway: docGateway,
		maxChars:   maxChars,
	}
}

func (s *ParentElevationStage) Name() string {
	return "ParentElevation"
}

// Execute 对融合后的文档执行父块提升，结果写入 state.ParentSearchDocs
func (s *ParentElevationStage) Execute(ctx context.Context, state *RetrievalState) error {
	if len(state.FusedDocs) == 0 {
		return nil
	}
	docs, err := s.elevateToParentChunks(ctx, state.FusedDocs, s.maxChars)
	if err != nil {
		return err
	}
	state.ParentSearchDocs = docs
	return nil
}

// elevateToParentChunks 将子文档提升到父块级别
func (s *ParentElevationStage) elevateToParentChunks(ctx context.Context, childDocuments []*vo.DocumentChunk, maxChars int) ([]*vo.DocumentChunk, error) {
	if len(childDocuments) == 0 {
		return nil, nil
	}

	childGroupsByParent := make(map[int64][]*vo.DocumentChunk, len(childDocuments))
	fallbackDocuments := make([]*vo.DocumentChunk, 0, len(childDocuments))
	parentChunkIds := make([]int64, 0, len(childDocuments))
	for _, childDocument := range childDocuments {
		parentChunkId := childDocument.ParentChunkId
		if parentChunkId == 0 {
			fallbackDocuments = append(fallbackDocuments, childDocument)
			continue
		}
		childGroupsByParent[parentChunkId] = append(childGroupsByParent[parentChunkId], childDocument)
		if _, exists := childGroupsByParent[parentChunkId]; exists {
			parentChunkIds = append(parentChunkIds, parentChunkId)
		}
	}

	if len(childGroupsByParent) == 0 {
		return fallbackDocuments, nil
	}

	parentChunks, err := s.docGateway.FindParentChunks(ctx, parentChunkIds)
	if err != nil {
		return nil, err
	}
	parentChunkMap := make(map[string]*vo.DocumentChunk, len(parentChunks))
	for _, item := range parentChunks {
		parentChunkMap[item.ID] = item
	}

	elevatedDocuments := make([]*vo.DocumentChunk, 0, len(childGroupsByParent)+len(fallbackDocuments))
	for parentId, children := range childGroupsByParent {
		parentChunk, ok := parentChunkMap[strconv.FormatInt(parentId, 10)]
		if !ok {
			elevatedDocuments = append(elevatedDocuments, children...)
			continue
		}
		elevatedDocuments = append(elevatedDocuments, s.buildParentEvidenceDocument(parentChunk, children, maxChars))
	}
	elevatedDocuments = append(elevatedDocuments, fallbackDocuments...)

	slices.SortFunc(elevatedDocuments, func(a, b *vo.DocumentChunk) int {
		if a.Score != b.Score {
			return cmp.Compare(b.Score, a.Score)
		} else if a.ParentChunkNo != b.ParentChunkNo {
			return a.ParentChunkNo - b.ParentChunkNo
		}
		return a.ChunkNo - b.ChunkNo
	})

	return elevatedDocuments, nil
}

// buildParentEvidenceDocument 构建父级证据文档
func (s *ParentElevationStage) buildParentEvidenceDocument(parentChunk *vo.DocumentChunk, childDocuments []*vo.DocumentChunk, maxChars int) *vo.DocumentChunk {
	if parentChunk == nil || len(childDocuments) == 0 {
		return nil
	}

	bestChild := s.selectBestChildForParentEvidence(childDocuments)
	channels := utils.FilterMapUniqueLimit(childDocuments, -1, func(item *vo.DocumentChunk) (string, string, bool) {
		return item.Channel, item.Channel, item.Channel != ""
	})

	maxScore := slices.MaxFunc(childDocuments, func(a, b *vo.DocumentChunk) int { return cmp.Compare(b.Score, a.Score) }).Score
	supportCount := max(0, len(childDocuments)-1)
	supportWeight := min(0.36, float64(supportCount)*0.12)
	multiChannelWeight := utils.Ternary(len(channels) > 1, 0.10, 0.0)
	parentScore := maxScore * (1.0 + supportWeight + multiChannelWeight)

	return &vo.DocumentChunk{
		ID:                fmt.Sprintf("parent-%s", parentChunk.ID),
		Score:             parentScore,
		Content:           s.renderParentEvidenceText(parentChunk, childDocuments, maxChars),
		SourceType:        parentChunk.SourceType,
		Channel:           utils.Ternary(len(channels) > 1, "hybrid", channels[0]),
		TaskId:            parentChunk.TaskId,
		ParentChunkId:     utils.StringToInt64(parentChunk.ID),
		DocumentId:        parentChunk.DocumentId,
		ChunkNo:           parentChunk.ChunkNo,
		SectionPath:       parentChunk.SectionPath,
		StructureNodeId:   parentChunk.StructureNodeId,
		StructureNodeType: parentChunk.StructureNodeType,
		CanonicalPath:     parentChunk.CanonicalPath,
		ItemIndex:         parentChunk.ItemIndex,
		OriginalSnippet:   parentChunk.Content,
		IsElevated:        1,
		ParentChunkNo:     parentChunk.ChunkNo,
		Extra:             bestChild.Extra,
	}
}

// todo 待优化完善，目前先简单选择score最高的子块作为代表
func (s *ParentElevationStage) selectBestChildForParentEvidence(childDocuments []*vo.DocumentChunk) *vo.DocumentChunk {
	return slices.MaxFunc(childDocuments, func(a, b *vo.DocumentChunk) int { return cmp.Compare(b.Score, a.Score) })
}

// renderParentEvidenceText 渲染父级证据文本：[父块内容] + [命中子片段]
func (s *ParentElevationStage) renderParentEvidenceText(chunk *vo.DocumentChunk, childDocuments []*vo.DocumentChunk, maxChars int) string {
	parentText := utils.Trim(chunk.Content)

	if parentText == "" {
		if len(childDocuments) == 0 {
			return ""
		}
		return childDocuments[0].OriginalSnippet
	}

	var childSummaryBuilder strings.Builder
	for i, childDocument := range childDocuments {
		if i > 0 {
			childSummaryBuilder.WriteByte('\n')
		}
		childSummaryBuilder.WriteString("- child#")
		childSummaryBuilder.WriteString(strconv.Itoa(childDocument.ChunkNo))
		childSummaryBuilder.WriteString("：")
		childSummaryBuilder.WriteString(utils.ClipHead(childDocument.OriginalSnippet, 140))
	}

	var composed string
	if childSummaryBuilder.Len() > 0 {
		composed = fmt.Sprintf("[父块内容]\n%s\n\n[命中子片段]\n%s", parentText, childSummaryBuilder.String())
	} else {
		composed = fmt.Sprintf("[父块内容]\n%s", parentText)
	}

	return utils.ClipHead(composed, max(maxChars, 1))
}
