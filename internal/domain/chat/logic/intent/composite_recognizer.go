package intent

import (
	"context"

	"github.com/swiftbit/know-agent/common/logx"
	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/chat/adapter"
	"github.com/swiftbit/know-agent/internal/domain/chat/adapter/model"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/conversation"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/enum"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
)

// 置信度阈值
const (
	advisorConfidenceThreshold             = 0.72 // LLM advisor 置信度阈值
	structureNavigationConfidenceThreshold = 0.65 // 结构导航置信度阈值
)

// CompositeIntentRecognizer 基于大模型（LLM）的意图识别器
// 将散落在路由、表格、GraphRAG、RAPTOR 的主链路关键词硬判收口成受控建议。
type CompositeIntentRecognizer struct {
	fallback *DeterministicFallbackRecognizer
	advisor  *LlmAdvisorRecognizer
}

// NewCompositeIntentRecognizer 创建意图识器
func NewCompositeIntentRecognizer(chatModel model.ChatModel, renderer adapter.PromptRenderer) *CompositeIntentRecognizer {
	return &CompositeIntentRecognizer{
		fallback: NewDeterministicFallbackRecognizer(),
		advisor:  NewLlmAdvisorRecognizer(chatModel, renderer),
	}
}

func (s *CompositeIntentRecognizer) Name() string {
	return "composite-recognizer"
}

// Recognize 统一意图识别入口, 确定性回退 → LLM advisor 增强 → 验证合并
func (s *CompositeIntentRecognizer) Recognize(ctx context.Context, input *conversation.RecognitionInput) (*vo.IntentRecognitionResult, error) {
	fallback, _ := s.fallback.Recognize(ctx, input)
	advised, err := s.advisor.Recognize(ctx, input)
	if err != nil {
		logx.Warnf("意图分析 advisor 调用失败，回退确定性结构信号: question=%q, err=%v", input.OriginalQuestion, err)
	}
	return s.validate(advised, fallback), nil
}

// validate 验证并合并 advisor 和 fallback 结果，生成最终意图识别结果。
// 核心策略：优先采纳 advisor 的高置信度建议，但确定性结构导航（如锚点语法）具有最高优先级，
// 同时确保 fallback 作为安全兜底，最终结果始终包含通用检索通道。
func (s *CompositeIntentRecognizer) validate(advised, fallback *vo.IntentRecognitionResult) *vo.IntentRecognitionResult {
	if advised == nil {
		return fallback
	}

	// 归一化置信度到 [0,1] 区间
	confidence := normalizeConfidence(advised.Confidence)
	queryType := utils.BlankToDefault(advised.QueryType, enum.QueryTypeDocumentQA)

	// 合并检索通道，通用通道作为基础，fallback 通道提供保守补充
	channels := []enum.RetrievalIntent{enum.RetrievalIntentGeneral}
	channels = append(channels, fallback.Channels...)
	highConfidence := confidence >= advisorConfidenceThreshold
	if highConfidence {
		// 高置信度时，追加 advisor 的通道（如语义搜索等）
		channels = append(channels, advised.Channels...)
	}
	channels = utils.Distinct(channels, func(ch enum.RetrievalIntent) enum.RetrievalIntent {
		return ch
	})

	// 确定有效查询类型和置信度
	effectiveType := queryType
	effectiveConfidence := confidence
	if !highConfidence {
		// 低置信度时回退到 fallback 的类型和置信度
		effectiveType = utils.BlankToDefault(fallback.QueryType, enum.QueryTypeDocumentQA)
		effectiveConfidence = fallback.Confidence
	}

	// 如果 fallback 中包含明确的结构导航语法（如锚点），则强制设为结构导航类型
	deterministicStructureNavigation := fallback.IsStructureNavigationConfident(structureNavigationConfidenceThreshold)
	if deterministicStructureNavigation {
		effectiveType = enum.QueryTypeStructureNavigation
		effectiveConfidence = normalizeConfidence(fallback.Confidence)
	}

	// 辅助函数：提取并去重字符串实体（去空格，非空才保留）
	keyOf := func(entity string) (string, string, bool) {
		trim := utils.Trim(entity)
		return trim, trim, trim != ""
	}

	// 从 fallback 中提取确定性锚点（最多8个）
	deterministicAnchors := utils.FilterMapUniqueLimit(fallback.SectionAnchors, 8, keyOf)

	// 根据有效类型和置信度，选择最终的结构导航意图（可能来自 advisor 或 fallback）
	structureNavigationIntent := selectStructureNavigationIntent(
		effectiveType, effectiveConfidence, advised, fallback, deterministicAnchors,
	)

	// 合并章节锚点
	// 确定性锚点优先，若存在结构导航意图则追加其锚点（总数限制8）
	sectionAnchors := deterministicAnchors
	if structureNavigationIntent != nil {
		sectionAnchors = mergeStrings(deterministicAnchors, structureNavigationIntent.SectionAnchors, 8)
	}

	// 收集决策原因（用于调试和可观测性）
	reasons := append([]string{}, advised.Reasons...)
	if deterministicStructureNavigation && queryType != enum.QueryTypeStructureNavigation {
		reasons = append(reasons, "当前轮确定性结构语法已确认，保留 STRUCTURE_NAVIGATION；advisor 只补充语义通道。")
	}
	if !highConfidence {
		reasons = append(reasons, "advisor 置信度不足，保留确定性保守计划。")
	}

	// 降级处理：若决定为结构导航但缺少合法 intent，则安全回退
	if effectiveType == enum.QueryTypeStructureNavigation && structureNavigationIntent == nil {
		// 回退到 fallback 类型，若 fallback 也是结构导航则改为文档问答
		fallbackType := utils.BlankToDefault(fallback.QueryType, enum.QueryTypeDocumentQA)
		if fallbackType == enum.QueryTypeStructureNavigation {
			fallbackType = enum.QueryTypeDocumentQA
		}
		effectiveType = fallbackType
		effectiveConfidence = normalizeConfidence(fallback.Confidence)

		// 移除 STRUCTURE 检索通道，确保至少保留通用通道
		channels = utils.Filter(channels, func(ch enum.RetrievalIntent) bool {
			return ch != enum.RetrievalIntentStructure
		})
		if len(channels) == 0 {
			channels = append(channels, enum.RetrievalIntentGeneral)
		}
		reasons = append(reasons, "structure navigation 缺少合法 intent，按当前轮确定性结果 fail closed。")
	}

	// 回答形状计划：仅在 advisor 高置信度时保留
	answerShapePlan := advised.AnswerShapePlan
	if !highConfidence {
		answerShapePlan = nil
	}

	// 组装最终结果
	return &vo.IntentRecognitionResult{
		QueryType:                 effectiveType,
		Channels:                  channels,
		Entities:                  utils.FilterMapUniqueLimit(advised.Entities, 8, keyOf),
		TargetEntities:            utils.FilterMapUniqueLimit(advised.TargetEntities, 8, keyOf),
		ExcludedEntities:          utils.FilterMapUniqueLimit(advised.ExcludedEntities, 8, keyOf),
		SectionAnchors:            sectionAnchors,
		StructureNavigationIntent: structureNavigationIntent,
		TableOps:                  utils.FilterMapUniqueLimit(advised.TableOps, 8, keyOf),
		AnswerShapePlan:           answerShapePlan,
		Confidence:                effectiveConfidence,
		Reasons:                   utils.FilterMapUniqueLimit(reasons, 10, keyOf),
		Source:                    utils.BlankToDefault(advised.Source, "intent-recognize"),
	}
}

// selectStructureNavigationIntent 选择最终使用的结构导航意图
func selectStructureNavigationIntent(effectiveType enum.QueryType, confidence float64, advised, fallback *vo.IntentRecognitionResult, sectionAnchors []string) *vo.StructureNavigationIntent {
	if effectiveType != enum.QueryTypeStructureNavigation || confidence < structureNavigationConfidenceThreshold {
		return nil
	}

	// 优先使用 fallback 的确定性意图
	if fallback.IsStructureNavigationConfident(structureNavigationConfidenceThreshold) {
		return normalizeStructureNavigationIntent(fallback.StructureNavigationIntent, sectionAnchors, "deterministic-fallback")
	}

	// 其次使用 advisor 的 LLM 意图
	if advised != nil {
		advisedIntent := advised.StructureNavigationIntent
		if advisedIntent.IsConfident(structureNavigationConfidenceThreshold) {
			return normalizeStructureNavigationIntent(advisedIntent, sectionAnchors, "llm-intent-recognize")
		}
	}

	return nil
}

// normalizeStructureNavigationIntent 归一化结构导航意图
func normalizeStructureNavigationIntent(intent *vo.StructureNavigationIntent, sectionAnchors []string, fallbackSource string) *vo.StructureNavigationIntent {
	if intent == nil {
		return nil
	}

	operations := utils.FilterUniqueLimit(intent.Operations, 4, func(op enum.StructureNavigationOperation) (string, bool) {
		return op, op != ""
	})
	if len(operations) == 0 {
		return nil
	}
	return &vo.StructureNavigationIntent{
		Operations:            operations,
		AnchorStructureNodeId: intent.AnchorStructureNodeId,
		AnchorSectionPath:     intent.AnchorSectionPath,
		AnchorCanonicalPath:   intent.AnchorCanonicalPath,
		SectionAnchors:        mergeStrings(sectionAnchors, intent.SectionAnchors, 8),
		Confidence:            normalizeConfidence(intent.Confidence),
		Source:                utils.BlankToDefault(intent.Source, fallbackSource),
	}
}

// mergeStrings 合并两个字符串列表并去重，限制长度
func mergeStrings(first, second []string, limit int) []string {
	result := append(first, second...)
	return utils.FilterUniqueLimit(result, limit, func(s string) (string, bool) {
		return s, s != ""
	})
}

// normalizeConfidence 将模型返回的可能超出 0-1 的置信度规范化到 [0, 1)，再由调用方与阈值比较
func normalizeConfidence(confidence float64) float64 {
	if confidence > 1 {
		return confidence / 100.0
	}
	return max(0, confidence)
}
