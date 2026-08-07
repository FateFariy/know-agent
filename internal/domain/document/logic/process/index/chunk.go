package index

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/duke-git/lancet/v2/maputil"
	"github.com/duke-git/lancet/v2/slice"
	"github.com/duke-git/lancet/v2/stream"
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
	repo     adapter.DocumentRepository
	port     *adapter.DocumentPort
	registry map[int]chunk.Chunker
	option   *chunkingOption
}

// chunkingOption 切块配置项
type chunkingOption struct {
	semanticMaxChars int
	semanticMinChars int
	llmEnabled       bool
	llmMaxChars      int
}

// NewChunkingPhase 创建切块阶段
func NewChunkingPhase(repo adapter.DocumentRepository, port *adapter.DocumentPort,
	registry map[int]chunk.Chunker, opt *chunkingOption) *ChunkingPhase {
	return &ChunkingPhase{
		repo:     repo,
		port:     port,
		registry: registry,
		option:   opt,
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
	steps []*entity.DocumentStrategyStep, blocks []*entity.DocumentBlock) ([]*vo.ParentBlockCandidate, error) {
	// 按父/子流水线拆分并排序步骤；任一缺失则返回相应错误
	parentSteps := p.sortPipelineSteps(steps, enum.PipelineTypeParent)
	childSteps := p.sortPipelineSteps(steps, enum.PipelineTypeChild)
	if len(parentSteps) == 0 {
		return nil, errorx.ErrParentBlockMissing
	}
	if len(childSteps) == 0 {
		return nil, errorx.ErrChildBlockMissing
	}

	orderedBlocks := p.cleanupBlocks(blocks)
	if len(orderedBlocks) == 0 {
		return nil, errorx.ErrDocumentBlocksMissing
	}
	blockMap := utils.SliceToMapBy(orderedBlocks, func(block *entity.DocumentBlock) (int64, *entity.DocumentBlock) {
		return block.ID, block
	})

	// 加载已解析的文档结构节点（用于结构切块策略）
	var structureNodes []*entity.StructureNode
	if document != nil {
		nodes, err := p.repo.SelectStructureNodeListByTask(ctx, document.ID, document.LastParseTaskId)
		if err != nil {
			return nil, err
		}
		structureNodes = nodes
	}

	// 从文档存储中读取解析后的全文（用于兜底：结构节点不可用时直接以全文走流水线）
	parsedText := ""
	if document != nil && strutil.IsNotBlank(document.ParseTextPath) {
		if text, err := p.port.DownloadText(ctx, document.ParseTextPath); err == nil {
			parsedText = text
		}
	}

	// 生成父块种子列表
	parentSeedList := p.buildParentSeedList(ctx, parsedText, parentSteps, structureNodes)

	// 为每个父块种子派生其子块；无子块时以父块本身兜底
	parentBlockList := make([]*vo.ParentBlockCandidate, 0)
	for _, parentSeed := range p.cleanupChunkList(parentSeedList) {
		if parentSeed != nil && strutil.IsNotBlank(parentSeed.Text) {
			childSeedList := p.buildChildSeedList(ctx, parentSeed, childSteps, structureNodes)
			finalChildren := p.cleanupChunkList(childSeedList)

			trim := strutil.Trim(parentSeed.Text)
			if len(finalChildren) == 0 {
				// 兜底策略：子块流水线无产出 → 使用父块本身作为唯一子块
				finalChildren = []*vo.ChunkCandidate{
					p.cloneChunkCandidate(parentSeed, trim),
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
func (p *ChunkingPhase) cleanupBlocks(blocks []*entity.DocumentBlock) []*entity.DocumentBlock {
	if len(blocks) == 0 {
		return nil
	}
	less := func(a, b *entity.DocumentBlock) bool {
		if a.BlockNo == 0 {
			return false
		} else if a.BlockNo != b.BlockNo {
			return a.BlockNo < b.BlockNo
		}
		return a.ID < b.ID
	}
	predicate := func(item *entity.DocumentBlock) bool {
		return item != nil && item.HasBlockContent()
	}
	return stream.FromSlice(blocks).Filter(predicate).Sorted(less).ToSlice()
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

// sortPipelineSteps 过滤属于指定流水线的步骤并按 StepNo 升序排列
func (p *ChunkingPhase) sortPipelineSteps(steps []*entity.DocumentStrategyStep, pipelineType string) []*entity.DocumentStrategyStep {
	filtered := slice.Filter(steps, func(index int, item *entity.DocumentStrategyStep) bool {
		return utils.EqualsIgnoreCase(pipelineType, utils.BlankToDefault(item.PipelineType, enum.PipelineTypeChild))
	})
	slice.SortBy(filtered, func(a, b *entity.DocumentStrategyStep) bool { return a.StepNo < b.StepNo })
	return filtered
}

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
func (p *ChunkingPhase) buildParentSeedList(ctx context.Context, parsedText string,
	parentSteps []*entity.DocumentStrategyStep, structureNodes []*entity.StructureNode) []*vo.ChunkCandidate {
	if p.containsStructureStep(parentSteps) && len(structureNodes) > 0 {
		// 结构切块有节点可用 → 先产出章节级种子，再将剩余策略作为后续流水线
		structureSeeds := p.buildStructureParentSeeds(structureNodes)
		if len(structureSeeds) != 0 {
			remainingSteps := p.stripStructureSteps(parentSteps)
			if len(remainingSteps) == 0 {
				return structureSeeds
			}

			return p.executePipeline(ctx, structureSeeds, remainingSteps, enum.PipelineTypeParent)
		}
	}

	// 无结构步骤或节点 → 用整段文本作为父种子走完整流水线
	originalSeed := &vo.ChunkCandidate{
		Text:       parsedText,
		SourceType: enum.ChunkSourceTypeOriginal,
	}

	// 执行父块流水线
	return p.executePipeline(ctx, []*vo.ChunkCandidate{originalSeed}, parentSteps, enum.PipelineTypeParent)
}

// buildChildSeedList 为指定父种子构建子块种子列表，若步骤中含有结构切块且结构节点存在，优先按结构节点拆解子章节，否则克隆父种子再跑流水线
func (p *ChunkingPhase) buildChildSeedList(ctx context.Context, parentSeed *vo.ChunkCandidate,
	childSteps []*entity.DocumentStrategyStep, structureNodes []*entity.StructureNode) []*vo.ChunkCandidate {
	if p.containsStructureStep(childSteps) && parentSeed.StructureNodeId != 0 && len(structureNodes) > 0 {
		// 基于父种子的节点 ID 收集子节点，再进入后续流水线
		structureSeeds := p.buildStructureChildSeeds(parentSeed, structureNodes)

		remainingSteps := p.stripStructureSteps(childSteps)
		if len(remainingSteps) == 0 {
			return structureSeeds
		}

		return p.executePipeline(ctx, structureSeeds, remainingSteps, enum.PipelineTypeChild)
	}

	// 直接克隆父种子作为子块流水线的起点
	clonedSeed := p.cloneChunkCandidate(parentSeed, parentSeed.Text)

	// 执行子块流水线
	return p.executePipeline(ctx, []*vo.ChunkCandidate{clonedSeed}, childSteps, enum.PipelineTypeChild)
}

// containsStructureStep 检查步骤列表中是否存在结构切块策略
func (p *ChunkingPhase) containsStructureStep(steps []*entity.DocumentStrategyStep) bool {
	for _, step := range steps {
		if step.StrategyType == enum.StrategyTypeStructure {
			return true
		}
	}
	return false
}

// stripStructureSteps 过滤掉结构切块步骤（结构切块已经在流水线前处理）
func (p *ChunkingPhase) stripStructureSteps(steps []*entity.DocumentStrategyStep) []*entity.DocumentStrategyStep {
	return slice.Filter(steps, func(index int, step *entity.DocumentStrategyStep) bool {
		return step.StrategyType != enum.StrategyTypeStructure
	})
}

// buildStructureParentSeeds 从结构节点中筛选"内容承载章节"生成父块种子，判定规则：含有子章节时需额外验证内容长度显著超过标题或出现换行
func (p *ChunkingPhase) buildStructureParentSeeds(structureNodes []*entity.StructureNode) []*vo.ChunkCandidate {
	// 预计算：哪些节点拥有子章节（用于后续内容判定）
	parentHasChildSection := make(map[int64]bool)
	for _, node := range structureNodes {
		if node.ParentNodeId != 0 && node.NodeType == vo.NodeTypeSection {
			parentHasChildSection[node.ParentNodeId] = true
		}
	}

	// 产出章节种子（仅保留"有实质内容"的章节）
	seeds := make([]*vo.ChunkCandidate, 0, len(structureNodes))
	for _, node := range structureNodes {
		if node.NodeType == vo.NodeTypeSection && p.isContentBearingSection(node, parentHasChildSection[node.ID]) {
			seeds = append(seeds, p.newChunkCandidate(node, enum.ChunkSourceTypeOriginal))
		}
	}

	return seeds
}

// buildStructureChildSeeds 根据父种子的节点 ID 从结构节点中挑出其子节点作为子块种子。
// 仅保留 SECTION / STEP / LIST_ITEM 三类有实际内容的子节点；否则回退到克隆父种子。
func (p *ChunkingPhase) buildStructureChildSeeds(parentSeed *vo.ChunkCandidate, structureNodes []*entity.StructureNode) []*vo.ChunkCandidate {
	// 按 ParentNodeId 索引结构节点，快速定位当前父种子的子节点集合
	childrenByParent := make(map[int64][]*entity.StructureNode)
	for _, node := range structureNodes {
		if node.ParentNodeId != 0 {
			childrenByParent[node.ParentNodeId] = append(childrenByParent[node.ParentNodeId], node)
		}
	}

	seeds := make([]*vo.ChunkCandidate, 0)
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
	return []*vo.ChunkCandidate{p.cloneChunkCandidate(parentSeed, parentSeed.Text)}
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

// cloneChunkCandidate 克隆 ChunkCandidate；可替换文本字段，其他元信息保留
func (p *ChunkingPhase) cloneChunkCandidate(original *vo.ChunkCandidate, text string) *vo.ChunkCandidate {
	if original == nil {
		return &vo.ChunkCandidate{
			Text:       text,
			SourceType: enum.ChunkSourceTypeOriginal,
		}
	}
	return &vo.ChunkCandidate{
		SectionPath:       original.SectionPath,
		StructureNodeId:   original.StructureNodeId,
		StructureNodeType: original.StructureNodeType,
		CanonicalPath:     original.CanonicalPath,
		ItemIndex:         original.ItemIndex,
		Text:              text,
		SourceType:        original.SourceType,
	}
}

// cloneParentBlockCandidate 克隆 ParentBlockCandidate
func (p *ChunkingPhase) cloneParentBlockCandidate(source *vo.ParentBlockCandidate, childChunks []*vo.ChunkCandidate, text string) *vo.ParentBlockCandidate {
	if source == nil {
		return &vo.ParentBlockCandidate{
			Text:        text,
			SourceType:  enum.ChunkSourceTypeOriginal,
			ChildChunks: append([]*vo.ChunkCandidate{}, childChunks...),
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
		ChildChunks:       append([]*vo.ChunkCandidate{}, childChunks...),
	}
}

// ---------------- 流水线调度 ----------------

// executePipeline 按步骤顺序调度分块策略，当前步骤的输出作为下一步骤的输入
func (p *ChunkingPhase) executePipeline(ctx context.Context, inputSeeds []*vo.ChunkCandidate, steps []*entity.DocumentStrategyStep, pipelineType string) []*vo.ChunkCandidate {
	// 初次清理：去除空文本和重复项
	currentChunks := p.cleanupChunkList(inputSeeds)
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

		nextChunks := make([]*vo.ChunkCandidate, 0, len(currentChunks))
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
					nextChunks = append(nextChunks, p.cloneChunkCandidate(candidate, out.Text))
				}
			}
		}
		// 每步结束后清理，避免中间产物污染下游
		currentChunks = p.cleanupChunkList(nextChunks)
	}
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

// cleanupChunkList 清理 ChunkCandidate 列表：过滤空文本并按 路径+序号+文本 去重
func (p *ChunkingPhase) cleanupChunkList(chunks []*vo.ChunkCandidate) []*vo.ChunkCandidate {
	result := make(map[string]*vo.ChunkCandidate)
	for _, candidate := range chunks {
		if candidate != nil && strutil.IsNotBlank(candidate.Text) {
			path := utils.BlankToDefault(candidate.CanonicalPath, candidate.SectionPath)
			trim := strutil.Trim(candidate.Text)
			uniqueKey := fmt.Sprintf("%s||%d||%s", path, candidate.ItemIndex, trim)
			if _, ok := result[uniqueKey]; !ok {
				result[uniqueKey] = p.cloneChunkCandidate(candidate, trim)
			}
		}
	}
	return maputil.Values(result)
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
