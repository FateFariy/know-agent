package logic

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/cloudwego/eino/schema"
	"github.com/duke-git/lancet/v2/maputil"
	"github.com/duke-git/lancet/v2/slice"
	"github.com/duke-git/lancet/v2/strutil"
	"github.com/zeromicro/go-zero/core/logx"

	"github.com/swiftbit/know-agent/common"
	"github.com/swiftbit/know-agent/common/utils"
	chatlogic "github.com/swiftbit/know-agent/internal/domain/chat/logic"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/prompt"
	"github.com/swiftbit/know-agent/internal/domain/document/logic/chunk"
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

var (
	englishWordPattern = regexp.MustCompile("[A-Za-z0-9]{2,}") // 英文单词正则表达式
	paragraphSplitRe   = regexp.MustCompile(`\n\s*\n`)         // 段落分隔符
	lineSplitRe        = regexp.MustCompile(`\n`)              // 换行分隔符
	sentenceSplitRe    = regexp.MustCompile(`[。！？!?;；.]`)      // 句子分隔符
	chineseCharRe      = regexp.MustCompile(`[\u4e00-\u9fa5]`) // 中文字符正则
)

// StrategyLogicImpl 策略业务逻辑实现
// 核心的分块策略由 chunk 包的 Registry 管理，支持动态注册和组合
type StrategyLogicImpl struct {
	chatModel        chatlogic.ObservedChatModelImpl[*schema.Message]
	promptTemplate   chatlogic.PromptTemplateLogic
	structureNode    StructureNodeLogic
	strategyRegistry *chunk.Registry
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

	// 构建策略注册表：默认内置 结构/递归/语义 三种策略，大模型策略根据配置注入
	registry := chunk.NewRegistry()

	// 覆盖默认的递归/语义策略，使其使用应用配置
	registry.RegisterOverride(chunk.NewRecursiveStrategy(
		chunk.WithRecursive(svcCtx.Config.Chunk.RecursiveMaxChars, svcCtx.Config.Chunk.RecursiveOverlapChars),
	))
	registry.RegisterOverride(chunk.NewSemanticStrategy(
		chunk.WithSemantic(svcCtx.Config.Chunk.SemanticMinChars, svcCtx.Config.Chunk.SemanticMaxChars, svcCtx.Config.Chunk.SemanticSimilarityThreshold),
	))

	// 大模型策略需要业务层注入 chatModel 和 promptTemplate；注册失败静默降级为语义切块
	llmModel := newLLMModelAdapter(&chatModel)
	llmRenderer := newLLMRenderer(promptTemplate)
	_ = registry.Register(chunk.NewLLMStrategy(
		llmModel,
		llmRenderer,
		chunk.WithLLM(svcCtx.Config.Chunk.LlmEnabled, svcCtx.Config.Chunk.LlmMaxChars),
	))

	return &StrategyLogicImpl{
		chatModel:        chatModel,
		promptTemplate:   promptTemplate,
		structureNode:    structureNode,
		strategyRegistry: registry,
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

// Registry 返回底层的策略注册表，便于单元测试动态替换策略
func (s *StrategyLogicImpl) Registry() *chunk.Registry {
	return s.strategyRegistry
}

// llmModelAdapter 将业务层 chatModel 适配为 chunk 包的 LLMModel 接口
type llmModelAdapter struct {
	model *chatlogic.ObservedChatModelImpl[*schema.Message]
}

func newLLMModelAdapter(model *chatlogic.ObservedChatModelImpl[*schema.Message]) chunk.LLMModel {
	return &llmModelAdapter{model: model}
}

func (a *llmModelAdapter) Generate(ctx context.Context, prompt string) (string, error) {
	return a.model.Generate(ctx, "", prompt)
}

// llmRendererAdapter 将业务层 promptTemplate 适配为 chunk 包的 LLMPromptRenderer 接口
type llmRendererAdapter struct {
	template chatlogic.PromptTemplateLogic
}

func newLLMRenderer(template chatlogic.PromptTemplateLogic) chunk.LLMPromptRenderer {
	return &llmRendererAdapter{template: template}
}

func (r *llmRendererAdapter) Render(ctx context.Context, sourceText string) (string, error) {
	return r.template.Render(prompt.DocumentLlmSplit, map[string]any{"sourceText": sourceText})
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
		if pipelineType == "" {
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
// 根据策略方案执行父块和子块的切分，构建 Parent-Child 结构
func (s *StrategyLogicImpl) BuildParentBlocks(ctx context.Context, document *entity.Document, plan *entity.DocumentStrategyPlan,
	steps []*entity.DocumentStrategyStep, parsedText string) ([]*vo.ParentBlockCandidate, error) {

	if plan == nil || parsedText == "" {
		return nil, newStrategyError("方案或解析文本不能为空")
	}

	// 按流水线类型排序步骤
	parentSteps := s.sortPipelineSteps(steps, vo.PipelineTypeParent)
	childSteps := s.sortPipelineSteps(steps, vo.PipelineTypeChild)

	if len(parentSteps) == 0 {
		return nil, errorx.ErrParentBlockMissing
	}
	if len(childSteps) == 0 {
		return nil, errorx.ErrChildBlockMissing
	}

	// 从结构节点服务加载结构节点；document 为 nil 时回退为空列表
	var structureNodes []*entity.DocumentStructureNode
	if s.structureNode != nil && document != nil {
		nodes, err := s.structureNode.ListDocumentNodes(ctx, document.ID, document.LastParseTaskId)
		if err == nil {
			structureNodes = nodes
		}
	}

	// 构建父块种子列表
	parentSeedList := s.buildParentSeedList(parsedText, parentSteps, structureNodes)

	// 为每个父块种子生成子块
	parentBlockList := make([]*vo.ParentBlockCandidate, 0)
	for _, parentSeed := range s.cleanupChunkList(parentSeedList) {
		if parentSeed == nil || strutil.IsBlank(parentSeed.Text) {
			continue
		}

		// 构建子块种子列表
		childSeedList := s.buildChildSeedList(parentSeed, childSteps, structureNodes)
		finalChildren := s.cleanupChunkList(childSeedList)

		// 如果没有子块，使用父块本身作为子块
		if len(finalChildren) == 0 {
			finalChildren = []*vo.ChunkCandidate{
				s.cloneChunkCandidate(parentSeed, strutil.Trim(parentSeed.Text)),
			}
		}

		parentBlock := &vo.ParentBlockCandidate{
			SectionPath: parentSeed.SectionPath,
			NodeId:      parentSeed.StructureNodeId,
			NodeType:    parentSeed.StructureNodeType,
			Text:        strutil.Trim(parentSeed.Text),
			SourceType:  parentSeed.SourceType,
			ChildChunks: finalChildren,
		}

		parentBlockList = append(parentBlockList, parentBlock)
	}

	return s.cleanupParentBlockList(parentBlockList), nil
}

// ---------------- 推荐判定 ----------------

// shouldUseStructure 判断是否应该使用结构切块
// 逻辑：适合的文件类型（PDF/DOC/DOCX/MD/HTML），并且结构等级达到中等或标题数量>=2
func (s *StrategyLogicImpl) shouldUseStructure(document *entity.Document, analysisResult *vo.DocumentAnalysisResult) bool {
	fileType := document.FileType
	suitableType := fileType == vo.FileTypePDF ||
		fileType == vo.FileTypeDOC ||
		fileType == vo.FileTypeDOCX ||
		fileType == vo.FileTypeMD ||
		fileType == vo.FileTypeHTML
	return suitableType && (analysisResult.StructureLevel >= vo.StructureLevelMedium || analysisResult.HeadingCount >= 2)
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
	parentSnapshot := s.buildPipelineSnapshot(slice.Map(parentSteps, func(_ int, step *vo.DocumentStrategyStepDraft) int {
		return step.StrategyType
	}))
	childSnapshot := s.buildPipelineSnapshot(slice.Map(childSteps, func(_ int, step *vo.DocumentStrategyStepDraft) int {
		return step.StrategyType
	}))
	return "PARENT:" + parentSnapshot + ";CHILD:" + childSnapshot
}

// buildPipelineSnapshot 将策略类型列表拼接为字符串
func (s *StrategyLogicImpl) buildPipelineSnapshot(strategyTypes []int) string {
	strList := slice.Map(strategyTypes, func(_ int, t int) string {
		return intToString(t)
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

// getStrategyTypeName 获取策略类型名称
func (s *StrategyLogicImpl) getStrategyTypeName(strategyType int) string {
	switch strategyType {
	case vo.StrategyTypeStructure:
		return "STRUCTURE"
	case vo.StrategyTypeRecursive:
		return "RECURSIVE"
	case vo.StrategyTypeSemantic:
		return "SEMANTIC"
	case vo.StrategyTypeLLM:
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
	case vo.StrategyTypeLLM:
		return "大模型智能切块策略。"
	case vo.StrategyTypeMarkdown:
		return "Markdown格式切块策略。"
	default:
		return "系统为当前流水线生成的推荐步骤。"
	}
}

// ---------------- 种子构建 ----------------

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
				SourceType: vo.ChunkSourceTypeOriginal,
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
		SourceType: vo.ChunkSourceTypeOriginal,
	}
	return s.executePipeline([]*vo.ChunkCandidate{originalSeed}, parentSteps, vo.PipelineTypeParent)
}

// buildChildSeedList 构建子块种子列表
func (s *StrategyLogicImpl) buildChildSeedList(parentSeed *vo.ChunkCandidate, childSteps []*entity.DocumentStrategyStep,
	structureNodes []*entity.DocumentStructureNode) []*vo.ChunkCandidate {

	if s.containsStructureStep(childSteps) && parentSeed.StructureNodeId != 0 && len(structureNodes) > 0 {
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

		// 只处理 SECTION、STEP、LIST_ITEM 类型的节点
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
			ChildChunks: childChunks,
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
		ChildChunks:       childChunks,
	}
}

// ---------------- 流水线调度 ----------------

// executePipeline 执行流水线
// 按步骤顺序将种子列表传递给各切块策略，产生最终的块候选列表
// 委托给 chunk.Registry 执行，实现策略模式 + 注册表的统一调度
func (s *StrategyLogicImpl) executePipeline(inputSeeds []*vo.ChunkCandidate, steps []*entity.DocumentStrategyStep, pipelineType string) []*vo.ChunkCandidate {
	currentChunks := s.cleanupChunkList(inputSeeds)
	if len(currentChunks) == 0 || s.strategyRegistry == nil {
		return currentChunks
	}

	// 映射到 chunk 包的 PipelineType
	var targetPipeline chunk.PipelineType
	switch pipelineType {
	case vo.PipelineTypeParent:
		targetPipeline = chunk.PipelineTypeParent
	default:
		targetPipeline = chunk.PipelineTypeChild
	}

	// 按步骤顺序调用 Registry 中的策略
	for _, step := range steps {
		strategyType := step.StrategyType
		if !s.isValidStrategyType(strategyType) {
			continue
		}

		strategyName := s.getStrategyTypeName(strategyType)
		if !s.strategyRegistry.Has(strategyName) {
			continue
		}

		// 将当前块候选转换为 chunk.Input
		outputList := make([]*chunk.Output, 0, len(currentChunks))
		for _, c := range currentChunks {
			if c == nil || strutil.IsBlank(c.Text) {
				continue
			}
			outs, err := s.strategyRegistry.Run(context.Background(), strategyName, &chunk.Input{
				Text:        strings.TrimSpace(c.Text),
				SectionPath: c.SectionPath,
				SourceType:  c.SourceType,
			}, targetPipeline)
			if err != nil {
				// 单个策略失败不中断，降级为保持原样
				outputList = append(outputList, &chunk.Output{
					Text:        strings.TrimSpace(c.Text),
					SectionPath: c.SectionPath,
					SourceType:  c.SourceType,
				})
				continue
			}
			outputList = append(outputList, outs...)
		}

		// 转换回 vo.ChunkCandidate
		nextChunks := make([]*vo.ChunkCandidate, 0, len(outputList))
		for _, out := range outputList {
			if out == nil || strutil.IsBlank(out.Text) {
				continue
			}
			nextChunks = append(nextChunks, &vo.ChunkCandidate{
				SectionPath: out.SectionPath,
				Text:        strings.TrimSpace(out.Text),
				SourceType:  out.SourceType,
			})
		}
		currentChunks = s.cleanupChunkList(nextChunks)
	}
	return s.cleanupChunkList(currentChunks)
}

// ---------------- 结构切块 ----------------

// applyStructureChunking 对候选列表应用结构切块（按标题行分段）
func (s *StrategyLogicImpl) applyStructureChunking(sourceList []*vo.ChunkCandidate, pipelineType string) []*vo.ChunkCandidate {
	resultList := make([]*vo.ChunkCandidate, 0)
	for _, candidate := range sourceList {
		if candidate == nil || strutil.IsBlank(candidate.Text) {
			continue
		}
		resultList = append(resultList, s.applyStructureChunkingText(candidate, pipelineType)...)
	}
	return resultList
}

// applyStructureChunkingText 对单段文本进行结构切块：使用行分类器识别标题，维护标题栈切分文本
func (s *StrategyLogicImpl) applyStructureChunkingText(candidate *vo.ChunkCandidate, pipelineType string) []*vo.ChunkCandidate {
	parsedText := candidate.Text
	baseSectionPath := candidate.SectionPath
	sourceType := candidate.SourceType

	resultList := make([]*vo.ChunkCandidate, 0)
	headingStack := make([]string, 0) // 标题栈
	currentChunkBuilder := strings.Builder{}
	currentSectionPath := utils.BlankToDefault(baseSectionPath, "")

	lines := strings.Split(parsedText, "\n")
	for _, line := range lines {
		trimmed := strutil.Trim(line)
		classification := s.Classify(trimmed)

		if classification.IsHeading() {
			// 遇到新标题，先刷出当前块
			resultList = s.flushChunk(resultList, sourceType, currentSectionPath, currentChunkBuilder.String())
			currentChunkBuilder.Reset()

			// 弹出栈中同级或更高的标题（仅保留更高级）
			classificationLevel := max(1, classification.Level)
			for len(headingStack) >= classificationLevel {
				headingStack = headingStack[:len(headingStack)-1]
			}
			headingStack = append(headingStack, classification.Title)
			currentSectionPath = s.composeSectionPath(baseSectionPath, strings.Join(headingStack, " > "))

			// 标题本身也计入当前块内容
			currentChunkBuilder.WriteString(trimmed)
			currentChunkBuilder.WriteByte('\n')
			continue
		}

		currentChunkBuilder.WriteString(line)
		currentChunkBuilder.WriteByte('\n')
	}

	resultList = s.flushChunk(resultList, sourceType, currentSectionPath, currentChunkBuilder.String())

	// 如果最终没有识别到任何结构，回退为递归切块
	if len(resultList) == 0 {
		return s.applyRecursiveChunking([]*vo.ChunkCandidate{
			{
				SectionPath: baseSectionPath,
				Text:        parsedText,
				SourceType:  sourceType,
			},
		}, pipelineType)
	}
	return resultList
}

// flushChunk 将累计文本作为一个块输出
func (s *StrategyLogicImpl) flushChunk(candidateList []*vo.ChunkCandidate, sourceType int, sectionPath, text string) []*vo.ChunkCandidate {
	trimmed := strutil.Trim(text)
	if trimmed == "" {
		return candidateList
	}
	return append(candidateList, &vo.ChunkCandidate{
		SectionPath: sectionPath,
		Text:        trimmed,
		SourceType:  sourceType,
	})
}

// composeSectionPath 拼接基础路径与当前层级路径
func (s *StrategyLogicImpl) composeSectionPath(baseSectionPath, currentSectionPath string) string {
	normalizedBase := strutil.Trim(baseSectionPath)
	normalizedCurrent := strutil.Trim(currentSectionPath)
	if normalizedBase == "" {
		return normalizedCurrent
	}
	if normalizedCurrent == "" {
		return normalizedBase
	}
	return normalizedBase + " > " + normalizedCurrent
}

// ---------------- 递归切块 ----------------

// applyRecursiveChunking 对候选列表应用递归切块
func (s *StrategyLogicImpl) applyRecursiveChunking(sourceList []*vo.ChunkCandidate, pipelineType string) []*vo.ChunkCandidate {
	resultList := make([]*vo.ChunkCandidate, 0)
	maxChars := s.resolveRecursiveMaxChars(pipelineType)
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

// recursiveSplit 递归切分文本：段落→行→句子→固定窗口，按优先级尝试
func (s *StrategyLogicImpl) recursiveSplit(text string, maxChars, overlapChars int) []string {
	trimmed := strutil.Trim(text)
	if trimmed == "" {
		return []string{}
	}
	if utf8.RuneCountInString(trimmed) <= maxChars {
		return []string{trimmed}
	}

	// 1) 按段落切分
	paragraphList := s.splitByRegex(trimmed, paragraphSplitRe)
	if len(paragraphList) > 1 {
		return s.mergeAndSplit(paragraphList, maxChars, overlapChars)
	}

	// 2) 按换行切分
	lineList := s.splitByRegex(trimmed, lineSplitRe)
	if len(lineList) > 1 {
		return s.mergeAndSplit(lineList, maxChars, overlapChars)
	}

	// 3) 按句子切分
	sentenceList := s.splitSentences(trimmed)
	if len(sentenceList) > 1 {
		return s.mergeAndSplit(sentenceList, maxChars, overlapChars)
	}

	// 4) 固定窗口切分
	return s.fixedWindowSplit(trimmed, maxChars, overlapChars)
}

// mergeAndSplit 将片段依次累加直到超出 maxChars 后切出，最后整体应用重叠
func (s *StrategyLogicImpl) mergeAndSplit(segmentList []string, maxChars, overlapChars int) []string {
	rawResultList := make([]string, 0)
	current := strings.Builder{}

	for _, segment := range segmentList {
		trimmed := strutil.Trim(segment)
		if trimmed == "" {
			continue
		}

		// 单个片段过大：先把累积的刷出，再对该片段递归切分
		if utf8.RuneCountInString(trimmed) > maxChars {
			if current.Len() > 0 {
				rawResultList = append(rawResultList, strutil.Trim(current.String()))
				current.Reset()
			}
			rawResultList = append(rawResultList, s.recursiveSplit(trimmed, maxChars, overlapChars)...)
			continue
		}

		// 累积超出：刷出当前
		if utf8.RuneCountInString(current.String())+utf8.RuneCountInString(trimmed)+1 > maxChars {
			rawResultList = append(rawResultList, strutil.Trim(current.String()))
			current.Reset()
		}
		current.WriteString(trimmed)
		current.WriteByte('\n')
	}

	if current.Len() > 0 {
		rawResultList = append(rawResultList, strutil.Trim(current.String()))
	}
	return s.applyOverlap(rawResultList, maxChars, overlapChars)
}

// applyOverlap 为块列表追加重叠前缀：取前一块尾部作为下一块的前缀上下文
func (s *StrategyLogicImpl) applyOverlap(rawChunkList []string, maxChars, overlapChars int) []string {
	if len(rawChunkList) == 0 || overlapChars <= 0 {
		return rawChunkList
	}

	overlappedChunkList := make([]string, 0, len(rawChunkList))
	for index, current := range rawChunkList {
		currentTrimmed := strutil.Trim(current)
		if currentTrimmed == "" {
			continue
		}
		if index == 0 {
			overlappedChunkList = append(overlappedChunkList, currentTrimmed)
			continue
		}

		previous := strutil.Trim(rawChunkList[index-1])
		overlapPrefix := s.buildOverlapPrefix(previous, currentTrimmed, maxChars, overlapChars)
		if overlapPrefix != "" {
			overlappedChunkList = append(overlappedChunkList, overlapPrefix+"\n"+currentTrimmed)
		} else {
			overlappedChunkList = append(overlappedChunkList, currentTrimmed)
		}
	}
	return overlappedChunkList
}

// buildOverlapPrefix 取 previous 尾部作为重叠前缀，长度受 maxChars-current 余量约束
func (s *StrategyLogicImpl) buildOverlapPrefix(previous, current string, maxChars, overlapChars int) string {
	previous = strutil.Trim(previous)
	current = strutil.Trim(current)
	if previous == "" || current == "" {
		return ""
	}

	allowedChars := min(overlapChars, max(0, maxChars-utf8.RuneCountInString(current)-1))
	if allowedChars <= 0 {
		return ""
	}

	prevRunes := []rune(previous)
	startIdx := len(prevRunes) - allowedChars
	if startIdx < 0 {
		startIdx = 0
	}
	return strutil.Trim(string(prevRunes[startIdx:]))
}

// fixedWindowSplit 使用固定窗口+步进对长文本进行兜底切分
func (s *StrategyLogicImpl) fixedWindowSplit(text string, maxChars, overlapChars int) []string {
	runes := []rune(strutil.Trim(text))
	total := len(runes)
	if total == 0 {
		return []string{}
	}
	if total <= maxChars {
		return []string{string(runes)}
	}

	result := make([]string, 0)
	step := max(1, maxChars-overlapChars)
	start := 0
	for start < total {
		end := start + maxChars
		if end > total {
			end = total
		}
		result = append(result, strutil.Trim(string(runes[start:end])))
		if end >= total {
			break
		}
		start += step
	}
	return result
}

// resolveRecursiveMaxChars 解析递归切块最大字符数（父块使用较大常量兜底）
func (s *StrategyLogicImpl) resolveRecursiveMaxChars(pipelineType string) int {
	if pipelineType == vo.PipelineTypeParent {
		return ParentBlockMaxChars
	}
	return s.recursiveMaxChars
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

// splitByRegex 按正则切分并去除空白片段
func (s *StrategyLogicImpl) splitByRegex(text string, re *regexp.Regexp) []string {
	raw := re.Split(text, -1)
	result := make([]string, 0, len(raw))
	for _, part := range raw {
		trimmed := strutil.Trim(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// splitSentences 按句末标点切分文本，并保留标点符号
func (s *StrategyLogicImpl) splitSentences(text string) []string {
	// 找到所有句末标点索引
	indices := sentenceSplitRe.FindAllStringIndex(text, -1)
	if len(indices) == 0 {
		trimmed := strutil.Trim(text)
		if trimmed == "" {
			return []string{}
		}
		return []string{trimmed}
	}
	result := make([]string, 0, len(indices)+1)
	prev := 0
	for _, idxPair := range indices {
		end := idxPair[1]
		segment := strutil.Trim(text[prev:end])
		if segment != "" {
			result = append(result, segment)
		}
		prev = end
	}
	if prev < len(text) {
		tail := strutil.Trim(text[prev:])
		if tail != "" {
			result = append(result, tail)
		}
	}
	return result
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

// extractTokens 提取文本的词元集合：英文单词 + 单个中文字符
func (s *StrategyLogicImpl) extractTokens(text string) map[string]bool {
	tokenSet := make(map[string]bool)
	// 英文单词逐个作为词元
	lower := strings.ToLower(text)
	matches := englishWordPattern.FindAllString(lower, -1)
	for _, m := range matches {
		tokenSet[m] = true
	}
	// 中文字符逐个作为词元
	for _, r := range text {
		if r >= 0x4e00 && r <= 0x9fa5 {
			tokenSet[string(r)] = true
		}
	}
	return tokenSet
}

// jaccard 计算两个集合的 Jaccard 相似度
func (s *StrategyLogicImpl) jaccard(left, right map[string]bool) float64 {
	if len(left) == 0 || len(right) == 0 {
		return 0
	}
	unionSize := len(left)
	intersectionSize := 0
	for token := range right {
		if left[token] {
			intersectionSize++
		} else {
			unionSize++
		}
	}
	if unionSize == 0 {
		return 0
	}

	return float64(intersectionSize) / float64(unionSize)
}

// resolveSemanticMaxChars 解析语义切块最大字符数（父块使用较大常量兜底）
func (s *StrategyLogicImpl) resolveSemanticMaxChars(pipelineType string) int {
	if pipelineType == vo.PipelineTypeParent {
		return max(ParentSemanticMaxChars, s.semanticMaxChars)
	}
	return s.semanticMaxChars
}

// resolveSemanticMinChars 解析语义切块最小字符数
func (s *StrategyLogicImpl) resolveSemanticMinChars(pipelineType string) int {
	if pipelineType == vo.PipelineTypeParent {
		return max(ParentSemanticMinChars, s.semanticMinChars)
	}
	return s.semanticMinChars
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
			llmChunks := s.llmSplit(ctx, sourceText)
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

// llmSplit 调用大模型提示模板切分文本，并从返回内容中解析 JSON 数组作为切块结果
func (s *StrategyLogicImpl) llmSplit(ctx context.Context, sourceText string) []string {
	userPrompt, err := s.promptTemplate.Render(prompt.DocumentLlmSplit, map[string]any{
		"sourceText": sourceText,
	})
	if err != nil {
		Warnf("大模型智能切块失败，回退到语义切块，err=%v", err)
		return nil
	}

	content, err := s.chatModel.Generate(ctx, "", userPrompt)
	if err != nil {
		Warnf("大模型智能切块失败，回退到语义切块，err=%v", err)
		return nil
	}

	jsonArray := s.extractJsonArray(content)
	if strutil.IsBlank(jsonArray) {
		return nil
	}
	// 简单按字符串方式解析 JSON 数组，避免额外依赖
	elements := s.parseStringJsonArray(jsonArray)
	filtered := make([]string, 0, len(elements))
	for _, e := range elements {
		trimmed := strutil.Trim(e)
		if trimmed != "" {
			filtered = append(filtered, trimmed)
		}
	}
	return filtered
}

// extractJsonArray 从文本中抽取首个完整 [ ... ] JSON 数组
func (s *StrategyLogicImpl) extractJsonArray(content string) string {
	start := strings.Index(content, "[")
	end := strings.LastIndex(content, "]")
	if start < 0 || end <= start {
		return ""
	}
	return content[start : end+1]
}

// parseStringJsonArray 简易解析 JSON 字符串数组（仅处理双引号字符串元素，无需引入额外依赖）
func (s *StrategyLogicImpl) parseStringJsonArray(content string) []string {
	trimmed := strutil.Trim(content)
	if !strings.HasPrefix(trimmed, "[") || !strings.HasSuffix(trimmed, "]") {
		return []string{}
	}
	inner := strutil.Trim(trimmed[1 : len(trimmed)-1])
	if inner == "" {
		return []string{}
	}

	elements := make([]string, 0)
	runes := []rune(inner)
	n := len(runes)
	i := 0
	for i < n {
		// 跳过空白和逗号
		for i < n && (runes[i] == ',' || runes[i] == ' ' || runes[i] == '\t' || runes[i] == '\r' || runes[i] == '\n') {
			i++
		}
		if i >= n {
			break
		}
		// 期望字符串起始
		if runes[i] != '"' {
			// 跳过非字符串元素直到下一个逗号
			for i < n && runes[i] != ',' {
				i++
			}
			continue
		}
		i++ // 跳过 "
		sb := strings.Builder{}
		for i < n {
			r := runes[i]
			if r == '\\' && i+1 < n {
				next := runes[i+1]
				switch next {
				case '"':
					sb.WriteByte('"')
				case '\\':
					sb.WriteByte('\\')
				case '/':
					sb.WriteByte('/')
				case 'n':
					sb.WriteByte('\n')
				case 't':
					sb.WriteByte('\t')
				case 'r':
					sb.WriteByte('\r')
				default:
					sb.WriteRune(r)
					sb.WriteRune(next)
				}
				i += 2
				continue
			}
			if r == '"' {
				i++
				break
			}
			sb.WriteRune(r)
			i++
		}
		elements = append(elements, sb.String())
	}
	return elements
}

// resolveLlmMaxChars 解析大模型切块最大字符数（父块使用较大常量兜底）
func (s *StrategyLogicImpl) resolveLlmMaxChars(pipelineType string) int {
	if pipelineType == vo.PipelineTypeParent {
		return max(s.llmMaxChars, ParentBlockMaxChars)
	}
	return s.llmMaxChars
}

// ---------------- 列表清理/去重 ----------------

// cleanupChunkList 去除空/重复块候选，以 章节路径+条目索引+文本 作为唯一键
func (s *StrategyLogicImpl) cleanupChunkList(chunks []*vo.ChunkCandidate) []*vo.ChunkCandidate {
	uniqueMap := make(map[string]*vo.ChunkCandidate)
	for _, chunk := range chunks {
		if strutil.IsBlank(chunk.Text) {
			continue
		}
		normalizedText := strutil.Trim(chunk.Text)
		uniqueKey := fmt.Sprintf("%s||%d||%s", utils.BlankToDefault(chunk.CanonicalPath, chunk.SectionPath), chunk.ItemIndex, normalizedText)
		if _, exists := uniqueMap[uniqueKey]; !exists {
			uniqueMap[uniqueKey] = s.cloneChunkCandidate(chunk, normalizedText)
		}
	}
	return maputil.Values(uniqueMap)
}

// cleanupParentBlockList 去除空/重复父块候选
func (s *StrategyLogicImpl) cleanupParentBlockList(blocks []*vo.ParentBlockCandidate) []*vo.ParentBlockCandidate {
	uniqueMap := make(map[string]*vo.ParentBlockCandidate)
	for _, block := range blocks {
		if strutil.IsBlank(block.Text) {
			continue
		}
		normalizedText := strutil.Trim(block.Text)
		uniqueKey := fmt.Sprintf("%s||%d||%s", utils.BlankToDefault(block.CanonicalPath, block.SectionPath), block.ItemIndex, normalizedText)
		if _, exists := uniqueMap[uniqueKey]; !exists {
			childChunks := append([]*vo.ChunkCandidate{}, block.ChildChunks...)
			uniqueMap[uniqueKey] = s.cloneParentBlockCandidate(block, childChunks, normalizedText)
		}
	}
	return maputil.Values(uniqueMap)
}

// ---------------- 小工具 ----------------

// intToString 将整数转为字符串（自实现，避免引入 strconv 外部调用）
func intToString(v int) string {
	if v == 0 {
		return "0"
	}
	neg := v < 0
	if neg {
		v = -v
	}
	var buf [20]byte
	pos := len(buf)
	for v > 0 {
		pos--
		buf[pos] = byte('0' + v%10)
		v /= 10
	}
	if neg {
		pos--
		buf[pos] = '-'
	}
	return string(buf[pos:])
}

// newStrategyError 构造策略业务错误
func newStrategyError(msg string) error {
	return common.NewBizError(20005, msg)
}

func (s *StrategyLogicImpl) Warnf(format string, args ...any) {
	logx.Alert(fmt.Sprintf(format, args...))
}
