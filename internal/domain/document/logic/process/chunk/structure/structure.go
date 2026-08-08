package structure

import (
	"context"

	"github.com/duke-git/lancet/v2/strutil"

	"github.com/swiftbit/know-agent/internal/domain/document/logic/process/chunk"
	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/document/model/enum"
	"github.com/swiftbit/know-agent/internal/domain/document/model/vo"
	"github.com/swiftbit/know-agent/internal/domain/document/support"
)

const (
	Name = "STRUCTURE" // 名称
)

// Chunker 基于文档标题结构的分块器
type Chunker struct {
	classifier *support.DocumentLineClassifier
}

// NewChunker 创建结构分块器
func NewChunker(opts ...chunk.Option) *Chunker {
	return &Chunker{
		classifier: &support.DocumentLineClassifier{},
	}
}

// Name 返回策略名称
func (c *Chunker) Name() string {
	return Name
}

// Chunk 按标题结构切分文本
func (c *Chunker) Chunk(_ context.Context, input *chunk.Input, opts ...chunk.Option) ([]*vo.ChunkCandidate, error) {
	if input == nil || len(input.Blocks) == 0 || len(input.Nodes) == 0 {
		return nil, nil
	}

	seeds := make(vo.ChunkCandidates, 0)
	currentGroup := make(entity.DocumentBlocks, 0)
	currentSectionKey := ""
	for _, block := range input.Blocks {
		sectionKey := strutil.Trim(block.SectionPath)
		sectionChanged := len(currentGroup) > 0 && currentSectionKey != sectionKey
		startsNewTitleSection := block.IsTitleBlock() && sectionChanged

		if sectionChanged || startsNewTitleSection {
			c.appendParentSeedsFromBlockGroup(currentGroup, nodes, options)
			currentGroup = make([]SuperAgentDocumentBlock, 0)
		}

		if len(currentGroup) == 0 {
			currentSectionKey = sectionKey
		}
		currentGroup = append(currentGroup, block)
	}

	// 产出章节种子（仅保留"有实质内容"的章节）
	seeds := make(vo.ChunkCandidates, 0, len(nodes))
	for _, node := range nodes {
		if node.NodeType == vo.NodeTypeSection && p.isContentBearingSection(node, parentHasChildSection[node.ID]) {
			seeds = append(seeds, p.newChunkCandidate(node, enum.ChunkSourceTypeOriginal))
		}
	}

	return result, nil
}

func (c *Chunker) appendParentSeedsFromBlockGroup(group entity.DocumentBlocks, nodes []*entity.StructureNode, options *vo.IndexingOptions) {
	if len(group) == 0 {
		return
	}
	results := make(vo.ChunkCandidates, 0, len(group))

	text := group.JoinBlockTexts()
	parentMaxChars := options.ResolveRecursiveMaxChars(enum.PipelineTypeParent)

	if len([]rune(text)) <= parentMaxChars {
		*seeds = append(*seeds, p.toParentSeed(group, structureNodes))
		return
	}

	*seeds = append(*seeds, p.buildBlockWindowParentSeeds(group, structureNodes, parentMaxChars)...)
}

// composeSectionPath 拼接基础路径与当前层级路径，用 " > " 分隔
func (c *Chunker) composeSectionPath(base, current string) string {
	baseTrimmed := strutil.Trim(base)
	currentTrimmed := strutil.Trim(current)
	if baseTrimmed == "" {
		return currentTrimmed
	}
	if currentTrimmed == "" {
		return baseTrimmed
	}
	return baseTrimmed + " > " + currentTrimmed
}
