package index

import (
	"context"
	"encoding/json"
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/duke-git/lancet/v2/strutil"

	"github.com/swiftbit/know-agent/common/logx"
	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/document/adapter"
	"github.com/swiftbit/know-agent/internal/domain/document/logic/process/chunk"
	chunkllm "github.com/swiftbit/know-agent/internal/domain/document/logic/process/chunk/llm"
	chunkrecursive "github.com/swiftbit/know-agent/internal/domain/document/logic/process/chunk/recursive"
	chunksemantic "github.com/swiftbit/know-agent/internal/domain/document/logic/process/chunk/semantic"
	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/document/model/enum"
	"github.com/swiftbit/know-agent/internal/domain/document/model/vo"
	errorx "github.com/swiftbit/know-agent/internal/error"
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
		logx.Infof("从已提交 GraphRAG outcome 恢复索引任务，跳过切块流水线: documentId=%d, taskId=%d",
			buildCtx.DocumentId, buildCtx.TaskId)
		return nil
	}

	// 查询结构化解析 blocks
	blocks, err := p.repo.SelectDocumentBlocksByTask(ctx, buildCtx.DocumentId, buildCtx.Task.SourceParseTaskId)
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

	// 按步骤执行切块流水线
	chunkStartTime := time.Now()
	parentCandidates, err := p.BuildParentBlocks(ctx, buildCtx.Document, pipelineSteps, blocks)
	if err != nil {
		return err
	}
	buildCtx.ParentCandidates = parentCandidates
	costMillis := time.Since(chunkStartTime).Milliseconds()

	if err = p.repo.UpdateStepExecuteStatus(ctx, buildCtx.PlanId, enum.StrategyExecuteStatusExecuteSuccess); err != nil {
		return err
	}
	chunkEndDetail, _ := json.Marshal(map[string]any{
		"parentCount": len(parentCandidates),
		"childCount":  parentCandidates.CountChildCandidates(),
		"blockCount":  len(blocks),
		"costMillis":  costMillis,
	})
	chunkEndLog := &entity.DocumentTaskLog{
		TaskId:       buildCtx.TaskId,
		DocumentId:   buildCtx.DocumentId,
		StageType:    enum.TaskStageChunkExecute,
		EventType:    enum.TaskEventComplete,
		LogLevel:     enum.LogLevelInfo,
		OperatorType: enum.OperatorTypeSystem,
		Content:      "切块流水线执行完成",
		DetailJson:   string(chunkEndDetail),
	}
	_ = p.repo.InsertTaskLog(ctx, chunkEndLog)

	return nil
}

// BuildParentBlocks 执行完整的父-子块构建流程：先通过父块流水线生成父种子，再针对每个父种子走子块流水线产出子块
func (p *ChunkingPhase) BuildParentBlocks(ctx context.Context, document *entity.Document,
	steps entity.DocumentStrategySteps, blocks entity.DocumentBlocks) (vo.ParentChunkCandidates, error) {
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
	blockMap := blocks.ToMap()

	// 加载已解析的文档结构节点（用于结构切块策略）
	var nodes []*entity.StructureNode
	var err error
	if document != nil {
		nodes, err = p.repo.SelectStructureNodeListByTask(ctx, document.ID, document.LastParseTaskId)
		if err != nil {
			return nil, err
		}
	}

	options := p.resolver.Resolve(ctx, document)

	// 生成父块种子列表
	parentSeedList := p.buildParentSeedList(ctx, blocks, parentSteps, nodes, options)

	// 为每个父块种子派生其子块；无子块时以父块本身兜底
	parentChunks := make(vo.ParentChunkCandidates, 0)
	for _, parentSeed := range parentSeedList.CleanupAndUnique() {
		if parentSeed != nil && strutil.IsNotBlank(parentSeed.Text) {
			childSeedList := p.buildChildSeedList(ctx, parentSeed, childSteps, blockMap, options)
			finalChildren := childSeedList.CleanupAndUnique()

			trim := strutil.Trim(parentSeed.Text)
			if len(finalChildren) == 0 {
				// 兜底策略：子块流水线无产出 → 使用父块本身作为唯一子块
				finalChildren = vo.ChunkCandidates{
					p.cloneChunkCandidate(parentSeed, trim),
				}
			}

			parentChunks = append(parentChunks, &vo.ParentChunkCandidate{
				SectionPath:       parentSeed.SectionPath,
				StructureNodeId:   parentSeed.StructureNodeId,
				StructureNodeType: parentSeed.StructureNodeType,
				Text:              trim,
				SourceType:        parentSeed.SourceType,
				ChildChunks:       finalChildren,
				CanonicalPath:     parentSeed.CanonicalPath,
				ItemIndex:         parentSeed.ItemIndex,
				PageRange:         parentSeed.PageRange,
				SourceBlockIds:    parentSeed.SourceBlockIds,
			})
		}
	}

	// 对父块进行去重与清理后返回
	return parentChunks.CleanupAndUnique(), nil
}

// ---------------- 种子构建 ----------------

func (p *ChunkingPhase) buildParentSeedList(ctx context.Context, blocks entity.DocumentBlocks, parentSteps entity.DocumentStrategySteps,
	nodes []*entity.StructureNode, options *vo.IndexingOptions) vo.ChunkCandidates {
	if parentSteps.Contains(enum.StrategyTypeStructure) {
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
	return p.executePipeline(ctx, parentSeeds, parentSteps, enum.PipelineTypeParent, options)
}

func (p *ChunkingPhase) buildChildSeedList(ctx context.Context, parentSeed *vo.ChunkCandidate, childSteps entity.DocumentStrategySteps,
	blockMap map[int64]*entity.DocumentBlock, options *vo.IndexingOptions) vo.ChunkCandidates {
	blockSeeds := p.buildBlockChildSeeds(parentSeed, blockMap)
	if len(blockSeeds) == 0 {
		blockSeeds = vo.ChunkCandidates{p.cloneChunkCandidate(parentSeed, parentSeed.Text)}
	}
	remainingSteps := childSteps.DeleteStep(enum.StrategyTypeStructure)
	if len(remainingSteps) == 0 {
		return blockSeeds
	}
	// 执行子块流水线
	return p.executePipeline(ctx, blockSeeds, remainingSteps, enum.PipelineTypeChild, options)
}

// buildBlockSectionParentSeeds 构建基于章节分组的父块种子, 按 SectionPath 分组，组内按字符上限决定是否窗口化拆分
func (p *ChunkingPhase) buildBlockSectionParentSeeds(blocks entity.DocumentBlocks,
	nodes []*entity.StructureNode, options *vo.IndexingOptions) vo.ChunkCandidates {

	seeds := make(vo.ChunkCandidates, 0)
	currentGroup := make(entity.DocumentBlocks, 0)
	currentSectionKey := ""
	maxChars := options.ResolveRecursiveMaxChars(enum.PipelineTypeParent)

	// 闭包：处理一个完整组的收尾逻辑
	add := func(group entity.DocumentBlocks) {
		if len(group) == 0 {
			return
		}

		text := group.JoinBlockTexts()
		// 若组内总字符数在限制内，直接生成父种子
		if utils.Len(text) <= maxChars {
			seeds = append(seeds, p.toParentSeed(group, nodes))
		} else {
			// 超出限制，走滑动窗口拆分逻辑
			seeds = append(seeds, p.buildBlockWindowParentSeeds(group, nodes, maxChars)...)
		}
	}

	// 遍历所有块，按章节路径分组
	for _, block := range blocks {
		sectionKey := strings.TrimSpace(block.SectionPath)
		sectionChanged := len(currentGroup) > 0 && currentSectionKey != sectionKey

		// 章节发生变化，提交上一组
		if sectionChanged {
			add(currentGroup)
			currentGroup = make(entity.DocumentBlocks, 0)
		}

		// 新组初始化章节标识
		if len(currentGroup) == 0 {
			currentSectionKey = sectionKey
		}

		currentGroup = append(currentGroup, block)
	}

	// 处理最后一组
	add(currentGroup)

	// 兜底策略：若未生成任何种子，则对整个块列表做窗口化拆分
	if len(seeds) == 0 {
		return p.buildBlockWindowParentSeeds(blocks, nodes, maxChars)
	}

	return seeds
}

// buildBlockChildSeeds 构建父块对应的子块种子列表，根据父块的源块ID列表，从blockMap中提取对应的文档块，生成独立的子块候选对象
func (p *ChunkingPhase) buildBlockChildSeeds(parentSeed *vo.ChunkCandidate, blockMap map[int64]*entity.DocumentBlock) vo.ChunkCandidates {
	if parentSeed == nil || blockMap == nil || len(blockMap) == 0 {
		return nil
	}
	// 解析父块中的源块ID列表
	sourceBlockIds := make([]int64, 0)
	if err := json.Unmarshal([]byte(parentSeed.SourceBlockIds), &sourceBlockIds); err != nil {
		logx.Errorf("解析父块源块ID列表失败: %s\n", err.Error())
		return nil
	}

	if len(sourceBlockIds) == 0 {
		return nil
	}

	seeds := make(vo.ChunkCandidates, 0, len(sourceBlockIds))
	for _, blockID := range sourceBlockIds {
		block, exists := blockMap[blockID]
		if !exists || block == nil {
			continue
		}
		child := p.toBlockChunkCandidate(block)
		child.SectionPath = utils.BlankToDefault(child.SectionPath, parentSeed.SectionPath)
		child.StructureNodeId = parentSeed.StructureNodeId
		child.StructureNodeType = parentSeed.StructureNodeType
		child.CanonicalPath = utils.BlankToDefault(child.CanonicalPath, parentSeed.CanonicalPath)
		child.ItemIndex = parentSeed.ItemIndex
		seeds = append(seeds, child)
	}

	return seeds
}

// buildBlockWindowParentSeeds 构建块窗口父种子
func (p *ChunkingPhase) buildBlockWindowParentSeeds(blocks entity.DocumentBlocks, nodes []*entity.StructureNode, maxChars int) vo.ChunkCandidates {
	seeds := make(vo.ChunkCandidates, 0)
	currentBlocks := make(entity.DocumentBlocks, 0)
	currentChars := 0
	recursiveSplit := p.registry[enum.StrategyTypeRecursive]
	for _, block := range blocks {
		blockText := strings.TrimSpace(block.RenderBlockContent())
		if blockText == "" {
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
			parts, _ := recursiveSplit.Chunk(context.TODO(), blockText,
				chunkrecursive.WithMaxChars(maxChars),
				chunkrecursive.WithOverlapChars(0))
			for _, splitText := range parts {
				seeds = append(seeds, p.toSplitBlockSeed(block, nodes, splitText))
			}
			continue
		}

		// 检查窗口是否已满（考虑块间分隔符长度）
		if len(currentBlocks) > 0 && currentChars+utils.Len(blockText)+2 > maxChars {
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

// cloneChunkCandidate 克隆 ChunkCandidate
func (p *ChunkingPhase) cloneChunkCandidate(original *vo.ChunkCandidate, text string) *vo.ChunkCandidate {
	text = strings.TrimSpace(text)
	if original == nil {
		return &vo.ChunkCandidate{
			Text:       text,
			SourceType: enum.ChunkSourceTypeOriginal,
		}
	}
	keywords := strings.TrimSpace(original.Keywords)
	if keywords == "" {
		keywords = p.buildKeywords(original.Title, original.SectionPath, text)
	}
	questions := strings.TrimSpace(original.Questions)
	if questions == "" {
		questions = p.buildQuestions(original.Title, original.ChunkType, keywords)
	}

	contentWithWeight := original.ContentWithWeight
	if text != strings.TrimSpace(original.Text) || original.ContentWithWeight == "" {
		contentWithWeight = p.buildContentWithWeight(text, original.SectionPath, original.Title, original.ChunkType, keywords, questions, "")
	}
	candidate := *original
	candidate.Text = text
	candidate.Keywords = keywords
	candidate.Questions = questions
	candidate.ContentWithWeight = contentWithWeight
	return &candidate
}

// ---------------- 流水线调度 ----------------

func (p *ChunkingPhase) executePipeline(ctx context.Context, seeds vo.ChunkCandidates,
	steps entity.DocumentStrategySteps, pipelineType string, options *vo.IndexingOptions) vo.ChunkCandidates {
	// 初始清洗与去重
	chunks := seeds.CleanupAndUnique()
	if len(chunks) == 0 {
		return chunks
	}

	// 按配置的步骤顺序执行分块策略
	for _, step := range steps {
		strategy, ok := p.registry[step.StrategyType]
		// 跳过未注册的策略或结构型策略（由其他流程处理）
		if !ok || step.StrategyType == enum.StrategyTypeStructure {
			continue
		}

		// 为当前策略构建运行时参数
		extraOpts := p.buildPipelineOptions(options, step.StrategyType, pipelineType)
		currentChunks := make(vo.ChunkCandidates, 0, len(chunks))

		// 对每一个候选文本块执行分块策略
		for _, candidate := range chunks {
			if candidate == nil || strutil.IsBlank(candidate.Text) {
				continue
			}
			outputs, err := strategy.Chunk(ctx, candidate.Text, extraOpts...)
			if err != nil {
				logx.Errorf("分块策略 %d 处理块 %s 时出错: %v", step.StrategyType, candidate.Text, err)
				continue
			}
			// 将策略输出转换为新的候选块，并继承父块元数据
			for _, output := range outputs {
				currentChunks = append(currentChunks, p.cloneChunkCandidate(candidate, output))
			}
		}

		// 合并新生成的块并重新清洗去重，作为下一轮的输入
		chunks = append(chunks, currentChunks...)
		chunks = currentChunks.CleanupAndUnique()
	}

	// 最终清洗并返回结果
	return chunks.CleanupAndUnique()
}

// buildPipelineOptions 根据流水线类型和策略类型生成额外的策略选项
func (p *ChunkingPhase) buildPipelineOptions(options *vo.IndexingOptions, strategyType int, pipelineType string) []chunk.Option {
	switch strategyType {
	case enum.StrategyTypeRecursive:
		maxChars := options.ResolveRecursiveMaxChars(pipelineType)
		overlap := options.ResolveRecursiveOverlap(pipelineType, maxChars)
		return []chunk.Option{
			chunkrecursive.WithMaxChars(maxChars),
			chunkrecursive.WithOverlapChars(overlap),
		}
	case enum.StrategyTypeSemantic:
		maxChars := options.ResolveSemanticMaxChars(pipelineType)
		minChars := options.ResolveSemanticMinChars(pipelineType)
		return []chunk.Option{
			chunksemantic.WithMaxChars(maxChars),
			chunksemantic.WithMinChars(minChars),
			chunksemantic.WithSimilarityThreshold(options.Chunk.ChildSemanticSimilarityThreshold),
		}
	case enum.StrategyTypeLLM:
		return []chunk.Option{
			chunkllm.WithLlmMaxChars(options.ResolveLlmMaxChars(pipelineType)),
		}
	default:
		return nil
	}
}

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
	chunkType := block.ResolveChunkType()
	keywords := p.buildKeywords(title, block.SectionPath, text)
	questions := p.buildQuestions(title, chunkType, keywords)
	return p.buildContentWithWeight(text, block.SectionPath, title, chunkType, keywords, questions, "")
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

var re = regexp.MustCompile(`[>/|]`)

func (p *ChunkingPhase) buildKeywords(title, sectionPath, text string) string {
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
	data, _ := json.Marshal(keywords)

	return string(data)
}

// buildQuestions 构建面向检索的问答对, 基于标题、块类型和关键词生成假设性问题
func (p *ChunkingPhase) buildQuestions(title, chunkType, keywords string) string {
	seen := make(map[string]bool, 4)
	questions := make([]string, 0, 4)
	add := func(question string) {
		if !seen[question] && len(questions) < 4 {
			seen[question] = true
			questions = append(questions, question)
		}
	}

	// 确定主题词：优先使用标题，否则取关键词列表的第一个
	keywordSlice := parseJsonArray(keywords)
	topic := strings.TrimSpace(title)
	if topic == "" {
		if len(keywordSlice) > 0 {
			topic = keywordSlice[0]
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
		array := parseJsonArray(jsonArray)
		return strings.Join(array, ";")
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
		candidate.BboxJson = blocks[0].BboxJson
	}

	return candidate
}

// toSplitBlockSeed 处理超长文档块被递归拆分后的单个子块
func (p *ChunkingPhase) toSplitBlockSeed(block *entity.DocumentBlock, nodes entity.StructureNodes, splitText string) *vo.ChunkCandidate {
	sectionPath := block.SectionPath
	canonicalPath := block.CanonicalPath
	node := nodes.FindNodeByPath(sectionPath, canonicalPath)
	title := block.ResolveTitle(sectionPath)
	chunkType := block.ResolveChunkType()
	keywords := p.buildKeywords(title, sectionPath, splitText)
	questions := p.buildQuestions(title, chunkType, keywords)
	contentWithWeight := p.buildContentWithWeight(splitText, sectionPath, title, chunkType, keywords, questions, "")
	candidate := &vo.ChunkCandidate{
		SectionPath:       sectionPath,
		CanonicalPath:     canonicalPath,
		Text:              splitText,
		SourceType:        enum.ChunkSourceTypeOriginal,
		ContentWithWeight: contentWithWeight,
		ChunkType:         chunkType,
		Title:             title,
		Keywords:          keywords,
		Questions:         questions,
		PageNo:            block.PageNo,
		PageRange:         block.PageRange,
		BboxJson:          block.BboxJson,
		SourceBlockIds:    block.Ids(),
	}

	// 如果找到结构节点，覆盖相关字段
	if node != nil {
		candidate.StructureNodeId = node.ID
		candidate.StructureNodeType = node.NodeType
		candidate.ItemIndex = node.ItemIndex
		candidate.CanonicalPath = utils.BlankToDefault(node.CanonicalPath, canonicalPath)
	}

	return candidate
}

// ToBlockChunkCandidate 将文档块转换为ChunkCandidate
func (p *ChunkingPhase) toBlockChunkCandidate(block *entity.DocumentBlock) *vo.ChunkCandidate {
	text := block.RenderBlockContent()
	sectionPath := block.SectionPath
	title := block.ResolveTitle(sectionPath)
	chunkType := block.ResolveChunkType()
	keywords := p.buildKeywords(title, sectionPath, text)
	questions := p.buildQuestions(title, chunkType, keywords)
	weightedContent := p.renderBlockWeightedContent(block)
	contentWithWeight := p.buildContentWithWeight(text, sectionPath, title, chunkType, keywords, questions, weightedContent)
	return &vo.ChunkCandidate{
		SectionPath:       sectionPath,
		CanonicalPath:     block.CanonicalPath,
		Text:              text,
		ContentWithWeight: contentWithWeight,
		ChunkType:         chunkType,
		Title:             title,
		Keywords:          keywords,
		Questions:         questions,
		SourceType:        enum.ChunkSourceTypeOriginal,
		PageNo:            block.PageNo,
		PageRange:         block.PageRange,
		BboxJson:          block.BboxJson,
		SourceBlockIds:    block.Ids(),
	}
}

func parseJsonArray(jsonArray string) []string {
	if jsonArray == "" {
		return nil
	}
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
	return results[:j]
}
