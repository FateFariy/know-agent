package index

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/duke-git/lancet/v2/maputil"
	"github.com/duke-git/lancet/v2/slice"
	"github.com/duke-git/lancet/v2/strutil"

	"github.com/swiftbit/know-agent/common/logx"
	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/document/adapter"
	"github.com/swiftbit/know-agent/internal/domain/document/logic/process/chunk"
	chunkrecursive "github.com/swiftbit/know-agent/internal/domain/document/logic/process/chunk/recursive"
	chunksemantic "github.com/swiftbit/know-agent/internal/domain/document/logic/process/chunk/semantic"
	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/document/model/enum"
	"github.com/swiftbit/know-agent/internal/domain/document/model/vo"
	errorx "github.com/swiftbit/know-agent/internal/error"
)

// 父块/子块切块常量
const (
	ParentBlockMaxChars     = 2200 // 父块最大字符数
	ParentBlockOverlapChars = 180  // 父块重叠字符数
	ParentSemanticMaxChars  = 1600 // 语义块最大字符数
	ParentSemanticMinChars  = 480  // 语义块最小字符数
)

// ChunkingPhase 切块阶段：执行切块流水线、构建父子块实体、持久化
type ChunkingPhase struct {
	repo      adapter.DocumentRepository
	port      *adapter.DocumentPort
	registry  map[int]chunk.Chunker
	resolver  IndexingConfigResolver
	tokenizer adapter.Tokenizer
}

// NewChunkingPhase 创建切块阶段
func NewChunkingPhase(repo adapter.DocumentRepository, port *adapter.DocumentPort,
	registry map[int]chunk.Chunker, resolver IndexingConfigResolver, tokenizer adapter.Tokenizer) *ChunkingPhase {
	return &ChunkingPhase{
		repo:      repo,
		port:      port,
		registry:  registry,
		resolver:  resolver,
		tokenizer: tokenizer,
	}
}

func (p *ChunkingPhase) Name() string {
	return "切块阶段"
}

func (p *ChunkingPhase) Execute(ctx context.Context, buildCtx *Context) error {
	// 检查是否需要从已提交 GraphRAG 结果恢复
	if buildCtx.ResumeCommittedGraph {
		return nil
	}
	chunkStartDetail, _ := json.Marshal(map[string]any{"strategySnapshot": buildCtx.Plan.StrategySnapshot})
	chunkStartLog := &entity.DocumentTaskLog{
		TaskId:       buildCtx.TaskId,
		DocumentId:   buildCtx.DocumentId,
		StageType:    enum.TaskStageChunkExecute,
		EventType:    enum.TaskEventStart,
		LogLevel:     enum.LogLevelInfo,
		OperatorType: enum.OperatorTypeSystem,
		Content:      "开始执行切块流水线",
		DetailJson:   string(chunkStartDetail),
	}
	_ = p.repo.InsertTaskLog(ctx, chunkStartLog)

	if err := p.repo.UpdateStepExecuteStatus(ctx, buildCtx.PlanId, enum.StrategyExecuteStatusExecuting); err != nil {
		return err
	}
	blocks, err := p.repo.SelectDocumentBlocksByTask(ctx, buildCtx.DocumentId, buildCtx.TaskId)
	if err != nil {
		return err
	}
	if len(blocks) == 0 {
		return errors.New("当前文档没有结构化解析 blocks，无法执行 Parent/Child 切块。")
	}

	// 查询策略步骤列表
	pipelineSteps, err := p.repo.SelectStepListByPlanId(ctx, buildCtx.PlanId)
	if err != nil {
		return err
	}
	// 按步骤执行切块流水线
	chunkStartedNanos := time.Now()
	parentCandidates, err := p.BuildParentBlocks(ctx, buildCtx.Document, pipelineSteps, blocks)
	if err != nil {
		return err
	}
	buildCtx.ParentCandidates = parentCandidates
	costMillis := time.Since(chunkStartedNanos).Milliseconds()
	logx.Infof("切块流水线执行完成，documentId=%d, taskId=%d, parentCount=%d, childCount=%d, costMillis=%d",
		buildCtx.DocumentId, buildCtx.TaskId, len(parentCandidates), p.countChildCandidates(parentCandidates), costMillis)

	return nil
}

// countChildCandidates 计算子块候选数
func (p *ChunkingPhase) countChildCandidates(parentBlockCandidateList []*vo.ParentBlockCandidate) int {
	count := 0
	for _, candidate := range parentBlockCandidateList {
		for _, child := range candidate.ChildChunks {
			if child != nil && strutil.IsNotBlank(child.Text) {
				count++
			}
		}
	}
	return count
}

// cleanupParentCandidates 过滤"文本为空"或"无子块"的父块候选
func (p *ChunkingPhase) cleanupParentCandidates(candidates []*vo.ParentBlockCandidate) []*vo.ParentBlockCandidate {
	return slice.Filter(candidates, func(_ int, item *vo.ParentBlockCandidate) bool {
		fn := func(child *vo.ChunkCandidate) bool { return child != nil && strutil.IsNotBlank(child.Text) }
		return item != nil && strutil.IsNotBlank(item.Text) && slices.ContainsFunc(item.ChildChunks, fn)
	})
}

// buildParentChildEntities 将父块候选转换为可落库的"父块实体 + 子块实体"双列表
func (p *ChunkingPhase) buildParentChildEntities(documentId, taskId, planId int64,
	candidates []*vo.ParentBlockCandidate) ([]*entity.DocumentParentBlock, []*entity.DocumentChunk) {

	parentBlocks := make([]*entity.DocumentParentBlock, 0, len(candidates))
	chunks := make([]*entity.DocumentChunk, 0)

	globalChunkNo := 0
	for parentIdx, candidate := range candidates {
		parentBlock := &entity.DocumentParentBlock{
			ID:                utils.GetSnowflakeNextID(),
			DocumentId:        documentId,
			TaskId:            taskId,
			PlanId:            planId,
			ParentNo:          parentIdx + 1,
			SourceType:        candidate.SourceType,
			SectionPath:       candidate.SectionPath,
			StructureNodeId:   candidate.StructureNodeId,
			StructureNodeType: candidate.StructureNodeType,
			CanonicalPath:     candidate.CanonicalPath,
			ItemIndex:         candidate.ItemIndex,
			ParentText:        candidate.Text,
			CharCount:         utils.Len(candidate.Text),
			TokenCount:        utils.EstimateTokens(candidate.Text),
			StartChunkNo:      globalChunkNo,
		}

		for _, child := range candidate.ChildChunks {
			if child != nil && strutil.IsNotBlank(child.Text) {
				globalChunkNo++
				chunks = append(chunks, &entity.DocumentChunk{
					ID:                utils.GetSnowflakeNextID(),
					DocumentId:        documentId,
					TaskId:            taskId,
					PlanId:            planId,
					ParentBlockId:     parentBlock.ID,
					ChunkNo:           globalChunkNo,
					SourceType:        child.SourceType,
					SectionPath:       utils.BlankToDefault(child.SectionPath, candidate.SectionPath),
					StructureNodeId:   child.StructureNodeId,
					StructureNodeType: child.StructureNodeType,
					CanonicalPath:     child.CanonicalPath,
					ItemIndex:         child.ItemIndex,
					ChunkText:         child.Text,
					CharCount:         utils.Len(child.Text),
					TokenCount:        utils.EstimateTokens(child.Text),
					VectorStatus:      enum.VectorStatusWaitVector,
				})
				parentBlock.ChildCount++
			}
		}
		parentBlock.EndChunkNo = globalChunkNo - 1
		parentBlocks = append(parentBlocks, parentBlock)
	}
	return parentBlocks, chunks
}

// BuildParentBlocks 执行完整的父-子块构建流程：先通过父块流水线生成父种子，再针对每个父种子走子块流水线产出子块
func (p *ChunkingPhase) BuildParentBlocks(ctx context.Context, document *entity.Document,
	steps entity.DocumentStrategySteps, blocks entity.DocumentBlocks) ([]*vo.ParentBlockCandidate, error) {
	// 按父/子流水线拆分并排序步骤；任一缺失则返回相应错误
	parentSteps := steps.SortPipelineSteps(enum.PipelineTypeParent)
	childSteps := steps.SortPipelineSteps(enum.PipelineTypeChild)
	if len(parentSteps) == 0 {
		return nil, errorx.ErrParentBlockMissing
	}
	if len(childSteps) == 0 {
		return nil, errorx.ErrChildBlockMissing
	}

	orderedBlocks := blocks.CleanupAndSort()
	if len(orderedBlocks) == 0 {
		return nil, errorx.ErrDocumentBlocksMissing
	}
	blockMap := utils.SliceToMapBy(orderedBlocks, func(block *entity.DocumentBlock) (int64, *entity.DocumentBlock) {
		return block.ID, block
	})

	// 加载已解析的文档结构节点（用于结构切块策略）
	var nodes []*entity.StructureNode
	var err error
	if document != nil {
		nodes, err = p.repo.SelectStructureNodeListByTask(ctx, document.ID, document.LastParseTaskId)
		if err != nil {
			return nil, err
		}
	}

	// 从文档存储中读取解析后的全文（用于兜底：结构节点不可用时直接以全文走流水线）
	parsedText := ""
	if document != nil && strutil.IsNotBlank(document.ParseTextPath) {
		if text, err := p.port.DownloadText(ctx, document.ParseTextPath); err == nil {
			parsedText = text
		}
	}
	options := p.resolver.Resolve(ctx, document)

	// 生成父块种子列表
	parentSeedList := p.buildParentSeedList(ctx, blocks, parentSteps, nodes, options)

	// 为每个父块种子派生其子块；无子块时以父块本身兜底
	parentBlockList := make([]*vo.ParentBlockCandidate, 0)
	for _, parentSeed := range parentSeedList.CleanupAndUnique() {
		if parentSeed != nil && strutil.IsNotBlank(parentSeed.Text) {
			childSeedList := p.buildChildSeedList(ctx, parentSeed, childSteps, nodes)
			finalChildren := p.cleanupChunkList(childSeedList)

			trim := strutil.Trim(parentSeed.Text)
			if len(finalChildren) == 0 {
				// 兜底策略：子块流水线无产出 → 使用父块本身作为唯一子块
				finalChildren = vo.ChunkCandidates{
					vo.CopyChunkCandidate(parentSeed, trim),
				}
			}

			parentBlock := &vo.ParentBlockCandidate{
				SectionPath:       parentSeed.SectionPath,
				StructureNodeId:   parentSeed.StructureNodeId,
				StructureNodeType: parentSeed.StructureNodeType,
				Text:              trim,
				SourceType:        parentSeed.SourceType,
				ChildChunks:       finalChildren,
				CanonicalPath:     parentSeed.CanonicalPath,
				ItemIndex:         parentSeed.ItemIndex,
			}

			parentBlockList = append(parentBlockList, parentBlock)
		}
	}

	// 对父块进行去重与清理后返回
	return p.cleanupParentBlockList(parentBlockList), nil
}

// ---------------- 草稿/标准化 ----------------

// buildDraftSteps 将策略类型列表构造成推荐步骤草稿（带上角色与理由），首项默认为主策略，其余按类型赋予优化/兜底/增强角色
func (p *ChunkingPhase) buildDraftSteps(pipelineType string, strategyTypes []int, reasonMap map[int]string) []*vo.DocumentStrategyStepDraft {
	return slice.Map(strategyTypes, func(index, strategyType int) *vo.DocumentStrategyStepDraft {
		return &vo.DocumentStrategyStepDraft{
			PipelineType:    pipelineType,
			StrategyType:    strategyType,
			StrategyRole:    p.resolveRole(index, strategyType),
			SourceType:      enum.StrategySourceTypeSystemRecommend,
			RecommendReason: utils.BlankToDefault(reasonMap[strategyType], "系统为当前流水线生成的推荐步骤。"),
		}
	})
}

// sortPipelineSteps

// resolveRole 为指定步骤分配角色
func (p *ChunkingPhase) resolveRole(index int, strategyType int) int {
	if index == 0 {
		return enum.StrategyRolePrimary
	}
	if strategyType == enum.StrategyTypeRecursive {
		return enum.StrategyRoleFallback
	}
	if strategyType == enum.StrategyTypeSemantic {
		return enum.StrategyRoleOptimize
	}
	if strategyType == enum.StrategyTypeLLM {
		return enum.StrategyRoleEnhance
	}
	return enum.StrategyRoleOptimize
}

// ---------------- 种子构建 ----------------

// buildParentSeedList 构建父块种子列表，若步骤中含有结构切块且结构节点存在，优先走结构路径；否则从原始文本构造单一父种子
func (p *ChunkingPhase) buildParentSeedList(ctx context.Context, blocks entity.DocumentBlocks, parentSteps entity.DocumentStrategySteps,
	nodes []*entity.StructureNode, options *vo.IndexingOptions) vo.ChunkCandidates {
	if parentSteps.Contains(enum.StrategyTypeStructure) {
		// 结构切块有节点可用 → 先产出章节级种子，再将剩余策略作为后续流水线
		structureSeeds := p.buildBlockSectionParentSeeds(blocks, nodes, options)
		remainingSteps := parentSteps.DeleteStep(enum.StrategyTypeStructure)

		if len(remainingSteps) == 0 {
			return structureSeeds
		}

		return p.executePipeline(ctx, structureSeeds, remainingSteps, enum.PipelineTypeParent, options)

	}

	maxChars := options.ResolveRecursiveMaxChars(enum.PipelineTypeParent)
	parentSeeds := p.buildBlockWindowParentSeeds(blocks, nodes, maxChars)
	remainingSteps := parentSteps.DeleteStep(enum.StrategyTypeRecursive)

	if len(remainingSteps) == 0 {
		return parentSeeds
	}

	// 执行父块流水线
	return p.executePipeline(ctx, parentSeeds, parentSteps, enum.PipelineTypeParent, options)
}

// buildChildSeedList 为指定父种子构建子块种子列表，若步骤中含有结构切块且结构节点存在，优先按结构节点拆解子章节，否则克隆父种子再跑流水线
func (p *ChunkingPhase) buildChildSeedList(ctx context.Context, parentSeed *vo.ChunkCandidate,
	childSteps entity.DocumentStrategySteps, structureNodes []*entity.StructureNode) vo.ChunkCandidates {
	if childSteps.Contains(enum.StrategyTypeStructure) && parentSeed.StructureNodeId != 0 && len(structureNodes) > 0 {
		// 基于父种子的节点 ID 收集子节点，再进入后续流水线
		structureSeeds := p.buildStructureChildSeeds(parentSeed, structureNodes)

		remainingSteps := childSteps.DeleteStep(enum.StrategyTypeStructure)
		if len(remainingSteps) == 0 {
			return structureSeeds
		}

		return p.executePipeline(ctx, structureSeeds, remainingSteps, enum.PipelineTypeChild, nil)
	}

	// 直接克隆父种子作为子块流水线的起点
	clonedSeed := vo.CopyChunkCandidate(parentSeed, parentSeed.Text)

	// 执行子块流水线
	return p.executePipeline(ctx, vo.ChunkCandidates{clonedSeed}, childSteps, enum.PipelineTypeChild, nil)
}

// buildBlockSectionParentSeeds 从结构节点中筛选"内容承载章节"生成父块种子，判定规则：含有子章节时需额外验证内容长度显著超过标题或出现换行
func (p *ChunkingPhase) buildBlockSectionParentSeeds(blocks entity.DocumentBlocks, nodes []*entity.StructureNode,
	options *vo.IndexingOptions) vo.ChunkCandidates {
	seeds := make(vo.ChunkCandidates, 0)
	currentGroup := make(entity.DocumentBlocks, 0)
	currentSectionKey := ""
	for _, block := range blocks {
		sectionKey := strutil.Trim(block.SectionPath)
		sectionChanged := len(currentGroup) > 0 && currentSectionKey != sectionKey
		startsNewTitleSection := block.IsTitleBlock() && sectionChanged

		if sectionChanged || startsNewTitleSection {
			p.appendParentSeedsFromBlockGroup(currentGroup, nodes, options)
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

	return seeds
}

func (p *ChunkingPhase) buildBlockSectionParentSeeds(
	ctx context.Context,
	documentBlocks []SuperAgentDocumentBlock,
	structureNodes []SuperAgentDocumentStructureNode,
	indexingOptions KnowledgeBaseIndexingOptions,
) []ChunkCandidate {

	seeds := make([]ChunkCandidate, 0)
	currentGroup := make([]SuperAgentDocumentBlock, 0)
	currentSectionKey := ""

	for _, block := range documentBlocks {
		sectionKey := p.sectionKey(ctx, block)
		sectionChanged := len(currentGroup) > 0 && currentSectionKey != sectionKey
		startsNewTitleSection := p.isTitleBlock(ctx, block) && sectionChanged

		if sectionChanged || startsNewTitleSection {
			p.appendParentSeedsFromBlockGroup(ctx, seeds, currentGroup, structureNodes, indexingOptions)
			currentGroup = make([]SuperAgentDocumentBlock, 0)
		}

		if len(currentGroup) == 0 {
			currentSectionKey = sectionKey
		}
		currentGroup = append(currentGroup, block)
	}

	p.appendParentSeedsFromBlockGroup(ctx, seeds, currentGroup, structureNodes, indexingOptions)

	if len(seeds) == 0 {
		return p.buildBlockWindowParentSeeds(ctx, p.resolveRecursiveMaxChars(ctx, DocumentStrategyPipelineTypeEnum_PARENT, indexingOptions), documentBlocks)
	}
	return seeds
}

// buildStructureChildSeeds 根据父种子的节点 ID 从结构节点中挑出其子节点作为子块种子。
// 仅保留 SECTION / STEP / LIST_ITEM 三类有实际内容的子节点；否则回退到克隆父种子。
func (p *ChunkingPhase) buildStructureChildSeeds(parentSeed *vo.ChunkCandidate, structureNodes []*entity.StructureNode) vo.ChunkCandidates {
	// 按 ParentNodeId 索引结构节点，快速定位当前父种子的子节点集合
	childrenByParent := make(map[int64][]*entity.StructureNode)
	for _, node := range structureNodes {
		if node.ParentNodeId != 0 {
			childrenByParent[node.ParentNodeId] = append(childrenByParent[node.ParentNodeId], node)
		}
	}

	seeds := make(vo.ChunkCandidates, 0)
	children := childrenByParent[parentSeed.StructureNodeId]

	for _, child := range children {
		if strutil.IsNotBlank(child.ContentText) {
			// 仅保留结构化语义节点类型
			if child.NodeType == vo.NodeTypeSection || child.NodeType == vo.NodeTypeStep || child.NodeType == vo.NodeTypeListItem {
				seeds = append(seeds, p.newChunkCandidate(child, enum.ChunkSourceTypeOriginal))
			}
		}
	}

	if len(seeds) > 0 {
		return seeds
	}

	// 回退：无合适子节点时将父种子本身克隆为唯一子块
	return vo.ChunkCandidates{vo.CopyChunkCandidate(parentSeed, parentSeed.Text)}
}

// isContentBearingSection 判断该章节是否为"内容承载章节"，排除仅作为容器而没有实际文本的章节（如纯嵌套目录）
func (p *ChunkingPhase) isContentBearingSection(node *entity.StructureNode, hasChildSection bool) bool {
	// 空内容直接排除
	if strutil.IsBlank(node.ContentText) {
		return false
	}

	// 无子章节 → 直接视作内容承载
	if !hasChildSection {
		return true
	}

	// 有子章节时：内容不能完全等同于标题
	headingText := strutil.Trim(utils.BlankToDefault(node.AnchorText, node.Title))
	content := strutil.Trim(node.ContentText)

	if content == headingText {
		return false
	}

	// 长度显著超过标题或包含换行 → 视为存在独立内容
	return utils.Len(content) > utils.Len(headingText)+16 || strings.Contains(content, "\n")
}

// cloneParentBlockCandidate 克隆 ParentBlockCandidate
func (p *ChunkingPhase) cloneParentBlockCandidate(source *vo.ParentBlockCandidate, childChunks vo.ChunkCandidates, text string) *vo.ParentBlockCandidate {
	if source == nil {
		return &vo.ParentBlockCandidate{
			Text:        text,
			SourceType:  enum.ChunkSourceTypeOriginal,
			ChildChunks: append(vo.ChunkCandidates{}, childChunks...),
		}
	}
	return &vo.ParentBlockCandidate{
		SectionPath:       source.SectionPath,
		StructureNodeId:   source.StructureNodeId,
		StructureNodeType: source.StructureNodeType,
		CanonicalPath:     source.CanonicalPath,
		ItemIndex:         source.ItemIndex,
		Text:              text,
		SourceType:        source.SourceType,
		ChildChunks:       append(vo.ChunkCandidates{}, childChunks...),
	}
}

// ---------------- 流水线调度 ----------------

// executePipeline 按步骤顺序调度分块策略，当前步骤的输出作为下一步骤的输入
func (p *ChunkingPhase) executePipeline(ctx context.Context, seeds vo.ChunkCandidates, steps entity.DocumentStrategySteps, pipelineType string, options *vo.IndexingOptions) vo.ChunkCandidates {
	// 去除空文本和重复项
	currentChunks := seeds.CleanupAndUnique()
	if len(currentChunks) == 0 {
		return currentChunks
	}

	for _, step := range steps {
		strategy, ok := p.registry[step.StrategyType]
		if !ok {
			continue
		}

		// 根据策略类型与流水线类型生成额外选项（父块流水线会使用较大窗口）
		extraOpts := p.buildPipelineOptions(step.StrategyType, pipelineType)

		nextChunks := make(vo.ChunkCandidates, 0, len(currentChunks))
		for _, candidate := range currentChunks {
			if candidate == nil || strutil.IsBlank(candidate.Text) {
				continue
			}
			input := &chunk.TextBlock{
				SectionPath:   candidate.SectionPath,
				CanonicalPath: candidate.CanonicalPath,
				ItemIndex:     candidate.ItemIndex,
				Text:          candidate.Text,
				SourceType:    candidate.SourceType,
			}

			var outputs []*chunk.TextBlock
			if step.StrategyType == enum.StrategyTypeLLM {
				// 大模型切块走专用调用（含递归拆分与回退语义）
				outputs = p.applyLlmChunking(ctx, input, pipelineType, extraOpts...)
			} else {
				outputs, _ = strategy.Chunk(ctx, input, extraOpts...)
			}
			// 结构切块无产出时，使用递归策略兜底
			if len(outputs) == 0 && step.StrategyType == enum.StrategyTypeStructure {
				outputs, _ = p.registry[enum.StrategyTypeRecursive].Chunk(ctx, input, extraOpts...)
			}
			for _, out := range outputs {
				if strutil.IsNotBlank(out.Text) {
					nextChunks = append(nextChunks, vo.CopyChunkCandidate(candidate, out.Text))
				}
			}
		}
		// 每步结束后清理，避免中间产物污染下游
		currentChunks = p.cleanupChunkList(nextChunks)
	}
	return p.cleanupChunkList(currentChunks)
}

func (p *ChunkingPhase) executePipeline(
	sourceList []*ChunkCandidate,
	orderedSteps []*SuperAgentDocumentStrategyStep,
	pipelineType DocumentStrategyPipelineTypeEnum,
	indexingOptions *KnowledgeBaseIndexingOptions,
) []*ChunkCandidate {
	// 初始清理：过滤无效候选块
	currentChunks := p.cleanupChunkList(sourceList)

	// 按顺序执行每个策略步骤
	for _, step := range orderedSteps {
		strategyType := DocumentStrategyTypeEnumGetRc(step.GetStrategyType())
		if strategyType == nil {
			continue
		}

		// 根据策略类型分发处理
		switch *strategyType {
		case STRUCTURE:
			// 结构策略：保持原样，不做分块处理
			// 结构策略通常用于保留文档的原始层级关系
			currentChunks = currentChunks
		case RECURSIVE:
			// 递归策略：基于字符/词边界进行递归拆分
			currentChunks = p.applyRecursiveChunking(currentChunks, pipelineType, indexingOptions)
		case SEMANTIC:
			// 语义策略：基于 embedding 相似度进行语义边界分割
			currentChunks = p.applySemanticChunking(currentChunks, pipelineType, indexingOptions)
		case LLM:
			// LLM策略：使用大语言模型进行智能分块
			currentChunks = p.applyLlmChunking(currentChunks, pipelineType, indexingOptions)
		default:
			// 未知策略：保持原样
			currentChunks = currentChunks
		}

		// 每步处理后清理，移除空块或非法块
		currentChunks = p.cleanupChunkList(currentChunks)
	}

	// 最终清理并返回结果
	return p.cleanupChunkList(currentChunks)
}

// buildPipelineOptions 根据流水线类型和策略类型生成额外的策略选项
func (p *ChunkingPhase) buildPipelineOptions(strategyType int, pipelineType string) []chunk.Option {
	if pipelineType != enum.PipelineTypeParent {
		return nil
	}
	switch strategyType {
	case enum.StrategyTypeRecursive:
		// 递归：使用更大的 maxChars 和较小的重叠（同时确保 overlap < maxChars）
		maxChars := ParentBlockMaxChars
		overlap := min(ParentBlockOverlapChars, max(0, maxChars-1))
		return []chunk.Option{
			chunkrecursive.WithMaxChars(maxChars),
			chunkrecursive.WithOverlapChars(overlap),
		}
	case enum.StrategyTypeSemantic:
		// 语义：与配置/父块语义阈值取较大值，确保不被过度切分
		maxChars := max(p.option.semanticMaxChars, ParentSemanticMaxChars)
		minChars := max(p.option.semanticMinChars, ParentSemanticMinChars)
		return []chunk.Option{
			chunksemantic.WithMaxChars(maxChars),
			chunksemantic.WithMinChars(minChars),
		}
	default:
		return nil
	}
}

// cleanupParentBlockList 清理父块列表：规则与子块一致，path+itemIndex+trim 去重
func (p *ChunkingPhase) cleanupParentBlockList(blocks []*vo.ParentBlockCandidate) []*vo.ParentBlockCandidate {
	result := make(map[string]*vo.ParentBlockCandidate)
	for _, block := range blocks {
		if block != nil && strutil.IsNotBlank(block.Text) {
			path := utils.BlankToDefault(block.CanonicalPath, block.SectionPath)
			trim := strutil.Trim(block.Text)
			uniqueKey := fmt.Sprintf("%s||%d||%s", path, block.ItemIndex, trim)
			if _, ok := result[uniqueKey]; !ok {
				result[uniqueKey] = p.cloneParentBlockCandidate(block, block.ChildChunks, trim)
			}
		}
	}
	return maputil.Values(result)
}

// applyLlmChunking 大模型智能切块
//
// 策略：
//  1. LLM 未启用 → 回退到语义切块
//  2. 输入超长 → 先用递归切块切到 llmMaxChars 以下
//  3. 逐项调用 LLM；失败或无产出 → 回退到语义切块补全
func (p *ChunkingPhase) applyLlmChunking(ctx context.Context, input *chunk.TextBlock, pipeType string, extraOpts ...chunk.Option) []*chunk.TextBlock {
	var outputs []*chunk.TextBlock
	var err error
	// LLM 未启用 → 直接使用语义切块
	if p.option == nil || !p.option.llmEnabled {
		outputs, _ = p.registry[enum.StrategyTypeSemantic].Chunk(ctx, input, extraOpts...)
		return outputs
	}
	// 输入过长 → 先以递归切块拆分到 LLM 上限
	if utils.Len(input.Text) > p.option.llmMaxChars {
		llmMaxChars := utils.Ternary(pipeType == enum.PipelineTypeParent, max(p.option.llmMaxChars, ParentBlockMaxChars), p.option.llmMaxChars)
		outputs, _ = p.registry[enum.StrategyTypeRecursive].Chunk(ctx, input, chunkrecursive.WithOverlapChars(0), chunkrecursive.WithMaxChars(llmMaxChars))
	}

	// 逐项调用 LLM 切块；失败/空产出回退到语义切块
	resultList := make([]*chunk.TextBlock, 0, len(outputs))
	for _, item := range outputs {
		if strutil.IsNotBlank(item.Text) {
			outputs, err = p.registry[enum.StrategyTypeLLM].Chunk(ctx, item)
			if err != nil {
				logx.Warnf("大模型智能切块失败，回退到语义切块，err=%v", err)
			}
			if len(outputs) == 0 {
				outputs, _ = p.registry[enum.StrategyTypeSemantic].Chunk(ctx, item, extraOpts...)
			}
			resultList = append(resultList, outputs...)
		}
	}
	return resultList
}

// newChunkCandidate 由结构节点构造新的块候选（保留章节/路径/序号等元信息）
func (p *ChunkingPhase) newChunkCandidate(node *entity.StructureNode, sourceType int) *vo.ChunkCandidate {
	return &vo.ChunkCandidate{
		SectionPath:       node.SectionPath,
		StructureNodeId:   node.ID,
		StructureNodeType: node.NodeType,
		CanonicalPath:     node.CanonicalPath,
		ItemIndex:         node.ItemIndex,
		Text:              node.ContentText,
		SourceType:        sourceType,
	}
}

func (p *ChunkingPhase) buildBlockWindowParentSeeds(blocks entity.DocumentBlocks, nodes []*entity.StructureNode, maxChars int) vo.ChunkCandidates {
	seeds := make(vo.ChunkCandidates, 0)
	currentBlocks := make(entity.DocumentBlocks, 0)
	currentChars := 0
	for _, block := range blocks {
		blockText := block.RenderBlockContent()
		if strings.TrimSpace(blockText) == "" {
			continue
		}

		// 处理超长块：递归拆分
		if utils.Len(blockText) > maxChars {
			if len(currentBlocks) > 0 {
				seeds = append(seeds, p.toParentSeed(currentBlocks, nodes))
			}
			currentBlocks = make(entity.DocumentBlocks, 0)
			currentChars = 0

			// 递归拆分超长文本
			for _, splitText := range p.recursiveSplit(blockText, maxChars, 0) {
				seeds = append(seeds, p.toSplitBlockSeed(block, splitText, nodes))
			}
			continue
		}

		// 检查窗口是否已满（考虑块间分隔符长度）
		if len(currentBlocks) > 0 && currentChars+utf8.RuneCountInString(blockText)+2 > maxChars {
			seeds = append(seeds, p.toParentSeed(currentBlocks, nodes))
			currentBlocks = make(entity.DocumentBlocks, 0)
			currentChars = 0
		}

		currentBlocks = append(currentBlocks, block)
		currentChars += utils.Len(blockText) + 2
	}

	// 处理剩余块
	if len(currentBlocks) > 0 {
		seeds = append(seeds, p.toParentSeed(currentBlocks, nodes))
	}

	return seeds
}

var re = regexp.MustCompile(`[>/|]`)

func (p *ChunkingPhase) renderBlockWeightedContent(block *entity.DocumentBlock) string {
	if block == nil {
		return ""
	}
	space := strings.TrimSpace(block.ContentWithWeight)
	if space != "" {
		return space
	}
	text := block.RenderBlockContent()
	title := block.ResolveTitle(block.CanonicalPath)

}

func (p *ChunkingPhase) joinBlockWeightedContents(blocks entity.DocumentBlocks) string {
	var contents []string

	for _, block := range blocks {
		content := strings.TrimSpace(p.renderBlockWeightedContent(block))
		if content != "" {
			contents = append(contents, content)
		}
	}

	return strings.Join(contents, "\n\n")
}

func (p *ChunkingPhase) buildKeywords(title, sectionPath, text string) []string {
	keywords := make([]string, 0, 12)
	seen := make(map[string]bool, 12)
	add := func(words ...string) {
		for i := 0; i < len(words) && len(keywords) < 12; i++ {
			word := strutil.Trim(words[i])
			if utils.Len(word) >= 2 && !seen[word] {
				seen[word] = true
				keywords = append(keywords, word)
			}
		}
	}
	add(title)
	if strings.TrimSpace(sectionPath) != "" {
		add(re.Split(sectionPath, -1)...)
	}
	words := p.tokenizer.SegmentWords(text)
	add(words...)

	return keywords
}

// buildQuestions 构建面向检索的问答对, 基于标题、块类型和关键词生成假设性问题
func (p *ChunkingPhase) buildQuestions(title, chunkType string, keywords []string) string {
	seen := make(map[string]bool, 4)
	questions := make([]string, 0, 4)
	add := func(question string) {
		if !seen[question] && len(questions) < 4 {
			seen[question] = true
			questions = append(questions, question)
		}
	}

	// 确定主题词：优先使用标题，否则取关键词列表的第一个
	topic := strings.TrimSpace(title)
	if topic == "" {
		if len(keywords) > 0 {
			topic = keywords[0]
		}
	}

	// 基于主题生成通用问题
	if topic != "" {
		add("关于" + topic + "的核心内容是什么？")
		add(topic + "有哪些要求或注意事项？")
	}

	// 基于块类型生成特定问题
	upperType := strings.ToUpper(chunkType)
	if upperType == "TABLE" {
		add("这个表格说明了什么？")
	}
	if upperType == "IMAGE" || upperType == "FIGURE" {
		add("这张图片说明了什么？")
	}

	data, _ := json.Marshal(questions)

	return string(data)
}

// buildContentWithWeight 构建带权重的富文本内容, 将标题、章节路径、块类型、关键词、问题及正文按固定格式组装
func (p *ChunkingPhase) buildContentWithWeight(text, sectionPath, title, chunkType, keywords, questions, parserWeightedContent string) string {
	var parts []string

	// 标题部分
	if trimmed := strings.TrimSpace(title); trimmed != "" {
		parts = append(parts, "[TITLE]\n"+trimmed)
	}

	// 章节路径部分
	if trimmed := strings.TrimSpace(sectionPath); trimmed != "" {
		parts = append(parts, "[SECTION]\n"+trimmed)
	}

	// 块类型部分
	if trimmed := strings.TrimSpace(chunkType); trimmed != "" {
		parts = append(parts, "[CHUNK_TYPE]\n"+trimmed)
	}

	jsonArrayToDisplayText := func(jsonArray string) string {
		results := make([]string, 0)
		if err := json.Unmarshal([]byte(jsonArray), &results); err != nil {
			displayText := strings.Replace(jsonArray, "\"", "", -1)
			displayText = strings.TrimPrefix(displayText, "[")
			displayText = strings.TrimSuffix(displayText, "]")
			results = strings.Split(displayText, ",")
		}
		j := 0
		for i := range results {
			space := strings.TrimSpace(results[i])
			if space == "" {
				continue
			}
			results[j] = space
			j++
		}
		return strings.Join(results[:j], ";")
	}

	// 关键词部分（JSON数组转可读文本）
	if keywordText := jsonArrayToDisplayText(keywords); keywordText != "" {
		parts = append(parts, "[KEYWORDS]\n"+keywordText)
	}

	// 问题部分（JSON数组转可读文本）
	if questionText := jsonArrayToDisplayText(questions); questionText != "" {
		parts = append(parts, "[QUESTIONS]\n"+questionText)
	}

	// 正文内容部分（优先使用解析器生成的加权内容）
	weightedBody := strings.TrimSpace(parserWeightedContent)
	if weightedBody == "" {
		weightedBody = strings.TrimSpace(text)
	}
	if weightedBody != "" {
		parts = append(parts, "[CONTENT]\n"+weightedBody)
	}

	// 用双换行符连接所有部分并修剪首尾空白
	return strings.TrimSpace(strings.Join(parts, "\n\n"))
}

func (p *ChunkingPhase) toParentSeed(blocks entity.DocumentBlocks, nodes entity.StructureNodes) *vo.ChunkCandidate {
	sectionPath := blocks.CommonSectionPath()
	canonicalPath := utils.FirstNonBlank(blocks.CanonicalPaths()...)
	node := nodes.FindNodeByPath(sectionPath, canonicalPath)
	text := blocks.JoinBlockTexts()
	title := blocks.ResolveTitle(canonicalPath)
	chunkType := blocks.ResolveChunkType()
	keywords := p.buildKeywords(title, canonicalPath, text)
	questions := p.buildQuestions(title, chunkType, keywords)
	keywordJson, _ := json.Marshal(keywords)

	candidate := &vo.ChunkCandidate{
		SectionPath:       sectionPath,
		CanonicalPath:     canonicalPath,
		Text:              text,
		SourceType:        enum.ChunkSourceTypeOriginal,
		ContentWithWeight: "",
		ChunkType:         chunkType,
		Title:             title,
		Keywords:          string(keywordJson),
		Questions:         questions,
		PageNo:            blocks.FirstPageNo(),
		PageRange:         blocks.PageRange(),
		SourceBlockIds:    blocks.Ids(),
	}
	if node != nil {
		candidate.StructureNodeId = node.ID
		candidate.StructureNodeType = node.NodeType
		candidate.ItemIndex = node.ItemIndex
		candidate.CanonicalPath = utils.BlankToDefault(node.CanonicalPath, canonicalPath)
	}
	if len(blocks) > 0 {
		candidate.BboxJSON = blocks[0].BboxJSON
	}

	return candidate
}
