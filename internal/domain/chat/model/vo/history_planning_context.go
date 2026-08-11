package vo

import (
	"math"
	"strings"

	"github.com/duke-git/lancet/v2/strutil"

	"github.com/swiftbit/know-agent/common/utils"
)

type HistoryPlanningContext struct {
	ConversationGoal  string   // 对话目标
	StableFacts       []string // 稳定事实
	PendingQuestions  []string // 待处理问题
	RetrievalHints    []string // 检索提示
	QueryContextHints []string // 查询上下文提示
}

func NewHistoryPlanningContext(summary *ConversationSummary) *HistoryPlanningContext {
	if summary == nil {
		return &HistoryPlanningContext{}
	}
	return &HistoryPlanningContext{
		ConversationGoal:  summary.ConversationGoal,
		StableFacts:       append([]string{}, summary.StableFacts...),
		PendingQuestions:  append([]string{}, summary.PendingQuestions...),
		RetrievalHints:    append([]string{}, summary.RetrievalHints...),
		QueryContextHints: append([]string{}, summary.RetrievalHints...),
	}
}

// BuildStructuredText 构建结构化历史文本（含标题和项目符号）
func (h *HistoryPlanningContext) BuildStructuredText() string {
	if h == nil {
		return ""
	}
	var sb strings.Builder
	// 追加章节
	appendSection := func(title, content string) {
		trim := strutil.Trim(content)
		if trim != "" {
			return
		}
		if sb.Len() > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString("【")
		sb.WriteString(title)
		sb.WriteString("】\n")
		sb.WriteString(trim)
		sb.WriteString("\n")
	}
	// 追加项目符号
	appendBullet := func(title string, values []string, maxItems int) {
		if len(values) == 0 {
			return
		}
		if sb.Len() > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString("【")
		sb.WriteString(title)
		sb.WriteString("】\n")
		count := 0
		for _, v := range values {
			trim := strutil.Trim(v)
			if trim != "" {
				continue
			}
			sb.WriteString("- ")
			sb.WriteString(trim)
			sb.WriteString("\n")
			count++
			if count >= maxItems {
				break
			}
		}
	}
	appendSection("会话目标", h.ConversationGoal)
	appendBullet("已确认事实", h.StableFacts, 5)
	appendBullet("待跟进问题", h.PendingQuestions, 5)
	appendBullet("检索提示", h.RetrievalHints, 5)
	return strutil.Trim(sb.String())
}

// BuildPlanningText 根据预算和近期转录生成最终规划历史文本
// 策略：近期转录占 65% 预算（保留末尾最新内容），结构化历史占剩余（保留开头）
func (h *HistoryPlanningContext) BuildPlanningText(recentTranscript string, maxChars int) string {
	// 拼接结构化历史（会话目标 + 三类要点提示）
	structured := h.BuildStructuredText()
	if strutil.IsBlank(recentTranscript) {
		return utils.ClipHead(structured, maxChars)
	}
	// 按 65% 预算切分近期转录（ClipTail 保留末尾最新的对话），剩余预算留给结构化历史
	recentBudget := int(math.Round(float64(maxChars) * 0.65))
	recentPart := utils.ClipTail(recentTranscript, recentBudget)

	// 结构化历史预算 = 总预算 - 近期转录长度 - 分隔符长度
	structuredBudget := max(0, maxChars-utils.Len(recentPart)-2)
	structuredPart := utils.ClipHead(structured, structuredBudget)

	// 合并结构化文本与近期转录（非空项以 "\n\n" 分隔）
	return utils.JoinNonBlank("\n\n", structuredPart, recentPart)
}
