package logic

import (
	"context"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/cloudwego/eino/schema"
	"github.com/duke-git/lancet/v2/slice"
	"github.com/duke-git/lancet/v2/strutil"

	"github.com/swiftbit/know-agent/common/utils"
	chatlogic "github.com/swiftbit/know-agent/internal/domain/chat/logic"
	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/document/model/vo"
	"github.com/swiftbit/know-agent/internal/domain/document/support"
	errorx "github.com/swiftbit/know-agent/internal/error"
	"github.com/swiftbit/know-agent/internal/svc"
)

var (
	englishWordPattern2 = regexp.MustCompile(`[A-Za-z0-9]{2,}`) // 英文单词正则
)

const (
	ParentBlockMaxChars     = 2200 // 父块最大字符数
	ParentBlockOverlapChars = 180  // 父块重叠字符数
	ParentSemanticMaxChars  = 1600 // 语义块最大字符数
	ParentSemanticMinChars  = 480  // 语义块最小字符数
)

var (
	englishWordPattern = regexp.MustCompile("[A-Za-z0-9]{2,}") // 英文单词正则表达式
)

// StrategyLogicImpl 策略业务逻辑实现
type StrategyLogicImpl struct {
	chatModel      chatlogic.ObservedChatModelImpl[*schema.Message]
	promptTemplate chatlogic.PromptTemplateLogic
	structureNode  StructureNodeLogic
	support.DocumentLineClassifier
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

func NewStrategyLogic(svcCtx *svc.ServiceContext, chatModel chatlogic.ObservedChatModelImpl[*schema.Message],
	promptTemplate chatlogic.PromptTemplateLogic, structureNode StructureNodeLogic) StrategyLogic {
	return &StrategyLogicImpl{
		chatModel:      chatModel,
		promptTemplate: promptTemplate,
		structureNode:  structureNode,
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

// RecommendStrategy 推荐策略方案
// 根据文档分析结果推荐最优的父块和子块策略组合
func (s *StrategyLogicImpl) RecommendStrategy(ctx context.Context, document *entity.Document, analysisResult *vo.DocumentAnalysisResult) (*vo.DocumentStrategyPlanDraft, error) {
	if document == nil || analysisResult == nil {
		return nil, newStrategyError("文档或分析结果不能为空")
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
		childStrategyTypes = append(childStrategyTypes, vo.StrategyTypeLlm)
		childReasonMap[vo.StrategyTypeLlm] = "文档质量偏低或结构识别不稳定，子块先使用大模型智能切块增强复杂场景。"
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
	strategySnapshot := s.buildCombinedStrategySnapshot(parentSteps, childSteps)

	return &vo.DocumentStrategyPlanDraft{
		ParentSteps:      parentSteps,
		ChildSteps:       childSteps,
		StrategySnapshot: strategySnapshot,
		RecommendReason:  strings.Join(reasonList, "；"),
	}, nil
}

// NormalizeSteps 标准化策略步骤
// 将用户提交的策略类型标准化为可执行的步骤列表
func (s *StrategyLogicImpl) NormalizeSteps(ctx context.Context, basePlan *entity.DocumentStrategyPlan, baseSteps []*entity.DocumentStrategyStep,
	requestParentStrategyTypes []int, requestChildStrategyTypes []int, documentId int64) ([]*entity.DocumentStrategyStep, error) {

	// 标准化请求的策略类型
	normalizedParentTypes := s.normalizePipelineTypes(requestParentStrategyTypes)
	normalizedChildTypes := s.normalizePipelineTypes(requestChildStrategyTypes)

	// 构建基础步骤映射
	baseStepMap := make(map[int]map[int]*entity.DocumentStrategyStep)
	for _, baseStep := range baseSteps {
		pipelineType := baseStep.PipelineType
		if pipelineType == 0 {
			pipelineType = vo.PipelineTypeChild
		}

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

// BuildParentBlocks 构建父子块结构
// 步骤：排序父/子流水线步骤 -> 通过 StructureNodeLogic 拉取结构节点 ->
//
//	为每个父块种子执行父流水线切块 -> 为每个父块执行子流水线切块 ->
//	组装为 Parent-Child 结构并去重。
func (s *StrategyLogicImpl) BuildParentBlocks(ctx context.Context, document *entity.Document, plan *entity.DocumentStrategyPlan,
	steps []*entity.DocumentStrategyStep, parsedText string) ([]*vo.ParentBlockCandidate, error) {

	if plan == nil || parsedText == "" {
		return nil, newStrategyError("方案或解析文本不能为空")
	}

	parentSteps := s.sortPipelineSteps(steps, vo.PipelineTypeParent)
	childSteps := s.sortPipelineSteps(steps, vo.PipelineTypeChild)

	if len(parentSteps) == 0 {
		return nil, errorx.ErrParentBlockMissing
	}
	if len(childSteps) == 0 {
		return nil, errorx.ErrChildBlockMissing
	}

	// 通过 StructureNodeLogic 拉取文档解析对应的结构节点（documentId 为空时退化为空）
	var structureNodes []*entity.DocumentStructureNode
	if document != nil && document.ID > 0 {
		parseTaskId := int64(0)
		if document.LastParseTaskId > 0 {
			parseTaskId = document.LastParseTaskId
		}
		list, err := s.structureNode.ListDocumentNodes(ctx, document.ID, parseTaskId)
		if err == nil {
			structureNodes = list
		}
	}

	// 为文档整体生成父块种子；在 seeds 不为空时进一步跑父流水线的剩余步骤
	parentSeedList := s.buildParentSeedList(parsedText, parentSteps, structureNodes)

	parentBlockList := make([]*vo.ParentBlockCandidate, 0, len(parentSeedList))
	for _, parentSeed := range s.cleanupChunkList(parentSeedList) {
		if parentSeed == nil || strutil.IsBlank(parentSeed.Text) {
			continue
		}
		childSeedList := s.buildChildSeedList(parentSeed, childSteps, structureNodes)
		finalChildren := s.cleanupChunkList(childSeedList)
		if len(finalChildren) == 0 {
			finalChildren = []*vo.ChunkCandidate{s.cloneChunkCandidate(parentSeed, strings.TrimSpace(parentSeed.Text))}
		}
		parentBlockList = append(parentBlockList, &vo.ParentBlockCandidate{
			SectionPath: parentSeed.SectionPath,
			NodeId:      parentSeed.StructureNodeId,
			NodeType:    parentSeed.StructureNodeType,
			Text:        strings.TrimSpace(parentSeed.Text),
			SourceType:  parentSeed.SourceType,
			ChildChunks: finalChildren,
		})
	}
	return s.cleanupParentBlockList(parentBlockList), nil
}

// shouldUseStructure 判断是否应该使用结构切块
func (s *StrategyLogicImpl) shouldUseStructure(document *entity.Document, analysisResult *vo.DocumentAnalysisResult) bool {
	return analysisResult.StructureLevel >= vo.StructureLevelMedium
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

// buildDraftSteps 构建步骤草稿
func (s *StrategyLogicImpl) buildDraftSteps(pipelineType string, strategyTypes []int, reasonMap map[int]string) []*vo.DocumentStrategyStepDraft {
	draftList := make([]*vo.DocumentStrategyStepDraft, 0, len(strategyTypes))
	for index, strategyType := range strategyTypes {
		draftList = append(draftList, &vo.DocumentStrategyStepDraft{
			PipelineType:    pipelineType,
			StrategyType:    strategyType,
			StrategyRole:    s.resolveRole(index, strategyType),
			SourceType:      vo.StrategySourceTypeSystemRecommend,
			RecommendReason: utils.BlankToDefault(reasonMap[strategyType], "系统为当前流水线生成的推荐步骤。"),
		})
	}
	return draftList
}

// normalizePipelineTypes 标准化流水线类型
func (s *StrategyLogicImpl) normalizePipelineTypes(requestStrategyTypes []int) []int {
	// 去重并保持顺序
	typeSet := make(map[int]bool)
	normalizedTypes := make([]int, 0)

	for _, strategyType := range requestStrategyTypes {
		if s.isValidStrategyType(strategyType) && !typeSet[strategyType] {
			typeSet[strategyType] = true
			normalizedTypes = append(normalizedTypes, strategyType)
		}
	}

	return normalizedTypes
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

// buildParentSeedList 构建父块种子列表
func (s *StrategyLogicImpl) buildParentSeedList(parsedText string, parentSteps []*entity.DocumentStrategyStep,
	structureNodes []*entity.DocumentStructureNode) []*vo.ChunkCandidate {

	if s.containsStructureStep(parentSteps) && len(structureNodes) > 0 {
		// 如果有结构步骤且存在结构节点，优先使用结构切块
		structureSeeds := s.buildStructureParentSeeds(structureNodes)
		if len(structureSeeds) == 0 {
			// 结构种子为空，使用原始文本
			originalSeed := &vo.ChunkCandidate{
				Text:       parsedText,
				SourceType: vo.StrategySourceTypeSystemRecommend,
			}
			return s.executePipeline([]*vo.ChunkCandidate{originalSeed}, parentSteps, vo.PipelineTypeParent)
		}

		remainingSteps := s.stripStructureSteps(parentSteps)
		if len(remainingSteps) == 0 {
			return structureSeeds
		}

		return s.executePipeline(structureSeeds, remainingSteps, vo.PipelineTypeParent)
	}

	// 没有结构步骤或结构节点，直接使用原始文本
	originalSeed := &vo.ChunkCandidate{
		Text:       parsedText,
		SourceType: vo.StrategySourceTypeSystemRecommend,
	}
	return s.executePipeline([]*vo.ChunkCandidate{originalSeed}, parentSteps, vo.PipelineTypeParent)
}

// buildChildSeedList 构建子块种子列表
func (s *StrategyLogicImpl) buildChildSeedList(parentSeed *vo.ChunkCandidate, childSteps []*entity.DocumentStrategyStep,
	structureNodes []*entity.DocumentStructureNode) []*vo.ChunkCandidate {

	if s.containsStructureStep(childSteps) && parentSeed.NodeId != 0 && len(structureNodes) > 0 {
		// 有结构步骤且父种子有节点ID，使用结构切块
		structureSeeds := s.buildStructureChildSeeds(parentSeed, structureNodes)
		remainingSteps := s.stripStructureSteps(childSteps)

		if len(remainingSteps) == 0 {
			return structureSeeds
		}

		return s.executePipeline(structureSeeds, remainingSteps, vo.PipelineTypeChild)
	}

	// 直接克隆父种子
	clonedSeed := s.cloneChunkCandidate(parentSeed, parentSeed.Text)
	return s.executePipeline([]*vo.ChunkCandidate{clonedSeed}, childSteps, vo.PipelineTypeChild)
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
			seeds = append(seeds, &vo.ChunkCandidate{
				SectionPath:       node.SectionPath,
				StructureNodeId:   node.ID,
				StructureNodeType: node.NodeType,
				CanonicalPath:     node.SectionPath,
				ItemIndex:         node.ItemIndex,
				Text:              node.ContentText,
				SourceType:        vo.ChunkSourceTypeOriginal,
			})
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

		// 只处理SECTION、STEP、LIST_ITEM类型的节点
		if child.NodeType == vo.NodeTypeSection || child.NodeType == vo.NodeTypeStep || child.NodeType == vo.NodeTypeListItem {
			seeds = append(seeds, &vo.ChunkCandidate{
				SectionPath:       child.SectionPath,
				StructureNodeId:   child.ID,
				StructureNodeType: child.NodeType,
				CanonicalPath:     child.SectionPath,
				ItemIndex:         child.ItemIndex,
				Text:              child.ContentText,
				SourceType:        vo.ChunkSourceTypeOriginal,
			})

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

// executePipeline 执行流水线
// TODO: 这里需要实现具体的切块逻辑，包括递归切块、语义切块等
func (s *StrategyLogicImpl) executePipeline(inputSeeds []*vo.ChunkCandidate, steps []*entity.DocumentStrategyStep, pipelineType string) []*vo.ChunkCandidate {
	// 简化实现：暂时直接返回输入
	// 实际实现需要根据不同的策略类型调用对应的切块器
	result := make([]*vo.ChunkCandidate, len(inputSeeds))
	copy(result, inputSeeds)
	return result
}

// cleanupChunkList 清理块列表
func (s *StrategyLogicImpl) cleanupChunkList(chunks []*vo.ChunkCandidate) []*vo.ChunkCandidate {
	result := make([]*vo.ChunkCandidate, 0)
	for _, chunk := range chunks {
		if chunk != nil && strutil.IsNotBlank(chunk.Text) {
			result = append(result, chunk)
		}
	}
	return result
}

// cleanupParentBlockList 清理父块列表
func (s *StrategyLogicImpl) cleanupParentBlockList(blocks []*vo.ParentBlockCandidate) []*vo.ParentBlockCandidate {
	result := make([]*vo.ParentBlockCandidate, 0)
	for _, block := range blocks {
		if block != nil && strutil.IsNotBlank(block.Text) && len(block.ChildChunks) > 0 {
			result = append(result, block)
		}
	}
	return result
}

// sortPipelineSteps 按流水线类型排序步骤
func (s *StrategyLogicImpl) sortPipelineSteps(steps []*entity.DocumentStrategyStep, pipelineType string) []*entity.DocumentStrategyStep {
	filtered := slice.Filter(steps, func(index int, item *entity.DocumentStrategyStep) bool {
		return utils.EqualsIgnoreCase(pipelineType, utils.BlankToDefault(item.PipelineType, vo.PipelineTypeChild))
	})
	slice.SortBy(filtered, func(a, b *entity.DocumentStrategyStep) bool { return a.StepNo < b.StepNo })
	return filtered
}

// buildCombinedStrategySnapshot 构建组合策略快照
func (s *StrategyLogicImpl) buildCombinedStrategySnapshot(parentSteps []*vo.DocumentStrategyStepDraft, childSteps []*vo.DocumentStrategyStepDraft) string {
	parentTypes := make([]string, len(parentSteps))
	childTypes := make([]string, len(childSteps))

	for i, step := range parentSteps {
		parentTypes[i] = s.getStrategyTypeName(step.StrategyType)
	}
	for i, step := range childSteps {
		childTypes[i] = s.getStrategyTypeName(step.StrategyType)
	}

	return "PARENT:" + strings.Join(parentTypes, ",") + ";CHILD:" + strings.Join(childTypes, ",")
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
	if strategyType == vo.StrategyTypeLlm {
		return vo.StrategyTypeRecursive
	}
	return vo.StrategyRoleOptimize
}

// getStrategyTypeName 获取策略类型名称
func (s *StrategyLogicImpl) getStrategyTypeName(strategyType int) string {
	switch strategyType {
	case vo.StrategyTypeStructure:
		return "STRUCTURE"
	case vo.StrategyTypeRecursive:
		return "RECURSIVE"
	case vo.StrategyTypeSemantic:
		return "SEMANTIC"
	case vo.StrategyTypeLlm:
		return "LLM"
	case vo.StrategyTypeMarkdown:
		return "MARKDOWN"
	default:
		return "UNKNOWN"
	}
}

// isValidStrategyType 判断策略类型是否有效
func (s *StrategyLogicImpl) isValidStrategyType(strategyType int) bool {
	return strategyType >= vo.StrategyTypeStructure && strategyType <= vo.StrategyTypeMarkdown
}

// getDefaultReason 获取默认推荐理由
func (s *StrategyLogicImpl) getDefaultReason(strategyType int) string {
	switch strategyType {
	case vo.StrategyTypeStructure:
		return "基于文档结构的切块策略。"
	case vo.StrategyTypeRecursive:
		return "递归切块策略。"
	case vo.StrategyTypeSemantic:
		return "语义切块策略。"
	case vo.StrategyTypeLlm:
		return "大模型智能切块策略。"
	case vo.StrategyTypeMarkdown:
		return "Markdown格式切块策略。"
	default:
		return "系统为当前流水线生成的推荐步骤。"
	}
}
