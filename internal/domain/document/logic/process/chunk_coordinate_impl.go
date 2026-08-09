package process

import (
	"context"
	"fmt"
	"strings"

	"github.com/duke-git/lancet/v2/slice"
	"github.com/duke-git/lancet/v2/stream"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/chat/adapter/model"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/prompt"
	"github.com/swiftbit/know-agent/internal/domain/document/logic/process/chunk"
	chunkllm "github.com/swiftbit/know-agent/internal/domain/document/logic/process/chunk/llm"
	chunkrecursive "github.com/swiftbit/know-agent/internal/domain/document/logic/process/chunk/recursive"
	chunksemantic "github.com/swiftbit/know-agent/internal/domain/document/logic/process/chunk/semantic"
	"github.com/swiftbit/know-agent/internal/domain/document/model/aggregate"
	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/document/model/enum"
	"github.com/swiftbit/know-agent/internal/domain/document/model/vo"
	"github.com/swiftbit/know-agent/internal/domain/document/support"
	"github.com/swiftbit/know-agent/internal/svc"
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
func (s *ChunkCoordinateImpl) Recommend(ctx context.Context, document *entity.Document, analysisResult *aggregate.AnalysisResult) (*vo.DocumentStrategyPlanDraft, error) {
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

	strategySnapshot := fmt.Sprintf("PARENT:%s;CHILD:%s", parentSteps.PipelineSnapshot(), childSteps.PipelineSnapshot())

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

// ---------------- 草稿/标准化 ----------------

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
			StrategyRole:    enum.ResolveRole(index, strategyType),
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
