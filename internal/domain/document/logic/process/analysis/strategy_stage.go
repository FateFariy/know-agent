package analysis

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/duke-git/lancet/v2/slice"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/document/adapter"
	"github.com/swiftbit/know-agent/internal/domain/document/model/aggregate"
	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/document/model/enum"
	"github.com/swiftbit/know-agent/internal/domain/document/model/vo"
	"github.com/swiftbit/know-agent/internal/svc"
)

// StrategyStage 策略推荐阶段：推进任务到策略路由、生成推荐策略
type StrategyStage struct {
	repo                       adapter.DocumentRepository
	resolver                   IndexingConfigResolver
	llmEnabled                 bool
	recommendLlmWhenLowQuality bool
	llmValid                   bool
}

func NewStrategyStage(svcCtx *svc.ServiceContext, repo adapter.DocumentRepository, resolver IndexingConfigResolver) *StrategyStage {
	return &StrategyStage{
		repo:                       repo,
		resolver:                   resolver,
		llmEnabled:                 svcCtx.Config.Chunk.LlmEnabled,
		recommendLlmWhenLowQuality: svcCtx.Config.Chunk.RecommendLlmWhenLowQuality,
		llmValid:                   svcCtx.ChatModel != nil,
	}
}

func (p *StrategyStage) Name() string {
	return "策略推荐阶段"
}

func (p *StrategyStage) Execute(ctx context.Context, parseCtx *Context) (err error) {
	parseCtx.Task.CurrentStage = enum.TaskStageStrategyRoute

	// 推进任务阶段到"策略路由"
	if err = p.repo.UpdateTaskById(ctx, &entity.DocumentTask{
		ID:           parseCtx.Task.ID,
		CurrentStage: parseCtx.Task.CurrentStage,
	}); err != nil {
		return
	}

	// 写入"开始分析解析结果并生成推荐策略"日志
	strategyStartDetail, _ := json.Marshal(map[string]any{
		"blockCount":         len(parseCtx.AnalysisResult.Blocks),
		"structureNodeCount": len(parseCtx.SaveCtx.StructureNodes),
		"charCount":          parseCtx.AnalysisResult.CharCount,
		"tokenCount":         parseCtx.AnalysisResult.TokenCount,
	})
	strategyStartLog := &entity.DocumentTaskLog{
		TaskId:       parseCtx.TaskId,
		DocumentId:   parseCtx.DocumentId,
		StageType:    enum.TaskStageStrategyRoute,
		EventType:    enum.TaskEventStart,
		LogLevel:     enum.LogLevelInfo,
		OperatorType: enum.OperatorTypeSystem,
		Content:      "开始分析解析结果并生成推荐策略。",
		DetailJson:   string(strategyStartDetail),
	}
	_ = p.repo.InsertTaskLog(ctx, strategyStartLog)

	// 生成推荐方案
	planDraft, err := p.recommend(ctx, parseCtx.Document, parseCtx.AnalysisResult)
	if err != nil {
		return
	}
	parseCtx.StrategyPlanDraft = planDraft

	return nil
}

// Recommend 根据文档分析结果推荐最优的父块-子块策略组合。
// 整体思路：先通过若干判定函数分别评估结构/递归/语义/大模型切块的必要性，
// 再按"父块优先保留天然大语义单元、子块围绕召回边界精细化"的原则拼接流水线。
func (p *StrategyStage) recommend(ctx context.Context, document *entity.Document, analysisResult *aggregate.AnalysisResult) (*vo.DocumentStrategyPlanDraft, error) {
	if document == nil || analysisResult == nil {
		return nil, fmt.Errorf("invaild value")
	}
	options := p.resolver.Resolve(ctx, document)

	recursiveMaxChars := options.ResolveRecursiveMaxChars(enum.PipelineTypeChild)
	semanticMinChars := options.ResolveSemanticMinChars(enum.PipelineTypeChild)
	structureRecommended := analysisResult.ShouldUseStructure(document.FileType)
	recursiveRecommended := analysisResult.ShouldUseRecursive(recursiveMaxChars)
	semanticRecommended := analysisResult.ShouldUseSemantic(semanticMinChars)
	llmRecommended := analysisResult.ShouldUseLlm(semanticMinChars) &&
		p.llmEnabled && p.recommendLlmWhenLowQuality && p.llmValid

	reasonList := make([]string, 0, 4)
	// 构建父块策略流水线（结构优先，否则递归大窗口兜底）
	parentStrategyTypes := make([]int, 0, 4)
	parentReasonMap := make(map[int]string, 4)

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
	if recursiveRecommended || len(childStrategyTypes) == 0 {
		childStrategyTypes = append(childStrategyTypes, enum.StrategyTypeRecursive)
		childReasonMap[enum.StrategyTypeRecursive] = "文档整体较长、存在超长段落，或需要在增强切块后追加长度兜底。"
		reasonList = append(reasonList, "子块流水线追加递归分块，控制召回单元长度并作为兜底。")
	}

	// 基于推荐的策略类型构建步骤草稿、拼接快照与理由
	parentSteps := p.buildDraftSteps(enum.PipelineTypeParent, parentStrategyTypes, parentReasonMap)
	childSteps := p.buildDraftSteps(enum.PipelineTypeChild, childStrategyTypes, childReasonMap)

	strategySnapshot := fmt.Sprintf("PARENT:%s;CHILD:%s", parentSteps.PipelineSnapshot(), childSteps.PipelineSnapshot())

	return &vo.DocumentStrategyPlanDraft{
		ParentSteps:      parentSteps,
		ChildSteps:       childSteps,
		StrategySnapshot: strategySnapshot,
		RecommendReason:  strings.Join(reasonList, "；"),
	}, nil
}

// buildDraftSteps 将策略类型列表构造成推荐步骤草稿（带上角色与理由），首项默认为主策略，其余按类型赋予优化/兜底/增强角色
func (p *StrategyStage) buildDraftSteps(pipelineType string, strategyTypes []int, reasonMap map[int]string) vo.DocumentStrategyStepDrafts {
	return slice.Map(strategyTypes, func(index, strategyType int) *vo.DocumentStrategyStepDraft {
		return &vo.DocumentStrategyStepDraft{
			PipelineType:    pipelineType,
			StrategyType:    strategyType,
			StrategyRole:    enum.ResolveRole(index, strategyType),
			SourceType:      enum.StrategySourceTypeSystemRecommend,
			RecommendReason: utils.BlankToDefault(reasonMap[strategyType], "系统为当前流水线生成的推荐步骤。"),
		}
	})
}
