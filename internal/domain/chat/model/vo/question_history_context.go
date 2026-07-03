package vo

import (
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/duke-git/lancet/v2/strutil"

	"github.com/swiftbit/know-agent/common/utils"
)

// QuestionHistoryContext 提问历史上下文
type QuestionHistoryContext struct {
	RenderedText      string
	StructuredContext string
	RecentContext     string
	FollowUpQuestion  bool
	TotalBudget       int
	RecentBudget      int
	StructuredBudget  int
}

var followUpHints = []string{
	"刚才", "上面", "前面", "前文", "上一条", "上一个", "上一轮", "这个", "那个", "这条", "那条",
	"继续", "展开", "补充", "详细", "细说", "进一步", "为什么", "怎么做", "怎么理解", "还有呢",
}

var followUpPattern = regexp.MustCompile(`.*第\s*[0-9一二三四五六七八九十百]+\s*([条点项]).*`)

// NewQuestionHistoryContext 组装问题历史上下文, 仅当问题为续问类型（如"为什么"、"还有呢"等）且存在历史上下文时，才组装最近对话
func NewQuestionHistoryContext(question, recentQuestionTranscript string, questionHistoryMaxChars int) *QuestionHistoryContext {
	// 提取最近用户问题（过滤掉助手回答，只保留"用户："开头的行）
	recentUserContext := extractRecentUserQuestions(recentQuestionTranscript)

	// 判断当前问题是否为续问（包含"刚才"、"上面"、"为什么"等关键词）
	followUpQuestion := looksLikeFollowUpQuestion(strutil.Trim(question))

	// 初始化上下文对象
	questionHistoryContext := &QuestionHistoryContext{
		FollowUpQuestion: followUpQuestion,
		TotalBudget:      questionHistoryMaxChars,
	}

	// 非续问或无历史上下文时，直接返回（不需要组装）
	if !followUpQuestion || recentUserContext == "" {
		return questionHistoryContext
	}

	// 渲染最近上下文（添加标题并裁剪到预算长度）
	recentPart := renderRecentContext(recentUserContext, questionHistoryMaxChars)
	if recentPart == "" {
		return questionHistoryContext
	}

	// 填充上下文详情
	questionHistoryContext.RecentContext = recentPart
	questionHistoryContext.RecentBudget = questionHistoryMaxChars
	questionHistoryContext.FollowUpQuestion = followUpQuestion

	return questionHistoryContext
}

// 提取最近用户问题
func extractRecentUserQuestions(recentQuestionTranscript string) string {
	normalized := strutil.Trim(recentQuestionTranscript)

	if strings.HasPrefix(normalized, "【最近相关对话】") {
		normalized = strutil.Trim(normalized[len("【最近相关对话】"):])
	}
	if strings.HasPrefix(normalized, "最近相关对话：") {
		normalized = strutil.Trim(normalized[len("最近相关对话："):])
	}

	var builder strings.Builder
	lines := strings.Split(normalized, "\n")
	for _, line := range lines {
		trimmed := strutil.Trim(line)
		if !strings.HasPrefix(trimmed, "用户：") {
			continue
		}
		if builder.Len() > 0 {
			builder.WriteString("\n")
		}
		builder.WriteString(trimmed)
	}

	return strutil.Trim(builder.String())
}

// 判断问题是否为续问
func looksLikeFollowUpQuestion(question string) bool {
	if question == "" {
		return false
	}

	questionLen := utf8.RuneCountInString(question)
	if strutil.ContainsAny(question, followUpHints) || followUpPattern.MatchString(question) || questionLen <= 12 {
		return true
	}

	return questionLen <= 18 && (strings.HasSuffix(question, "呢") || strings.HasSuffix(question, "吗"))
}

// 渲染最近用户问题
func renderRecentContext(recentUserContext string, budget int) string {
	title := "对话承接上下文（仅用于理解指代，不作为事实证据）：\n"
	titleLen := utf8.RuneCountInString(title)
	if budget <= titleLen {
		return utils.ClipTail(recentUserContext, budget)
	}

	body := utils.ClipTail(recentUserContext, budget-titleLen)
	if body == "" {
		return ""
	}

	return title + body
}
