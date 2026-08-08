package process

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/duke-git/lancet/v2/maputil"
	"github.com/duke-git/lancet/v2/slice"
	"github.com/duke-git/lancet/v2/stream"
	"github.com/duke-git/lancet/v2/strutil"

	"github.com/swiftbit/know-agent/common/logx"
	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/chat/adapter/model"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/prompt"
	"github.com/swiftbit/know-agent/internal/domain/document/logic/process/chunk"
	chunkllm "github.com/swiftbit/know-agent/internal/domain/document/logic/process/chunk/llm"
	chunkrecursive "github.com/swiftbit/know-agent/internal/domain/document/logic/process/chunk/recursive"
	chunksemantic "github.com/swiftbit/know-agent/internal/domain/document/logic/process/chunk/semantic"
	chunkstructure "github.com/swiftbit/know-agent/internal/domain/document/logic/process/chunk/structure"
	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/document/model/enum"
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

// ChunkCoordinateImpl 分块策略实现
type ChunkCoordinateImpl struct {
	nodeManager StructureNodeManager
	registry    map[int]chunk.Chunker
	classifier  *support.DocumentLineClassifier
	*option
}

type option struct {
	recursiveMaxChars           int
	recursiveOverlapChars       int
	semanticMaxChars            int
	semanticMinChars            int
	semanticSimilarityThreshold float64
	llmEnabled                  bool
	llmMaxChars                 int
	recommendLlmWhenLowQuality  bool
}

func NewChunkCoordinateImpl(svcCtx *svc.ServiceContext, chatModel model.ChatModel,
	promptTemplate prompt.Renderer, nodeManager StructureNodeManager) *ChunkCoordinateImpl {

	registry := make(map[int]chunk.Chunker)

	// 递归分块
	registry[enum.StrategyTypeRecursive] = chunkrecursive.NewChunker(
		chunkrecursive.WithMaxChars(svcCtx.Config.Chunk.RecursiveMaxChars),
		chunkrecursive.WithOverlapChars(svcCtx.Config.Chunk.RecursiveOverlapChars),
	)

	// 语义分块
	registry[enum.StrategyTypeSemantic] = chunksemantic.NewChunker(
		chunksemantic.WithMinChars(svcCtx.Config.Chunk.SemanticMinChars),
		chunksemantic.WithMaxChars(svcCtx.Config.Chunk.SemanticMaxChars),
		chunksemantic.WithSimilarityThreshold(svcCtx.Config.Chunk.SemanticSimilarityThreshold),
	)
	// 大模型切块
	registry[enum.StrategyTypeLLM] = chunkllm.NewChunker(chatModel, promptTemplate,
		chunkllm.WithLlmSplitPrompt(prompt.DocumentLlmSplit),
	)

	return &ChunkCoordinateImpl{
		nodeManager: nodeManager,
		registry:    registry,
		classifier:  &support.DocumentLineClassifier{},
		option: &option{
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

// Recommend 根据文档分析结果推荐最优的父块-子块策略组合。
// 整体思路：先通过若干判定函数分别评估结构/递归/语义/大模型切块的必要性，
// 再按"父块优先保留天然大语义单元、子块围绕召回边界精细化"的原则拼接流水线。
func (s *ChunkCoordinateImpl) Recommend(ctx context.Context, document *entity.Document, analysisResult *vo.AnalysisResult) (*vo.DocumentStrategyPlanDraft, error) {
	if document == nil || analysisResult == nil {
		return nil, fmt.Errorf("invaild value")
	}

	reasonList := make([]string, 0)

	// 是否启用结构切块，启用条件：文件类型被识别 +（结构等级达到中等或标题数≥2）
	structureRecommended := enum.FileTypeName(document.FileType) != "" &&
		(analysisResult.StructureLevel >= enum.StructureLevelMedium || analysisResult.HeadingCount >= 2)

	// 是否启用递归切块，启用条件：文本总长度或最长段落长度 ≥ 递归窗口上限（需要控制单次块大小）
	recursiveRecommended := max(analysisResult.CharCount, analysisResult.MaxParagraphLength) >= s.recursiveMaxChars

	// 是否启用语义切块，启用条件：文本长度达标 + 内容质量中等以上 + 段落数≥3（保证语义断点有意义）
	semanticRecommended := analysisResult.CharCount >= s.semanticMinChars &&
		analysisResult.ContentQualityLevel >= enum.ContentQualityLevelMedium &&
		analysisResult.ParagraphCount >= 3

	// 是否启用大模型智能切块，启用条件：允许低质量文档走 LLM + 内容质量为 Low + 文本长度达到最小语义窗口
	llmRecommended := s.recommendLlmWhenLowQuality &&
		analysisResult.ContentQualityLevel == enum.ContentQualityLevelLow &&
		analysisResult.CharCount >= s.semanticMinChars

	// 构建父块策略流水线（结构优先，否则递归大窗口兜底）
	parentStrategyTypes := make([]int, 0)
	parentReasonMap := make(map[int]string)

	if structureRecommended {
		// 结构明显 → 父块以结构切块为主，保留天然章节边界
		parentStrategyTypes = append(parentStrategyTypes, enum.StrategyTypeStructure)
		parentReasonMap[enum.StrategyTypeStructure] = "检测到文档具有较明显的标题或章节结构，父块优先保留天然章节边界。"
		reasonList = append(reasonList, "父块流水线优先采用基于文档结构切块，保留回答阶段需要的大语义单元。")
	} else {
		// 结构不明显 → 用较大窗口的递归分块作为稳定回答单元
		parentStrategyTypes = append(parentStrategyTypes, enum.StrategyTypeRecursive)
		parentReasonMap[enum.StrategyTypeRecursive] = "未识别出稳定结构时，父块先使用较大粒度的递归分块作为稳定回答单元。"
		reasonList = append(reasonList, "父块流水线未命中明显结构信号，默认使用较大粒度递归分块作为回答单元。")
	}

	// 构建子块策略流水线（大模型增强 → 语义优化 → 递归兜底）
	childStrategyTypes := make([]int, 0)
	childReasonMap := make(map[int]string)

	if llmRecommended {
		// 低质量文档优先用大模型智能切块增强
		childStrategyTypes = append(childStrategyTypes, enum.StrategyTypeLLM)
		childReasonMap[enum.StrategyTypeLLM] = "文档质量偏低或结构识别不稳定，子块先使用大模型智能切块增强复杂场景。"
		reasonList = append(reasonList, "子块流水线追加大模型智能切块，处理低质量或结构不稳定文本。")
	} else if semanticRecommended {
		// 语义边界明确 → 优先用语义分块优化召回边界
		childStrategyTypes = append(childStrategyTypes, enum.StrategyTypeSemantic)
		childReasonMap[enum.StrategyTypeSemantic] = "文本主题边界相对明确，子块先使用语义分块优化召回边界。"
		reasonList = append(reasonList, "子块流水线优先采用语义分块，优化召回边界和主题完整性。")
	}

	// 递归分块作为子块兜底（长度控制或默认保底）
	if recursiveRecommended || llmRecommended || len(childStrategyTypes) == 0 {
		childStrategyTypes = append(childStrategyTypes, enum.StrategyTypeRecursive)
		childReasonMap[enum.StrategyTypeRecursive] = "文档整体较长、存在超长段落，或需要在增强切块后追加长度兜底。"
		reasonList = append(reasonList, "子块流水线追加递归分块，控制召回单元长度并作为兜底。")
	}

	// 基于推荐的策略类型构建步骤草稿、拼接快照与理由
	parentSteps := s.buildDraftSteps(enum.PipelineTypeParent, parentStrategyTypes, parentReasonMap)
	childSteps := s.buildDraftSteps(enum.PipelineTypeChild, childStrategyTypes, childReasonMap)

	strategySnapshot := fmt.Sprintf("PARENT:%s;CHILD:%s", s.buildPipelineSnapshot(parentSteps), s.buildPipelineSnapshot(childSteps))

	return &vo.DocumentStrategyPlanDraft{
		ParentSteps:      parentSteps,
		ChildSteps:       childSteps,
		StrategySnapshot: strategySnapshot,
		RecommendReason:  strings.Join(reasonList, "；"),
	}, nil
}

// NormalizeSteps 将用户提交的策略类型标准化为可执行的步骤列表，保留已有的用户配置
//
// 处理步骤：
//  1. 标准化父/子流水线的策略类型（过滤未知/重复类型）
//  2. 以流水线类型 + 策略类型为键，构建 baseStep 查找表
//  3. 分别构建父/子块的标准化步骤
func (s *ChunkCoordinateImpl) NormalizeSteps(ctx context.Context, baseSteps []*entity.DocumentStrategyStep,
	parentStrategyTypes []int, childStrategyTypes []int, documentId int64) ([]*entity.DocumentStrategyStep, error) {

	// 标准化策略类型（过滤无效 + 去重）
	normalizedParentTypes := s.normalizePipelineTypes(parentStrategyTypes)
	normalizedChildTypes := s.normalizePipelineTypes(childStrategyTypes)

	// 按流水线+策略类型构建基础步骤映射（便于复用已存在的用户配置）
	baseStepMap := make(map[string]map[int]*entity.DocumentStrategyStep)
	for _, baseStep := range baseSteps {
		pipelineType := utils.BlankToDefault(baseStep.PipelineType, enum.PipelineTypeChild)
		if _, exists := baseStepMap[pipelineType]; !exists {
			baseStepMap[pipelineType] = make(map[int]*entity.DocumentStrategyStep)
		}
		baseStepMap[pipelineType][baseStep.StrategyType] = baseStep
	}

	normalizedStepList := make([]*entity.DocumentStrategyStep, 0)
	// 生成父块标准化步骤
	parentSteps := s.buildNormalizedSteps(
		enum.PipelineTypeParent,
		normalizedParentTypes,
		baseStepMap[enum.PipelineTypeParent],
		documentId,
	)
	normalizedStepList = append(normalizedStepList, parentSteps...)

	// 生成子块标准化步骤
	childSteps := s.buildNormalizedSteps(
		enum.PipelineTypeChild,
		normalizedChildTypes,
		baseStepMap[enum.PipelineTypeChild],
		documentId,
	)
	normalizedStepList = append(normalizedStepList, childSteps...)

	return normalizedStepList, nil
}

// BuildParentBlocks 执行完整的父-子块构建流程：先通过父块流水线生成父种子，再针对每个父种子走子块流水线产出子块
func (s *ChunkCoordinateImpl) BuildParentBlocks(ctx context.Context, document *entity.Document,
	steps []*entity.DocumentStrategyStep, parsedText string) ([]*vo.ParentChunkCandidate, error) {
	// 按父/子流水线拆分并排序步骤；任一缺失则返回相应错误
	parentSteps := s.sortPipelineSteps(steps, enum.PipelineTypeParent)
	childSteps := s.sortPipelineSteps(steps, enum.PipelineTypeChild)
	if len(parentSteps) == 0 {
		return nil, errorx.ErrParentBlockMissing
	}
	if len(childSteps) == 0 {
		return nil, errorx.ErrChildBlockMissing
	}

	// 加载已解析的文档结构节点（用于结构切块策略）
	var structureNodes []*entity.StructureNode
	if document != nil {
		nodes, err := s.nodeManager.ListDocumentNodes(ctx, document.ID, document.LastParseTaskId)
		if err != nil {
			return nil, err
		}
		structureNodes = nodes
	}

	// 生成父块种子列表
	parentSeedList := s.buildParentSeedList(ctx, parsedText, parentSteps, structureNodes)

	// 为每个父块种子派生其子块；无子块时以父块本身兜底
	parentBlockList := make([]*vo.ParentChunkCandidate, 0)
	for _, parentSeed := range s.cleanupChunkList(parentSeedList) {
		if parentSeed != nil && strutil.IsNotBlank(parentSeed.Text) {
			childSeedList := s.buildChildSeedList(ctx, parentSeed, childSteps, structureNodes)
			finalChildren := s.cleanupChunkList(childSeedList)

			trim := strutil.Trim(parentSeed.Text)
			if len(finalChildren) == 0 {
				// 兜底策略：子块流水线无产出 → 使用父块本身作为唯一子块
				finalChildren = []*vo.ChunkCandidate{
					s.cloneChunkCandidate(parentSeed, trim),
				}
			}

			parentBlock := &vo.ParentChunkCandidate{
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
	return s.cleanupParentBlockList(parentBlockList), nil
}

// ---------------- 草稿/标准化 ----------------

// buildDraftSteps 将策略类型列表构造成推荐步骤草稿（带上角色与理由），首项默认为主策略，其余按类型赋予优化/兜底/增强角色
func (s *ChunkCoordinateImpl) buildDraftSteps(pipelineType string, strategyTypes []int, reasonMap map[int]string) []*vo.DocumentStrategyStepDraft {
	return slice.Map(strategyTypes, func(index, strategyType int) *vo.DocumentStrategyStepDraft {
		return &vo.DocumentStrategyStepDraft{
			PipelineType:    pipelineType,
			StrategyType:    strategyType,
			StrategyRole:    s.resolveRole(index, strategyType),
			SourceType:      enum.StrategySourceTypeSystemRecommend,
			RecommendReason: utils.BlankToDefault(reasonMap[strategyType], "系统为当前流水线生成的推荐步骤。"),
		}
	})
}

// normalizePipelineTypes 标准化流水线输入：过滤未知策略类型并去重
func (s *ChunkCoordinateImpl) normalizePipelineTypes(strategyTypes []int) []int {
	return stream.FromSlice(strategyTypes).
		Filter(func(strategyType int) bool { return enum.StrategyTypeName(strategyType) != "" }).
		Distinct().ToSlice()
}

// buildNormalizedSteps 构建标准化步骤实体，若 baseStep 存在则标记为用户保留并复用原因；否则标记为用户追加。
func (s *ChunkCoordinateImpl) buildNormalizedSteps(pipelineType string, normalizedTypes []int,
	baseStepMap map[int]*entity.DocumentStrategyStep, documentId int64) []*entity.DocumentStrategyStep {
	return slice.Map(normalizedTypes, func(index, strategyType int) *entity.DocumentStrategyStep {
		baseStep := baseStepMap[strategyType]
		step := &entity.DocumentStrategyStep{
			DocumentId:      documentId,
			PipelineType:    pipelineType,
			StepNo:          index + 1,
			StrategyType:    strategyType,
			StrategyRole:    s.resolveRole(index, strategyType),
			SourceType:      enum.StrategySourceTypeUserAdd,
			ExecuteStatus:   enum.StrategyExecuteStatusWaitExecute,
			RecommendReason: "用户手动追加该策略。",
		}
		if baseStep != nil {
			step.SourceType = enum.StrategySourceTypeUserKeep
			step.RecommendReason = baseStep.RecommendReason
		}
		return step
	})
}

// sortPipelineSteps 过滤属于指定流水线的步骤并按 StepNo 升序排列
func (s *ChunkCoordinateImpl) sortPipelineSteps(steps []*entity.DocumentStrategyStep, pipelineType string) []*entity.DocumentStrategyStep {
	filtered := slice.Filter(steps, func(index int, item *entity.DocumentStrategyStep) bool {
		return utils.EqualsIgnoreCase(pipelineType, utils.BlankToDefault(item.PipelineType, enum.PipelineTypeChild))
	})
	slice.SortBy(filtered, func(a, b *entity.DocumentStrategyStep) bool { return a.StepNo < b.StepNo })
	return filtered
}

// buildPipelineSnapshot 将步骤按策略类型序列化为逗号分隔的字符串快照
func (s *ChunkCoordinateImpl) buildPipelineSnapshot(steps []*vo.DocumentStrategyStepDraft) string {
	strList := slice.Map(steps, func(_ int, step *vo.DocumentStrategyStepDraft) string {
		return strconv.Itoa(step.StrategyType)
	})
	return strings.Join(strList, ",")
}

// resolveRole 为指定步骤分配角色
func (s *ChunkCoordinateImpl) resolveRole(index int, strategyType int) int {
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
func (s *ChunkCoordinateImpl) buildParentSeedList(ctx context.Context, parsedText string, parentSteps []*entity.DocumentStrategyStep, structureNodes []*entity.StructureNode) []*vo.ChunkCandidate {
	if s.containsStructureStep(parentSteps) && len(structureNodes) > 0 {
		// 结构切块有节点可用 → 先产出章节级种子，再将剩余策略作为后续流水线
		structureSeeds := s.buildStructureParentSeeds(structureNodes)
		if len(structureSeeds) != 0 {
			remainingSteps := s.stripStructureSteps(parentSteps)
			if len(remainingSteps) == 0 {
				return structureSeeds
			}

			return s.executePipeline(ctx, structureSeeds, remainingSteps, enum.PipelineTypeParent)
		}
	}

	// 无结构步骤或节点 → 用整段文本作为父种子走完整流水线
	originalSeed := &vo.ChunkCandidate{
		Text:       parsedText,
		SourceType: enum.ChunkSourceTypeOriginal,
	}

	// 执行父块流水线
	return s.executePipeline(ctx, []*vo.ChunkCandidate{originalSeed}, parentSteps, enum.PipelineTypeParent)
}

// buildChildSeedList 为指定父种子构建子块种子列表，若步骤中含有结构切块且结构节点存在，优先按结构节点拆解子章节，否则克隆父种子再跑流水线
func (s *ChunkCoordinateImpl) buildChildSeedList(ctx context.Context, parentSeed *vo.ChunkCandidate, childSteps []*entity.DocumentStrategyStep, structureNodes []*entity.StructureNode) []*vo.ChunkCandidate {
	if s.containsStructureStep(childSteps) && parentSeed.StructureNodeId != 0 && len(structureNodes) > 0 {
		// 基于父种子的节点 ID 收集子节点，再进入后续流水线
		structureSeeds := s.buildStructureChildSeeds(parentSeed, structureNodes)

		remainingSteps := s.stripStructureSteps(childSteps)
		if len(remainingSteps) == 0 {
			return structureSeeds
		}

		return s.executePipeline(ctx, structureSeeds, remainingSteps, enum.PipelineTypeChild)
	}

	// 直接克隆父种子作为子块流水线的起点
	clonedSeed := s.cloneChunkCandidate(parentSeed, parentSeed.Text)

	// 执行子块流水线
	return s.executePipeline(ctx, []*vo.ChunkCandidate{clonedSeed}, childSteps, enum.PipelineTypeChild)
}

// containsStructureStep 检查步骤列表中是否存在结构切块策略
func (s *ChunkCoordinateImpl) containsStructureStep(steps []*entity.DocumentStrategyStep) bool {
	for _, step := range steps {
		if step.StrategyType == enum.StrategyTypeStructure {
			return true
		}
	}
	return false
}

// stripStructureSteps 过滤掉结构切块步骤（结构切块已经在流水线前处理）
func (s *ChunkCoordinateImpl) stripStructureSteps(steps []*entity.DocumentStrategyStep) []*entity.DocumentStrategyStep {
	return slice.Filter(steps, func(index int, step *entity.DocumentStrategyStep) bool {
		return step.StrategyType != enum.StrategyTypeStructure
	})
}

// buildStructureParentSeeds 从结构节点中筛选"内容承载章节"生成父块种子，判定规则：含有子章节时需额外验证内容长度显著超过标题或出现换行
func (s *ChunkCoordinateImpl) buildStructureParentSeeds(structureNodes []*entity.StructureNode) []*vo.ChunkCandidate {
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
		if node.NodeType == vo.NodeTypeSection && s.isContentBearingSection(node, parentHasChildSection[node.ID]) {
			seeds = append(seeds, s.newChunkCandidate(node, enum.ChunkSourceTypeOriginal))
		}
	}

	return seeds
}

// buildStructureChildSeeds 根据父种子的节点 ID 从结构节点中挑出其子节点作为子块种子。
// 仅保留 SECTION / STEP / LIST_ITEM 三类有实际内容的子节点；否则回退到克隆父种子。
func (s *ChunkCoordinateImpl) buildStructureChildSeeds(parentSeed *vo.ChunkCandidate, structureNodes []*entity.StructureNode) []*vo.ChunkCandidate {
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
				seeds = append(seeds, s.newChunkCandidate(child, enum.ChunkSourceTypeOriginal))
			}
		}
	}

	if len(seeds) > 0 {
		return seeds
	}

	// 回退：无合适子节点时将父种子本身克隆为唯一子块
	return []*vo.ChunkCandidate{s.cloneChunkCandidate(parentSeed, parentSeed.Text)}
}

// isContentBearingSection 判断该章节是否为"内容承载章节"，排除仅作为容器而没有实际文本的章节（如纯嵌套目录）
func (s *ChunkCoordinateImpl) isContentBearingSection(node *entity.StructureNode, hasChildSection bool) bool {
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
func (s *ChunkCoordinateImpl) cloneChunkCandidate(original *vo.ChunkCandidate, text string) *vo.ChunkCandidate {
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

// cloneParentBlockCandidate 克隆 ParentChunkCandidate
func (s *ChunkCoordinateImpl) cloneParentBlockCandidate(source *vo.ParentChunkCandidate, childChunks []*vo.ChunkCandidate, text string) *vo.ParentChunkCandidate {
	if source == nil {
		return &vo.ParentChunkCandidate{
			Text:        text,
			SourceType:  enum.ChunkSourceTypeOriginal,
			ChildChunks: append([]*vo.ChunkCandidate{}, childChunks...),
		}
	}
	return &vo.ParentChunkCandidate{
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
func (s *ChunkCoordinateImpl) executePipeline(ctx context.Context, inputSeeds []*vo.ChunkCandidate, steps []*entity.DocumentStrategyStep, pipelineType string) []*vo.ChunkCandidate {
	// 初次清理：去除空文本和重复项
	currentChunks := s.cleanupChunkList(inputSeeds)
	if len(currentChunks) == 0 {
		return currentChunks
	}

	for _, step := range steps {
		strategy, ok := s.registry[step.StrategyType]
		if !ok {
			continue
		}

		// 根据策略类型与流水线类型生成额外选项（父块流水线会使用较大窗口）
		extraOpts := s.buildPipelineOptions(step.StrategyType, pipelineType)

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
				outputs = s.applyLlmChunking(ctx, input, pipelineType, extraOpts...)
			} else {
				outputs, _ = strategy.Chunk(ctx, input, extraOpts...)
			}
			// 结构切块无产出时，使用递归策略兜底
			if len(outputs) == 0 && step.StrategyType == enum.StrategyTypeStructure {
				outputs, _ = s.registry[enum.StrategyTypeRecursive].Chunk(ctx, input, extraOpts...)
			}
			for _, out := range outputs {
				if strutil.IsNotBlank(out.Text) {
					nextChunks = append(nextChunks, s.cloneChunkCandidate(candidate, out.Text))
				}
			}
		}
		// 每步结束后清理，避免中间产物污染下游
		currentChunks = s.cleanupChunkList(nextChunks)
	}
	return s.cleanupChunkList(currentChunks)
}

// buildPipelineOptions 根据流水线类型和策略类型生成额外的策略选项
func (s *ChunkCoordinateImpl) buildPipelineOptions(strategyType int, pipelineType string) []chunk.Option {
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
		maxChars := max(s.semanticMaxChars, ParentSemanticMaxChars)
		minChars := max(s.semanticMinChars, ParentSemanticMinChars)
		return []chunk.Option{
			chunksemantic.WithMaxChars(maxChars),
			chunksemantic.WithMinChars(minChars),
		}
	default:
		return nil
	}
}

// cleanupChunkList 清理 ChunkCandidate 列表：过滤空文本并按 路径+序号+文本 去重
func (s *ChunkCoordinateImpl) cleanupChunkList(chunks []*vo.ChunkCandidate) []*vo.ChunkCandidate {
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

// cleanupParentBlockList 清理父块列表：规则与子块一致，path+itemIndex+trim 去重
func (s *ChunkCoordinateImpl) cleanupParentBlockList(blocks []*vo.ParentChunkCandidate) []*vo.ParentChunkCandidate {
	result := make(map[string]*vo.ParentChunkCandidate)
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

// applyLlmChunking 大模型智能切块
//
// 策略：
//  1. LLM 未启用 → 回退到语义切块
//  2. 输入超长 → 先用递归切块切到 llmMaxChars 以下
//  3. 逐项调用 LLM；失败或无产出 → 回退到语义切块补全
func (s *ChunkCoordinateImpl) applyLlmChunking(ctx context.Context, input *chunk.TextBlock, pipeType string, extraOpts ...chunk.Option) []*chunk.TextBlock {
	var outputs []*chunk.TextBlock
	var err error
	// LLM 未启用 → 直接使用语义切块
	if !s.llmEnabled {
		outputs, _ = s.registry[enum.StrategyTypeSemantic].Chunk(ctx, input, extraOpts...)
		return outputs
	}
	// 输入过长 → 先以递归切块拆分到 LLM 上限
	if utils.Len(input.Text) > s.llmMaxChars {
		llmMaxChars := utils.Ternary(pipeType == enum.PipelineTypeParent, max(s.llmMaxChars, ParentBlockMaxChars), s.llmMaxChars)
		outputs, _ = s.registry[enum.StrategyTypeRecursive].Chunk(ctx, input, chunkrecursive.WithOverlapChars(0), chunkrecursive.WithMaxChars(llmMaxChars))
	}

	// 逐项调用 LLM 切块；失败/空产出回退到语义切块
	resultList := make([]*chunk.TextBlock, 0, len(outputs))
	for _, item := range outputs {
		if strutil.IsNotBlank(item.Text) {
			outputs, err = s.registry[enum.StrategyTypeLLM].Chunk(ctx, item)
			if err != nil {
				logx.Warnf("大模型智能切块失败，回退到语义切块，err=%v", err)
			}
			if len(outputs) == 0 {
				outputs, _ = s.registry[enum.StrategyTypeSemantic].Chunk(ctx, item, extraOpts...)
			}
			resultList = append(resultList, outputs...)
		}
	}
	return resultList
}

// newChunkCandidate 由结构节点构造新的块候选（保留章节/路径/序号等元信息）
func (s *ChunkCoordinateImpl) newChunkCandidate(node *entity.StructureNode, sourceType int) *vo.ChunkCandidate {
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
