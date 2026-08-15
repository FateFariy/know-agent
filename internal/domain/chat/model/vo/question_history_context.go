package vo

import (
	"strings"

	"github.com/swiftbit/know-agent/common/utils"
)

// QuestionHistoryContext 提问历史上下文
type QuestionHistoryContext struct {
	RenderedText      string          // 完整渲染文本
	StructuredContext string          // 结构化上下文（证据锚点）
	RecentContext     string          // 近期上下文
	EvidenceAnchors   EvidenceAnchors // 证据锚点列表
	ResolvedTopic     string          // 推断的主题
	FollowUpQuestion  bool            // 是否为追问
	TotalBudget       int             // 总预算字符数
	RecentBudget      int             // 近期上下文实际长度
	StructuredBudget  int             // 结构化上下文实际长度
}

// NewQuestionHistoryContext 构建提问历史上下文
func NewQuestionHistoryContext(recentQuestionTranscript string, maxChars int) *QuestionHistoryContext {
	totalBudget := max(maxChars, 1)
	historyContext := &QuestionHistoryContext{
		TotalBudget: totalBudget,
	}

	// 渲染近期用户提问记录（取尾部预算）
	userQuestions := renderRecentUserQuestions(recentQuestionTranscript, totalBudget)
	if utils.IsBlank(userQuestions) {
		return historyContext
	}

	historyContext.RenderedText = userQuestions
	historyContext.RecentContext = userQuestions
	historyContext.RecentBudget = utils.Len(userQuestions)

	return historyContext
}

// ApplyFollowUpAndEvidence 根据意图识别结果和证据锚点，补充跟进判断、结构化上下文及最终渲染文本
func (h *QuestionHistoryContext) ApplyFollowUpAndEvidence(question string, recognitionResult *IntentRecognitionResult, anchors EvidenceAnchors) {
	if h == nil {
		return
	}
	normalizedQuestion := utils.Trim(question)

	// 判断是否为跟进问题
	h.FollowUpQuestion = recognitionResult.IsFollowUpQuestion(normalizedQuestion)

	// 过滤证据锚点（最多5个）
	h.EvidenceAnchors = utils.FilterLimit(anchors, 5, func(anchor *EvidenceAnchor) bool {
		return anchor.HasAnchorIdentity()
	})

	// 若既不是跟进问题，又没有有效的用户问题和锚点，则无需补充内容
	if !h.FollowUpQuestion || (utils.IsBlank(h.RecentContext) && len(h.EvidenceAnchors) == 0) {
		return
	}

	// 计算结构化部分的预算（总预算减去已用 RecentContext 长度）
	structuredBudget := h.TotalBudget - h.RecentBudget
	h.StructuredContext = h.EvidenceAnchors.RenderStructuredContext(structuredBudget)
	h.StructuredBudget = utils.Len(h.StructuredContext)

	// 组装最终渲染文本（结构化内容 + 用户问题）
	renderedText := utils.JoinNonBlank("\n", h.StructuredContext, h.RecentContext)
	if renderedText != "" || len(h.EvidenceAnchors) > 0 {
		h.RenderedText = renderedText
		h.ResolvedTopic = h.EvidenceAnchors.ResolveTopic()
	}
}

// renderRecentUserQuestions 从原始转录中提取近期用户问题，并按预算渲染（含标题）
func renderRecentUserQuestions(rawTranscript string, budget int) string {
	if budget <= 0 || utils.IsBlank(rawTranscript) {
		return ""
	}

	// 提取用户问题部分（去除前缀，仅保留“用户：”行）
	normalized := utils.Trim(rawTranscript)
	prefixes := []string{"【最近相关提问】", "最近相关提问："}
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
	userQuestions := utils.Trim(builder.String())
	if userQuestions == "" {
		return ""
	}

	// 渲染（添加标题并截断）
	title := "对话承接上下文（仅用于理解指代，不作为事实证据）：\n"
	titleLen := utils.Len(title)
	if budget <= titleLen {
		// 预算不够标题，只返回裁剪后的用户问题
		return utils.ClipTail(userQuestions, budget)
	}
	body := utils.ClipTail(userQuestions, budget-titleLen)
	if body == "" {
		return ""
	}
	return title + body
}
