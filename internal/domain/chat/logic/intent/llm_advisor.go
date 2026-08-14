package intent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/duke-git/lancet/v2/strutil"

	"github.com/swiftbit/know-agent/common/logx"
	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/chat/adapter"
	"github.com/swiftbit/know-agent/internal/domain/chat/adapter/model"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/enum"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
)

// LlmAdvisorRecognizer 基于大模型的语义增强意图示识别器
type LlmAdvisorRecognizer struct {
	chatModel model.ChatModel
	renderer  adapter.PromptRenderer
}

func NewLlmAdvisorRecognizer(chatModel model.ChatModel, renderer adapter.PromptRenderer) *LlmAdvisorRecognizer {
	return &LlmAdvisorRecognizer{
		chatModel: chatModel,
		renderer:  renderer,
	}
}

// Name 返回提供者的名称
func (r *LlmAdvisorRecognizer) Name() string {
	return "llm_advisor"
}

// Recognize 识别意图
func (r *LlmAdvisorRecognizer) Recognize(ctx context.Context, input *RecognitionInput) (*vo.IntentRecognitionResult, error) {
	if r.chatModel == nil || r.renderer == nil {
		return nil, fmt.Errorf("chat model 或 renderer 未配置")
	}

	// 渲染 Prompt
	subQuestionsList := input.SubQuestions
	if len(subQuestionsList) == 0 {
		subQuestionsList = []string{}
	}

	prompt, err := r.renderer.Render(enum.DocumentQueryUnderstanding, map[string]any{
		"originalQuestion":       input.OriginalQuestion,
		"rewrittenQuestion":      input.RewrittenQuestion,
		"subQuestions":           subQuestionsList,
		"historySummary":         input.HistorySummary,
		"answerRecentTranscript": input.AnswerRecentTranscript,
	})
	if err != nil {
		return nil, fmt.Errorf("渲染 Prompt 失败: %w", err)
	}

	// 调用 LLM
	raw, err := r.chatModel.Generate(ctx, "", prompt,
		model.WithTemperature(0.0),
		model.WithTopP(0.1),
	)
	if err != nil {
		return nil, fmt.Errorf("调用 chat model 失败: %w", err)
	}

	if utils.IsBlank(raw) {
		return nil, fmt.Errorf("chat model 返回为空字符串")
	}

	return r.parseAdvice(raw)
}

// queryUnderstandingAdvicePayload LLM 返回的查询理解建议
type queryUnderstandingAdvicePayload struct {
	QueryType                 string                     `json:"queryType"`
	Channels                  []string                   `json:"channels"`
	Entities                  []string                   `json:"entities"`
	TargetEntities            []string                   `json:"targetEntities"`
	ExcludedEntities          []string                   `json:"excludedEntities"`
	SectionAnchors            []string                   `json:"sectionAnchors"`
	StructureNavigationIntent *structureNavigationAdvice `json:"structureNavigationIntent"`
	TableOps                  []string                   `json:"tableOps"`
	AnswerShape               []string                   `json:"answerShape"`
	Confidence                float64                    `json:"confidence"`
	Reasons                   []string                   `json:"reasons"`
}

// structureNavigationAdvice LLM 返回的结构导航意图
type structureNavigationAdvice struct {
	Operations            []string `json:"operations"`
	AnchorStructureNodeId *int64   `json:"anchorStructureNodeId"`
	AnchorSectionPath     string   `json:"anchorSectionPath"`
	AnchorCanonicalPath   string   `json:"anchorCanonicalPath"`
	SectionAnchors        []string `json:"sectionAnchors"`
	Confidence            float64  `json:"confidence"`
}

// ============ 解析方法 ============

// parseAdvice 解析 LLM 返回的建议
func (r *LlmAdvisorRecognizer) parseAdvice(raw string) (*vo.IntentRecognitionResult, error) {
	var payload queryUnderstandingAdvicePayload
	if err := utils.Unmarshal(raw, &payload); err != nil {
		return nil, fmt.Errorf("解析查询理解 LLM 输出失败: raw=%q, err=%w", raw, err)
	}
	keyOf := func(entity string) (string, string, bool) {
		trim := utils.Trim(entity)
		return trim, trim, trim != ""
	}

	structureNavigationIntent := parseStructureNavigationIntent(payload.StructureNavigationIntent)
	answerShapePlan := parseAnswerShapePlan(payload.AnswerShape)
	confidence := normalizeConfidence(payload.Confidence)

	reasons := utils.FilterMapUniqueLimit(payload.Reasons, 8, keyOf)
	if len(reasons) == 0 {
		reasons = []string{"LLM 意图识别完成。"}
	}

	return &vo.IntentRecognitionResult{
		QueryType:                 enum.ParseQueryType(payload.QueryType),
		Channels:                  enum.ParseRetrievalIntents(payload.Channels),
		Entities:                  utils.FilterMapUniqueLimit(payload.Entities, 8, keyOf),
		TargetEntities:            utils.FilterMapUniqueLimit(payload.TargetEntities, 8, keyOf),
		ExcludedEntities:          utils.FilterMapUniqueLimit(payload.ExcludedEntities, 8, keyOf),
		SectionAnchors:            utils.FilterMapUniqueLimit(payload.SectionAnchors, 8, keyOf),
		StructureNavigationIntent: structureNavigationIntent,
		TableOps:                  utils.FilterMapUniqueLimit(payload.TableOps, 8, keyOf),
		AnswerShapePlan:           answerShapePlan,
		Confidence:                confidence,
		Reasons:                   reasons,
		Source:                    r.Name(),
	}, nil
}

// parseStructureNavigationIntent 解析结构导航意图
func parseStructureNavigationIntent(advice *structureNavigationAdvice) *vo.StructureNavigationIntent {
	if advice == nil {
		return nil
	}
	keyOf := func(entity string) (string, string, bool) {
		trim := utils.Trim(entity)
		return trim, trim, trim != ""
	}

	operations := enum.ParseStructureOperations(advice.Operations)
	confidence := normalizeConfidence(advice.Confidence)

	return &vo.StructureNavigationIntent{
		Operations:            operations,
		AnchorStructureNodeId: advice.AnchorStructureNodeId,
		AnchorSectionPath:     strutil.Trim(advice.AnchorSectionPath),
		AnchorCanonicalPath:   strutil.Trim(advice.AnchorCanonicalPath),
		SectionAnchors:        utils.FilterMapUniqueLimit(advice.SectionAnchors, 8, keyOf),
		Confidence:            confidence,
		Source:                "llm-query-understanding",
	}
}

// parseAnswerShapePlan 解析答案形态计划
func parseAnswerShapePlan(answerShapeRaw []string) []enum.AnswerShapeRequirement {
	if len(answerShapeRaw) == 0 {
		return nil
	}

	seen := make(map[string]bool)
	var requirements []enum.AnswerShapeRequirement

	for _, raw := range answerShapeRaw {
		normalized := strings.ToUpper(strutil.Trim(raw))
		if strutil.IsBlank(normalized) {
			continue
		}

		switch normalized {
		case string(enum.AnswerShapeRequirementCompare),
			string(enum.AnswerShapeRequirementList),
			string(enum.AnswerShapeRequirementSteps),
			string(enum.AnswerShapeRequirementNegativeBoundary),
			string(enum.AnswerShapeRequirementCategoryTableRowCoverage):
			if !seen[normalized] {
				seen[normalized] = true
				requirements = append(requirements, enum.AnswerShapeRequirement(normalized))
			}
		default:
			// 遇到未知值时返回空列表
			return nil
		}
	}

	return requirements
}
