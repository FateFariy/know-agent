package logic

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/cloudwego/eino/schema"
	"github.com/duke-git/lancet/v2/maputil"
	"github.com/duke-git/lancet/v2/slice"
	"github.com/duke-git/lancet/v2/stream"
	"github.com/duke-git/lancet/v2/strutil"

	"github.com/swiftbit/know-agent/common/utils"
	chatlogic "github.com/swiftbit/know-agent/internal/domain/chat/logic"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/prompt"
	"github.com/swiftbit/know-agent/internal/domain/document/logic/chunk"
	chunkllm "github.com/swiftbit/know-agent/internal/domain/document/logic/chunk/llm"
	chunkrecursive "github.com/swiftbit/know-agent/internal/domain/document/logic/chunk/recursive"
	chunksemantic "github.com/swiftbit/know-agent/internal/domain/document/logic/chunk/semantic"
	chunkstructure "github.com/swiftbit/know-agent/internal/domain/document/logic/chunk/structure"
	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/document/model/vo"
	"github.com/swiftbit/know-agent/internal/domain/document/support"
	errorx "github.com/swiftbit/know-agent/internal/error"
	"github.com/swiftbit/know-agent/internal/svc"
)

const (
	ParentBlockMaxChars     = 2200 // 父块最大字符数
	ParentBlockOverlapChars = 180  // 父块重叠字符数
	ParentSemanticMaxChars  = 1600 // 语义块最大字符数
	ParentSemanticMinChars  = 480  // 语义块最小字符数
)

// StrategyLogicImpl 策略业务逻辑实现
type StrategyLogicImpl struct {
	structureNode StructureNodeLogic
	registry      map[int]chunk.Strategy
	classifier    *support.DocumentLineClassifier
	*strategyOption
}

type strategyOption struct {
	recursiveMaxChars           int
	recursiveOverlapChars       int
	semanticMaxChars            int
	semanticMinChars            int
	semanticSimilarityThreshold float64
	llmEnabled                  bool
	llmMaxChars                 int
	recommendLlmWhenLowQuality  bool
}

func NewStrategyLogic(svcCtx *svc.ServiceContext, chatModel *chatlogic.ObservedChatModelImpl[*schema.Message],
	promptTemplate chatlogic.PromptTemplateLogic, structureNode StructureNodeLogic) StrategyLogic {

	registry := make(map[int]chunk.Strategy)
	// 结构分块
	registry[vo.StrategyTypeStructure] = chunkstructure.NewStrategy()

	// 递归分块
	registry[vo.StrategyTypeRecursive] = chunkrecursive.NewStrategy(
		chunkrecursive.WithMaxChars(svcCtx.Config.Chunk.RecursiveMaxChars),
		chunkrecursive.WithOverlapChars(svcCtx.Config.Chunk.RecursiveOverlapChars),
	)

	// 语义分块
	registry[vo.StrategyTypeSemantic] = chunksemantic.NewStrategy(
		chunksemantic.WithMinChars(svcCtx.Config.Chunk.SemanticMinChars),
		chunksemantic.WithMaxChars(svcCtx.Config.Chunk.SemanticMaxChars),
		chunksemantic.WithSimilarityThreshold(svcCtx.Config.Chunk.SemanticSimilarityThreshold),
	)
	// 大模型切块
	registry[vo.StrategyTypeLLM] = chunkllm.NewStrategy(chatModel, promptTemplate,
		chunkllm.WithLlmSplitPrompt(prompt.DocumentLlmSplit),
	)

	return &StrategyLogicImpl{
		structureNode: structureNode,
		registry:      registry,
		classifier:    &support.DocumentLineClassifier{},
		strategyOption: &strategyOption{
			recursiveMaxChars:           svcCtx.Config.Chunk.RecursiveMaxChars,
			recursiveOverlapChars:       svcCtx.Config.Chunk.RecursiveOverlapChars,
			semanticMaxChars:            svcCtx.Config.Chunk.SemanticMaxChars,
			semanticMinChars:            svcCtx.Config.Chunk.SemanticMinChars,
			semanticSimilarityThreshold: svcCtx.Config.Chunk.SemanticSimilarityThreshold,
			llmEnabled:                  svcCtx.Config.Chunk.LlmEnabled,
			llmMaxChars:                 svcCtx.Config.Chunk.LlmMaxChars,
			recommendLlmWhenLowQuality:  svcCtx.Config.Chunk.RecommendLlmWhenLowQuality,
		},
	}
}

// RecommendStrategy 推荐策略方案，根据文档分析结果推荐最优的父块和子块策略组合
func (s *StrategyLogicImpl) RecommendStrategy(ctx context.Context, document *entity.Document, analysisResult *vo.DocumentAnalysisResult) (*vo.DocumentStrategyPlanDraft, error) {
	if document == nil || analysisResult == nil {
		return nil, nil
	}

	reasonList := make([]string, 0)

	// 根据文档特征判断推荐策略
	structureRecommended := s.shouldUseStructure(document, analysisResult)
	recursiveRecommended := s.shouldUseRecursive(analysisResult)
	semanticRecommended := s.shouldUseSemantic(analysisResult)
	llmRecommended := s.shouldUseLlm(analysisResult)

	// 构建父块策略
	parentStrategyTypes := make([]int, 0)
	parentReasonMap := make(map[int]string)

	if structureRecommended {
		parentStrategyTypes = append(parentStrategyTypes, vo.StrategyTypeStructure)
		parentReasonMap[vo.StrategyTypeStructure] = "检测到文档具有较明显的标题或章节结构，父块优先保留天然章节边界。"
		reasonList = append(reasonList, "父块流水线优先采用基于文档结构切块，保留回答阶段需要的大语义单元。")
	} else {
		parentStrategyTypes = append(parentStrategyTypes, vo.StrategyTypeRecursive)
		parentReasonMap[vo.StrategyTypeRecursive] = "未识别出稳定结构时，父块先使用较大粒度的递归分块作为稳定回答单元。"
		reasonList = append(reasonList, "父块流水线未命中明显结构信号，默认使用较大粒度递归分块作为回答单元。")
	}

	// 构建子块策略
	childStrategyTypes := make([]int, 0)
	childReasonMap := make(map[int]string)

	if llmRecommended {
		childStrategyTypes = append(childStrategyTypes, vo.StrategyTypeLLM)
		childReasonMap[vo.StrategyTypeLLM] = "文档质量偏低或结构识别不稳定，子块先使用大模型智能切块增强复杂场景。"
		reasonList = append(reasonList, "子块流水线追加大模型智能切块，处理低质量或结构不稳定文本。")
	} else if semanticRecommended {
		childStrategyTypes = append(childStrategyTypes, vo.StrategyTypeSemantic)
		childReasonMap[vo.StrategyTypeSemantic] = "文本主题边界相对明确，子块先使用语义分块优化召回边界。"
		reasonList = append(reasonList, "子块流水线优先采用语义分块，优化召回边界和主题完整性。")
	}

	// 默认添加递归分块作为兜底
	if recursiveRecommended || llmRecommended || len(childStrategyTypes) == 0 {
		childStrategyTypes = append(childStrategyTypes, vo.StrategyTypeRecursive)
		childReasonMap[vo.StrategyTypeRecursive] = "文档整体较长、存在超长段落，或需要在增强切块后追加长度兜底。"
		reasonList = append(reasonList, "子块流水线追加递归分块，控制召回单元长度并作为兜底。")
	}

	// 构建步骤草稿
	parentSteps := s.buildDraftSteps(vo.PipelineTypeParent, parentStrategyTypes, parentReasonMap)
	childSteps := s.buildDraftSteps(vo.PipelineTypeChild, childStrategyTypes, childReasonMap)

	// 构建策略快照
	strategySnapshot := fmt.Sprintf("PARENT:%s;CHILD:%s", s.buildPipelineSnapshot(parentSteps), s.buildPipelineSnapshot(childSteps))

	return &vo.DocumentStrategyPlanDraft{
		ParentSteps:      parentSteps,
		ChildSteps:       childSteps,
		StrategySnapshot: strategySnapshot,
		RecommendReason:  strings.Join(reasonList, "；"),
	}, nil
}

// NormalizeSteps 标准化策略步骤，将用户提交的策略类型标准化为可执行的步骤列表
func (s *StrategyLogicImpl) NormalizeSteps(ctx context.Context, baseSteps []*entity.DocumentStrategyStep,
	parentStrategyTypes []int, childStrategyTypes []int, documentId int64) ([]*entity.DocumentStrategyStep, error) {

	// 标准化策略类型
	normalizedParentTypes := s.normalizePipelineTypes(parentStrategyTypes)
	normalizedChildTypes := s.normalizePipelineTypes(childStrategyTypes)

	// 构建基础步骤映射
	baseStepMap := make(map[string]map[int]*entity.DocumentStrategyStep)
	for _, baseStep := range baseSteps {
		pipelineType := utils.BlankToDefault(baseStep.PipelineType, vo.PipelineTypeChild)
		if _, exists := baseStepMap[pipelineType]; !exists {
			baseStepMap[pipelineType] = make(map[int]*entity.DocumentStrategyStep)
		}
		baseStepMap[pipelineType][baseStep.StrategyType] = baseStep
	}

	// 构建标准化步骤列表
	normalizedStepList := make([]*entity.DocumentStrategyStep, 0)

	// 添加父块步骤
	parentSteps := s.buildNormalizedSteps(
		vo.PipelineTypeParent,
		normalizedParentTypes,
		baseStepMap[vo.PipelineTypeParent],
		documentId,
	)
	normalizedStepList = append(normalizedStepList, parentSteps...)

	// 添加子块步骤
	childSteps := s.buildNormalizedSteps(
		vo.PipelineTypeChild,
		normalizedChildTypes,
		baseStepMap[vo.PipelineTypeChild],
		documentId,
	)
	normalizedStepList = append(normalizedStepList, childSteps...)

	return normalizedStepList, nil
}

// BuildParentBlocks 构建父子块结构，根据策略方案执行父块和子块的切分，构建 Parent-Child 结构
func (s *StrategyLogicImpl) BuildParentBlocks(ctx context.Context, document *entity.Document,
	steps []*entity.DocumentStrategyStep, parsedText string) ([]*vo.ParentBlockCandidate, error) {
	// 按流水线类型排序步骤
	parentSteps := s.sortPipelineSteps(steps, vo.PipelineTypeParent)
	childSteps := s.sortPipelineSteps(steps, vo.PipelineTypeChild)
	if len(parentSteps) == 0 {
		return nil, errorx.ErrParentBlockMissing
	}
	if len(childSteps) == 0 {
		return nil, errorx.ErrChildBlockMissing
	}

	// 从结构节点服务加载结构节点
	var structureNodes []*entity.DocumentStructureNode
	if document != nil {
		nodes, err := s.structureNode.ListDocumentNodes(ctx, document.ID, document.LastParseTaskId)
		if err != nil {
			return nil, err
		}
		structureNodes = nodes
	}

	// 构建父块种子列表
	parentSeedList := s.buildParentSeedList(ctx, parsedText, parentSteps, structureNodes)

	// 为每个父块种子生成子块
	parentBlockList := make([]*vo.ParentBlockCandidate, 0)
	for _, parentSeed := range s.cleanupChunkList(parentSeedList) {
		if parentSeed == nil || strutil.IsBlank(parentSeed.Text) {
			continue
		}

		// 构建子块种子列表
		childSeedList := s.buildChildSeedList(ctx, parentSeed, childSteps, structureNodes)
		finalChildren := s.cleanupChunkList(childSeedList)

		// 如果没有子块，使用父块本身作为子块
		trim := strutil.Trim(parentSeed.Text)
		if len(finalChildren) == 0 {
			finalChildren = []*vo.ChunkCandidate{
				s.cloneChunkCandidate(parentSeed, trim),
			}
		}

		parentBlock := &vo.ParentBlockCandidate{
			SectionPath:       parentSeed.SectionPath,
			StructureNodeId:   parentSeed.StructureNodeId,
			StructureNodeType: parentSeed.StructureNodeType,
			Text:              trim,
			SourceType:        parentSeed.SourceType,
			ChildChunks:       finalChildren,
		}

		parentBlockList = append(parentBlockList, parentBlock)
	}

	return s.cleanupParentBlockList(parentBlockList), nil
}

// ---------------- 推荐判定 ----------------

// shouldUseStructure 判断是否应该使用结构切块
func (s *StrategyLogicImpl) shouldUseStructure(document *entity.Document, analysisResult *vo.DocumentAnalysisResult) bool {
	return vo.FileTypeName(document.FileType) != "" &&
		(analysisResult.StructureLevel >= vo.StructureLevelMedium || analysisResult.HeadingCount >= 2)
}

// shouldUseRecursive 判断是否应该使用递归切块
func (s *StrategyLogicImpl) shouldUseRecursive(analysisResult *vo.DocumentAnalysisResult) bool {
	return max(analysisResult.CharCount, analysisResult.MaxParagraphLength) >= s.recursiveMaxChars
}

// shouldUseSemantic 判断是否应该使用语义切块
func (s *StrategyLogicImpl) shouldUseSemantic(analysisResult *vo.DocumentAnalysisResult) bool {
	return analysisResult.CharCount >= s.semanticMinChars &&
		analysisResult.ContentQualityLevel >= vo.ContentQualityLevelMedium &&
		analysisResult.ParagraphCount >= 3
}

// shouldUseLlm 判断是否应该使用大模型智能切块
func (s *StrategyLogicImpl) shouldUseLlm(analysisResult *vo.DocumentAnalysisResult) bool {
	return s.recommendLlmWhenLowQuality &&
		analysisResult.ContentQualityLevel == vo.ContentQualityLevelLow &&
		analysisResult.CharCount >= s.semanticMinChars
}

// ---------------- 草稿/标准化 ----------------

// buildDraftSteps 构建步骤草稿
func (s *StrategyLogicImpl) buildDraftSteps(pipelineType string, strategyTypes []int, reasonMap map[int]string) []*vo.DocumentStrategyStepDraft {
	return slice.Map(strategyTypes, func(index, strategyType int) *vo.DocumentStrategyStepDraft {
		return &vo.DocumentStrategyStepDraft{
			PipelineType:    pipelineType,
			StrategyType:    strategyType,
			StrategyRole:    s.resolveRole(index, strategyType),
			SourceType:      vo.StrategySourceTypeSystemRecommend,
			RecommendReason: utils.BlankToDefault(reasonMap[strategyType], "系统为当前流水线生成的推荐步骤。"),
		}
	})
}

// normalizePipelineTypes 标准化流水线类型
func (s *StrategyLogicImpl) normalizePipelineTypes(strategyTypes []int) []int {
	return stream.FromSlice(strategyTypes).
		Filter(func(strategyType int) bool { return vo.StrategyTypeName(strategyType) != "" }).
		Distinct().ToSlice()
}

// buildNormalizedSteps 构建标准化步骤
func (s *StrategyLogicImpl) buildNormalizedSteps(pipelineType string, normalizedTypes []int,
	baseStepMap map[int]*entity.DocumentStrategyStep, documentId int64) []*entity.DocumentStrategyStep {
	return slice.Map(normalizedTypes, func(index, strategyType int) *entity.DocumentStrategyStep {
		baseStep := baseStepMap[strategyType]
		return &entity.DocumentStrategyStep{
			DocumentId:      documentId,
			PipelineType:    pipelineType,
			StepNo:          index + 1,
			StrategyType:    strategyType,
			StrategyRole:    s.resolveRole(index, strategyType),
			SourceType:      utils.Ternary(baseStep == nil, vo.StrategySourceTypeUserAdd, vo.StrategySourceTypeUserKeep),
			ExecuteStatus:   vo.StrategyExecuteStatusWaitExecute,
			RecommendReason: utils.Ternary(baseStep == nil, "用户手动追加该策略。", baseStep.RecommendReason),
		}
	})
}

// sortPipelineSteps 按流水线类型排序步骤
func (s *StrategyLogicImpl) sortPipelineSteps(steps []*entity.DocumentStrategyStep, pipelineType string) []*entity.DocumentStrategyStep {
	filtered := slice.Filter(steps, func(index int, item *entity.DocumentStrategyStep) bool {
		return utils.EqualsIgnoreCase(pipelineType, utils.BlankToDefault(item.PipelineType, vo.PipelineTypeChild))
	})
	slice.SortBy(filtered, func(a, b *entity.DocumentStrategyStep) bool { return a.StepNo < b.StepNo })
	return filtered
}

// buildPipelineSnapshot 将策略类型列表拼接为字符串
func (s *StrategyLogicImpl) buildPipelineSnapshot(steps []*vo.DocumentStrategyStepDraft) string {
	strList := slice.Map(steps, func(_ int, step *vo.DocumentStrategyStepDraft) string {
		return strconv.Itoa(step.StrategyType)
	})
	return strings.Join(strList, ",")
}

// resolveRole 解析策略角色
func (s *StrategyLogicImpl) resolveRole(index int, strategyType int) int {
	if index == 0 {
		return vo.StrategyRolePrimary
	}
	if strategyType == vo.StrategyTypeRecursive {
		return vo.StrategyRoleFallback
	}
	if strategyType == vo.StrategyTypeSemantic {
		return vo.StrategyRoleOptimize
	}
	if strategyType == vo.StrategyTypeLLM {
		return vo.StrategyRoleEnhance
	}
	return vo.StrategyRoleOptimize
}

// ---------------- 种子构建 ----------------

// buildParentSeedList 构建父块种子列表
func (s *StrategyLogicImpl) buildParentSeedList(ctx context.Context, parsedText string, parentSteps []*entity.DocumentStrategyStep, structureNodes []*entity.DocumentStructureNode) []*vo.ChunkCandidate {
	if s.containsStructureStep(parentSteps) && len(structureNodes) > 0 {
		// 如果有结构步骤且存在结构节点，优先使用结构切块
		structureSeeds := s.buildStructureParentSeeds(structureNodes)
		if len(structureSeeds) != 0 {
			// 移除结构步骤，继续执行其他策略
			remainingSteps := s.stripStructureSteps(parentSteps)
			if len(remainingSteps) == 0 {
				return structureSeeds
			}

			return s.executePipeline(ctx, structureSeeds, remainingSteps, vo.PipelineTypeParent)
		}
	}

	// 没有结构步骤或结构节点，直接使用原始文本
	originalSeed := &vo.ChunkCandidate{
		Text:       parsedText,
		SourceType: vo.ChunkSourceTypeOriginal,
	}
	return s.executePipeline(ctx, []*vo.ChunkCandidate{originalSeed}, parentSteps, vo.PipelineTypeParent)
}

// buildChildSeedList 构建子块种子列表
func (s *StrategyLogicImpl) buildChildSeedList(ctx context.Context, parentSeed *vo.ChunkCandidate, childSteps []*entity.DocumentStrategyStep, structureNodes []*entity.DocumentStructureNode) []*vo.ChunkCandidate {
	if s.containsStructureStep(childSteps) && parentSeed.StructureNodeId != 0 && len(structureNodes) > 0 {
		// 有结构步骤且父种子有节点ID，使用结构切块
		structureSeeds := s.buildStructureChildSeeds(parentSeed, structureNodes)

		// 移除结构步骤，继续执行其他策略
		remainingSteps := s.stripStructureSteps(childSteps)
		if len(remainingSteps) == 0 {
			return structureSeeds
		}

		return s.executePipeline(ctx, structureSeeds, remainingSteps, vo.PipelineTypeChild)
	}

	// 直接克隆父种子
	clonedSeed := s.cloneChunkCandidate(parentSeed, parentSeed.Text)

	return s.executePipeline(ctx, []*vo.ChunkCandidate{clonedSeed}, childSteps, vo.PipelineTypeChild)
}

// containsStructureStep 检查是否包含结构步骤
func (s *StrategyLogicImpl) containsStructureStep(steps []*entity.DocumentStrategyStep) bool {
	for _, step := range steps {
		if step.StrategyType == vo.StrategyTypeStructure {
			return true
		}
	}
	return false
}

// stripStructureSteps 移除结构步骤
func (s *StrategyLogicImpl) stripStructureSteps(steps []*entity.DocumentStrategyStep) []*entity.DocumentStrategyStep {
	return slice.Filter(steps, func(index int, step *entity.DocumentStrategyStep) bool {
		return step.StrategyType != vo.StrategyTypeStructure
	})
}

// buildStructureParentSeeds 构建结构父块种子
func (s *StrategyLogicImpl) buildStructureParentSeeds(structureNodes []*entity.DocumentStructureNode) []*vo.ChunkCandidate {
	// 检测哪些父节点有子章节
	parentHasChildSection := make(map[int64]bool)
	for _, node := range structureNodes {
		if node.ParentNodeId != 0 && node.NodeType == vo.NodeTypeSection {
			parentHasChildSection[node.ParentNodeId] = true
		}
	}

	// 筛选出内容承载的章节
	seeds := make([]*vo.ChunkCandidate, 0, len(structureNodes))
	for _, node := range structureNodes {
		if node.NodeType == vo.NodeTypeSection && s.isContentBearingSection(node, parentHasChildSection[node.ID]) {
			seeds = append(seeds, s.newChunkCandidate(node, vo.ChunkSourceTypeOriginal))
		}
	}

	return seeds
}

// buildStructureChildSeeds 构建结构子块种子
func (s *StrategyLogicImpl) buildStructureChildSeeds(parentSeed *vo.ChunkCandidate, structureNodes []*entity.DocumentStructureNode) []*vo.ChunkCandidate {
	// 按父节点分组
	childrenByParent := make(map[int64][]*entity.DocumentStructureNode)
	for _, node := range structureNodes {
		if node.ParentNodeId != 0 {
			childrenByParent[node.ParentNodeId] = append(childrenByParent[node.ParentNodeId], node)
		}
	}

	seeds := make([]*vo.ChunkCandidate, 0)
	children := childrenByParent[parentSeed.StructureNodeId]

	for _, child := range children {
		if strutil.IsBlank(child.ContentText) {
			continue
		}

		// 只处理 SECTION、STEP、LIST_ITEM 类型的节点
		if child.NodeType == vo.NodeTypeSection || child.NodeType == vo.NodeTypeStep || child.NodeType == vo.NodeTypeListItem {
			seeds = append(seeds, s.newChunkCandidate(child, vo.ChunkSourceTypeOriginal))
		}
	}

	if len(seeds) > 0 {
		return seeds
	}

	// 如果没有找到合适的子节点，克隆父种子
	clonedSeed := s.cloneChunkCandidate(parentSeed, parentSeed.Text)
	return []*vo.ChunkCandidate{clonedSeed}
}

// isContentBearingSection 判断是否为承载内容的章节
func (s *StrategyLogicImpl) isContentBearingSection(node *entity.DocumentStructureNode, hasChildSection bool) bool {
	if strutil.IsBlank(node.ContentText) {
		return false
	}

	// 如果没有子章节，则认为该章节有内容
	if !hasChildSection {
		return true
	}

	headingText := strutil.Trim(utils.BlankToDefault(node.AnchorText, node.Title))
	content := strutil.Trim(node.ContentText)

	// 如果内容等于标题，说明该章节没有独立内容
	if content == headingText {
		return false
	}

	// 内容长度大于标题长度+16字符，或者包含换行符，则认为有独立内容
	return utf8.RuneCountInString(content) > utf8.RuneCountInString(headingText)+16 || strings.Contains(content, "\n")
}

// cloneChunkCandidate 克隆块候选
func (s *StrategyLogicImpl) cloneChunkCandidate(original *vo.ChunkCandidate, text string) *vo.ChunkCandidate {
	if original == nil {
		return &vo.ChunkCandidate{
			Text:       text,
			SourceType: vo.ChunkSourceTypeOriginal,
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

// cloneParentBlockCandidate 克隆父块候选
func (s *StrategyLogicImpl) cloneParentBlockCandidate(source *vo.ParentBlockCandidate, childChunks []*vo.ChunkCandidate, text string) *vo.ParentBlockCandidate {
	if source == nil {
		return &vo.ParentBlockCandidate{
			Text:        text,
			SourceType:  vo.ChunkSourceTypeOriginal,
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

// executePipeline 执行流水线，按步骤顺序将种子列表传递给各切块策略，产生最终的块候选列表
// func (s *StrategyLogicImpl) executePipeline(ctx context.Context, inputSeeds []*vo.ChunkCandidate, steps []*entity.DocumentStrategyStep, pipelineType string) []*vo.ChunkCandidate {
// 	currentChunks := s.cleanupChunkList(inputSeeds)
// 	if len(currentChunks) == 0 {
// 		return currentChunks
// 	}
//
// 	// 按步骤顺序调用 Registry 中的策略
// 	for _, step := range steps {
// 		strategy, ok := s.registry[step.StrategyType]
// 		if !ok {
// 			continue
// 		}
// 		// 将当前块候选转换为 chunk.TextBlock
// 		outputList := make([]*chunk.Output, 0, len(currentChunks))
// 		var outs []*chunk.Output
// 		var err error
// 		for _, c := range currentChunks {
// 			if c == nil || strutil.IsBlank(c.Text) {
// 				continue
// 			}
//
// 			if err != nil {
// 				// 单个策略失败不中断，降级为保持原样
// 				outputList = append(outputList, &chunk.Output{
// 					Text:        strings.TrimSpace(c.Text),
// 					SectionPath: c.SectionPath,
// 					SourceType:  c.SourceType,
// 				})
// 				continue
// 			}
// 			outputList = append(outputList, outs...)
// 		}
//
// 		// 转换回 vo.ChunkCandidate
// 		nextChunks := make([]*vo.ChunkCandidate, 0, len(outputList))
// 		for _, out := range outputList {
// 			if out == nil || strutil.IsBlank(out.Text) {
// 				continue
// 			}
// 			nextChunks = append(nextChunks, &vo.ChunkCandidate{
// 				SectionPath: out.SectionPath,
// 				Text:        strings.TrimSpace(out.Text),
// 				SourceType:  out.SourceType,
// 			})
// 		}
// 		currentChunks = s.cleanupChunkList(nextChunks)
// 	}
// 	return s.cleanupChunkList(currentChunks)
// }

// executePipeline 执行流水线，按步骤顺序将种子列表传递给各切块策略，产生最终的块候选列表
func (s *StrategyLogicImpl) executePipeline(ctx context.Context, inputSeeds []*vo.ChunkCandidate, steps []*entity.DocumentStrategyStep, pipelineType string) []*vo.ChunkCandidate {
	currentChunks := s.cleanupChunkList(inputSeeds)
	if len(currentChunks) == 0 {
		return currentChunks
	}

	// 按步骤顺序调用 Registry 中的策略
	for _, step := range steps {
		strategy, ok := s.registry[step.StrategyType]
		if !ok {
			continue
		}
		// 将当前块候选转换为 chunk.TextBlock

		var err error
		for _, c := range currentChunks {
			if c == nil || strutil.IsBlank(c.Text) {
				continue
			}
			switch step.SourceType {
			case vo.StrategyTypeStructure:
				currentChunks = s.applyStructureChunking(ctx, currentChunks, pipelineType)
			}
		}
		currentChunks = s.cleanupChunkList(currentChunks)
	}
	return s.cleanupChunkList(currentChunks)
}

// cleanupChunkList 清理块列表
func (s *StrategyLogicImpl) cleanupChunkList(chunks []*vo.ChunkCandidate) []*vo.ChunkCandidate {
	result := make(map[string]*vo.ChunkCandidate)
	for _, candidate := range chunks {
		if candidate != nil && strutil.IsNotBlank(candidate.Text) {
			path := utils.BlankToDefault(candidate.CanonicalPath, candidate.SectionPath)
			trim := strutil.Trim(candidate.Text)
			uniqueKey := fmt.Sprintf("%s||%d||%s", path, candidate.ItemIndex, trim)
			if _, ok := result[uniqueKey]; !ok {
				result[uniqueKey] = s.cloneChunkCandidate(candidate, trim)
			}
		}
	}
	return maputil.Values(result)
}

// cleanupParentBlockList 清理父块列表
func (s *StrategyLogicImpl) cleanupParentBlockList(blocks []*vo.ParentBlockCandidate) []*vo.ParentBlockCandidate {
	result := make(map[string]*vo.ParentBlockCandidate)
	for _, block := range blocks {
		if block != nil && strutil.IsNotBlank(block.Text) {
			path := utils.BlankToDefault(block.CanonicalPath, block.SectionPath)
			trim := strutil.Trim(block.Text)
			uniqueKey := fmt.Sprintf("%s||%d||%s", path, block.ItemIndex, trim)
			if _, ok := result[uniqueKey]; !ok {
				result[uniqueKey] = s.cloneParentBlockCandidate(block, block.ChildChunks, trim)
			}
		}
	}
	return maputil.Values(result)
}

// ---------------- 结构切块 ----------------

// applyStructureChunking 对候选列表应用结构切块（按标题行分段）
func (s *StrategyLogicImpl) applyStructureChunking(ctx context.Context, sourceList []*vo.ChunkCandidate, pipelineType string) []*vo.ChunkCandidate {
	resultList := make([]*vo.ChunkCandidate, 0)
	for _, candidate := range sourceList {
		if candidate == nil || strutil.IsBlank(candidate.Text) {
			continue
		}
		strategy := s.registry[vo.StrategyTypeStructure]
		outputs, _ := strategy.Chunk(ctx, &chunk.TextBlock{
			SectionPath:   candidate.SectionPath,
			CanonicalPath: candidate.CanonicalPath,
			ItemIndex:     candidate.ItemIndex,
			Text:          candidate.Text,
			SourceType:    candidate.SourceType,
		})
		if len(outputs) == 0 {

		}
		// 转换为 vo.ChunkCandidate
		for _, out := range outputs {
			if out == nil || strutil.IsBlank(out.Text) {
				continue
			}
			resultList = append(resultList, &vo.ChunkCandidate{
				SectionPath: out.SectionPath,
				Text:        strings.TrimSpace(out.Text),
				SourceType:  out.SourceType,
			})
		}
	}

	return resultList
}

// ---------------- 递归切块 ----------------

// applyRecursiveChunking 对候选列表应用递归切块
func (s *StrategyLogicImpl) applyRecursiveChunking(sourceList []*vo.ChunkCandidate, pipelineType string) []*vo.ChunkCandidate {
	resultList := make([]*vo.ChunkCandidate, 0)
	maxChars := utils.Ternary(pipelineType == vo.PipelineTypeParent, ParentBlockMaxChars, s.recursiveMaxChars)
	overlapChars := s.resolveRecursiveOverlap(maxChars, pipelineType)
	for _, candidate := range sourceList {
		if candidate == nil || strutil.IsBlank(candidate.Text) {
			continue
		}
		splitTexts := s.recursiveSplit(candidate.Text, maxChars, overlapChars)
		for _, splitText := range splitTexts {
			resultList = append(resultList, s.cloneChunkCandidate(candidate, splitText))
		}
	}
	return resultList
}

// resolveRecursiveOverlap 解析递归切块重叠字符数（父块使用较大常量兜底）
func (s *StrategyLogicImpl) resolveRecursiveOverlap(maxChars int, pipelineType string) int {
	if pipelineType == vo.PipelineTypeParent {
		return min(ParentBlockOverlapChars, max(0, maxChars-1))
	}
	if s.recursiveOverlapChars <= 0 {
		return 0
	}
	return min(s.recursiveOverlapChars, max(0, maxChars-1))
}

// ---------------- 语义切块 ----------------

// applySemanticChunking 对候选列表应用基于 Jaccard 相似度的语义切块
func (s *StrategyLogicImpl) applySemanticChunking(_ context.Context, sourceList []*vo.ChunkCandidate, pipelineType string) []*vo.ChunkCandidate {
	resultList := make([]*vo.ChunkCandidate, 0)
	semanticMinChars := s.resolveSemanticMinChars(pipelineType)
	for _, candidate := range sourceList {
		if candidate == nil || strutil.IsBlank(candidate.Text) {
			continue
		}
		// 文本较短时保持原样，避免过碎
		if utf8.RuneCountInString(candidate.Text) <= semanticMinChars {
			resultList = append(resultList, candidate)
			continue
		}
		resultList = append(resultList, s.semanticSplit(candidate, pipelineType)...)
	}
	return resultList
}

// semanticSplit 按句子+相似度阈值切分单段文本
func (s *StrategyLogicImpl) semanticSplit(candidate *vo.ChunkCandidate, pipelineType string) []*vo.ChunkCandidate {
	resultList := make([]*vo.ChunkCandidate, 0)
	sentenceList := s.splitSentences(candidate.Text)
	if len(sentenceList) <= 1 {
		resultList = append(resultList, candidate)
		return resultList
	}

	currentChunk := strings.Builder{}
	currentTokenSet := make(map[string]bool) // 当前累计块的词元集合
	semanticMinChars := s.resolveSemanticMinChars(pipelineType)
	semanticMaxChars := s.resolveSemanticMaxChars(pipelineType)

	for _, sentence := range sentenceList {
		sentenceTokenSet := s.extractTokens(sentence)

		currentLen := utf8.RuneCountInString(currentChunk.String())
		sentenceLen := utf8.RuneCountInString(sentence)
		exceedMaxChars := currentLen+sentenceLen > semanticMaxChars
		var similarity float64
		if len(currentTokenSet) == 0 {
			similarity = 1.0
		} else {
			similarity = s.jaccard(currentTokenSet, sentenceTokenSet)
		}
		semanticBreak := currentLen >= semanticMinChars && similarity < s.semanticSimilarityThreshold

		// 达到上限或出现语义断层则切出当前块
		if currentLen > 0 && (exceedMaxChars || semanticBreak) {
			resultList = append(resultList, s.cloneChunkCandidate(candidate, strutil.Trim(currentChunk.String())))
			currentChunk.Reset()
			currentTokenSet = make(map[string]bool)
		}

		currentChunk.WriteString(sentence)
		for token := range sentenceTokenSet {
			currentTokenSet[token] = true
		}
	}

	if currentChunk.Len() > 0 {
		resultList = append(resultList, s.cloneChunkCandidate(candidate, strutil.Trim(currentChunk.String())))
	}
	return resultList
}

// resolveSemanticMaxChars 解析语义切块最大字符数（父块使用较大常量兜底）
func (s *StrategyLogicImpl) resolveSemanticMaxChars(pipelineType string) int {
	return utils.Ternary(pipelineType == vo.PipelineTypeParent, max(ParentSemanticMaxChars, s.semanticMaxChars), s.semanticMaxChars)
}

// resolveSemanticMinChars 解析语义切块最小字符数
func (s *StrategyLogicImpl) resolveSemanticMinChars(pipelineType string) int {
	return utils.Ternary(pipelineType == vo.PipelineTypeParent, max(ParentSemanticMinChars, s.semanticMinChars), s.semanticMinChars)
}

// ---------------- 大模型智能切块 ----------------

// applyLlmChunking 使用大模型智能切分；不可用时回退到语义切块
func (s *StrategyLogicImpl) applyLlmChunking(ctx context.Context, sourceList []*vo.ChunkCandidate, pipelineType string) []*vo.ChunkCandidate {
	if !s.llmEnabled {
		return s.applySemanticChunking(ctx, sourceList, pipelineType)
	}

	resultList := make([]*vo.ChunkCandidate, 0)
	llmMaxChars := s.resolveLlmMaxChars(pipelineType)

	for _, candidate := range sourceList {
		if candidate == nil || strutil.IsBlank(candidate.Text) {
			continue
		}

		// 过长文本先按递归切块分片，再逐个调用大模型
		var sourceTextList []string
		if utf8.RuneCountInString(candidate.Text) > llmMaxChars {
			sourceTextList = s.recursiveSplit(candidate.Text, llmMaxChars, 0)
		} else {
			sourceTextList = []string{candidate.Text}
		}

		for _, sourceText := range sourceTextList {
			llmChunks, err := s.registry[chunkllm.Name].Chunk(ctx)
			if err != nil {
				Warnf("大模型智能切块失败，回退到语义切块，err=%v", err)
				return nil
			}
			if len(llmChunks) == 0 {
				// 大模型失败，回退语义切分
				fallback := s.semanticSplit(s.cloneChunkCandidate(candidate, sourceText), pipelineType)
				resultList = append(resultList, fallback...)
				continue
			}
			for _, chunk := range llmChunks {
				resultList = append(resultList, s.cloneChunkCandidate(candidate, chunk))
			}
		}
	}
	return resultList
}

func (s *StrategyLogicImpl) newChunkCandidate(node *entity.DocumentStructureNode, sourceType int) *vo.ChunkCandidate {
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

// // resolveMaxChars 根据流水线类型返回块大小
// func (r *RecursiveStrategy) resolveMaxChars(pipelineType PipelineType) int {
// 	if pipelineType == PipelineTypeParent {
// 		return parentBlockMaxChars
// 	}
// 	return r.base.recursiveMaxChars
// }
//
// // resolveOverlapChars 根据流水线类型返回重叠字符数
// func (r *RecursiveStrategy) resolveOverlapChars(maxChars int, pipelineType PipelineType) int {
// 	if pipelineType == PipelineTypeParent {
// 		return min(parentBlockOverlapChars, max(0, maxChars-1))
// 	}
// 	if r.base.recursiveOverlapChars <= 0 {
// 		return 0
// 	}
// 	return min(r.base.recursiveOverlapChars, max(0, maxChars-1))
// }

// func (s *Strategy) Chunk(ctx context.Context, input *chunk.TextBlock, opts ...chunk.Option) ([]*chunk.Output, error) {
// 	// 未启用或未配置大模型：直接降级为语义分块
// 	if !s.opt.enabled || s.model == nil {
// 		fallback := chunkSemantic.NewStrategy(
// 			chunkSemantic.WithMinChars(240),
// 			chunkSemantic.WithMaxChars(700),
// 			chunkSemantic.WithSimilarityThreshold(0.18),
// 		)
// 		return fallback.Chunk(ctx, input)
// 	}
//
// 	if input == nil || strings.TrimSpace(input.Text) == "" {
// 		return nil, nil
// 	}
// 	opt := chunk.GetChunkImplSpecificOptions(s.opt, opts...)
//
// 	// 对超长文本先做递归分块，再逐个调用大模型
// 	var sourceTextList []string
// 	if utf8.RuneCountInString(input.Text) > opt.maxChars {
// 		recursiveStrategy := chunkRecursive.NewStrategy(
// 			chunkRecursive.WithMaxChars(opt.maxChars),
// 			chunkRecursive.WithOverlapChars(0),
// 		)
// 		rawChunks, err := recursiveStrategy.Chunk(ctx, input)
// 		if err != nil {
// 			return nil, err
// 		}
// 		sourceTextList = make([]string, 0, len(rawChunks))
// 		for _, c := range rawChunks {
// 			if c == nil {
// 				continue
// 			}
// 			if t := strings.TrimSpace(c.Text); t != "" {
// 				sourceTextList = append(sourceTextList, t)
// 			}
// 		}
// 	} else {
// 		sourceTextList = []string{strings.TrimSpace(input.Text)}
// 	}
//
// 	resultList := make([]*chunk.Output, 0, len(sourceTextList))
// 	for _, sourceText := range sourceTextList {
// 		chunks := s.split(ctx, sourceText)
// 		// 大模型调用失败：降级为语义分块
// 		if len(chunks) == 0 {
// 			fallback := chunkSemantic.NewStrategy(
// 				chunkSemantic.WithMinChars(240),
// 				chunkSemantic.WithMaxChars(700),
// 				chunkSemantic.WithSimilarityThreshold(0.18),
// 			)
// 			fallbackInput := &chunk.TextBlock{
// 				SectionPath:   input.SectionPath,
// 				CanonicalPath: input.CanonicalPath,
// 				ItemIndex:     input.ItemIndex,
// 				Text:          sourceText,
// 				SourceType:    input.SourceType,
// 			}
// 			fallbackChunks, err := fallback.Chunk(ctx, fallbackInput)
// 			if err != nil {
// 				return nil, err
// 			}
// 			resultList = append(resultList, fallbackChunks...)
// 			continue
// 		}
// 		for _, chunkText := range chunks {
// 			trimmed := strings.TrimSpace(chunkText)
// 			if trimmed == "" {
// 				continue
// 			}
// 			resultList = append(resultList, &chunk.Output{
// 				SectionPath:   strings.TrimSpace(input.SectionPath),
// 				CanonicalPath: strings.TrimSpace(input.CanonicalPath),
// 				ItemIndex:     input.ItemIndex,
// 				Text:          trimmed,
// 				SourceType:    input.SourceType,
// 			})
// 		}
// 	}
// 	return resultList, nil
// }
