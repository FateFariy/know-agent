package vo

import (
	"strings"

	"github.com/swiftbit/know-agent/common/utils"
)

// QuestionHistoryContext 提问历史上下文
type QuestionHistoryContext struct {
	RenderedText      string          // 完整渲染文本
	StructuredContext string          // 结构化上下文（证据锚点）
	RecentContext     string          // 近期对话上下文
	EvidenceAnchors   EvidenceAnchors // 证据锚点列表
	ResolvedTopic     string          // 推断的主题
	FollowUpQuestion  bool            // 是否为追问
	TotalBudget       int             // 总预算字符数
	RecentBudget      int             // 近期上下文实际长度
	StructuredBudget  int             // 结构化上下文实际长度
}

// NewQuestionHistoryContext 构建提问历史上下文
// 参数：
//   - question: 当前问题
//   - recentQuestionTranscript: 最近对话转录（含"用户："）
//   - queryUnderstanding: 查询理解结果（可为 nil）
//   - recentEvidenceAnchors: 最近的证据锚点（可为 nil）
//   - maxChars: 总预算字符数（应 > 0）
func NewQuestionHistoryContext(question, recentQuestionTranscript string, queryUnderstanding *IntentRecognitionResult,
	recentEvidenceAnchors EvidenceAnchors, maxChars int) *QuestionHistoryContext {
	normalizedQuestion := strings.TrimSpace(question)
	recentUserContext := extractRecentUserQuestions(recentQuestionTranscript)
	totalBudget := max(maxChars, 1)
	hasRecentContext := utils.IsNotBlank(recentUserContext)
	followUpQuestion := queryUnderstanding.IsFollowUpQuestion(normalizedQuestion)
	anchors := utils.FilterLimit(recentEvidenceAnchors, 5, func(anchor *EvidenceAnchor) bool {
		return anchor.HasAnchorIdentity()
	})

	historyContext := &QuestionHistoryContext{
		FollowUpQuestion: followUpQuestion,
		TotalBudget:      totalBudget,
	}
	if !followUpQuestion || (!hasRecentContext && len(anchors) == 0) {
		return historyContext
	}

	// 渲染近期上下文（取尾部预算）
	recentPart := renderRecentContext(recentUserContext, totalBudget)
	// 结构化部分使用剩余预算
	structuredBudget := totalBudget - utils.Len(recentPart)
	structuredPart := EvidenceAnchors(anchors).RenderStructuredContext(structuredBudget)
	renderedText := utils.JoinNonBlank("\n", structuredPart, recentPart)

	if renderedText == "" && len(anchors) == 0 {
		return historyContext
	}

	historyContext.RenderedText = renderedText
	historyContext.StructuredContext = structuredPart
	historyContext.ResolvedTopic = resolveTopic(anchors)
	historyContext.RecentContext = recentPart
	historyContext.EvidenceAnchors = anchors
	historyContext.RecentBudget = utils.Len(recentPart)
	historyContext.StructuredBudget = utils.Len(structuredPart)
	return historyContext
}

// 提取最近用户问题
func extractRecentUserQuestions(recentQuestionTranscript string) string {
	normalized := utils.Trim(recentQuestionTranscript)

	prefixes := []string{"【最近相关对话】", "最近相关对话："}
	for _, prefix := range prefixes {
		if strings.HasPrefix(normalized, prefix) {
			normalized = strings.TrimSpace(normalized[len(prefix):])
			break
		}
	}

	var builder strings.Builder
	lines := strings.Split(normalized, "\n")
	for _, line := range lines {
		trimmed := utils.Trim(line)
		if !strings.HasPrefix(trimmed, "用户：") {
			continue
		}
		if builder.Len() > 0 {
			builder.WriteString("\n")
		}
		builder.WriteString(trimmed)
	}

	return utils.Trim(builder.String())
}

// resolveTopic 从锚点中推断主题（优先取 sectionPath，否则取 documentName）
func resolveTopic(anchors []*EvidenceAnchor) string {
	if len(anchors) == 0 {
		return ""
	}
	anchor := anchors[0]
	return utils.BlankToDefault(anchor.SectionPath, anchor.DocumentName)
}

// renderRecentContext 渲染近期上下文（用户问题部分）
func renderRecentContext(recentUserContext string, budget int) string {
	if budget <= 0 || utils.IsBlank(recentUserContext) {
		return ""
	}
	title := "对话承接上下文（仅用于理解指代，不作为事实证据）：\n"
	titleLen := utils.Len(title)
	if budget <= titleLen {
		// 预算不够标题，只返回裁剪后的用户问题
		return utils.ClipTail(recentUserContext, budget)
	}
	body := utils.ClipTail(recentUserContext, budget-titleLen)
	if body == "" {
		return ""
	}
	return title + body
}
