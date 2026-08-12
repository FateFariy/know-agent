package vo

import (
	"regexp"
	"slices"
	"strings"

	"github.com/duke-git/lancet/v2/strutil"

	"github.com/swiftbit/know-agent/common/utils"
)

var (
	numberedMultiQuestionPattern = regexp.MustCompile(`(^|\s)(\d+[)\.、]|[A-Za-z][)])`)
	multiLinePattern             = regexp.MustCompile(`\n+`)
	splitPattern                 = regexp.MustCompile(`[?？；;\n]+`)
)

// OriginalQuestion 原始问题值对象，封装问题文本及历史上下文，提供多问题判定、改写必要性判断和规则拆分能力
type OriginalQuestion struct {
	question       string // 原始问题
	historySummary string // 历史摘要（用于判断是否需要改写）
}

// NewOriginalQuestion 创建原始问题值对象
func NewOriginalQuestion(question, historySummary string) *OriginalQuestion {
	return &OriginalQuestion{
		question:       strutil.Trim(question),
		historySummary: strutil.Trim(historySummary),
	}
}

// Question 返回原始问题
func (oq *OriginalQuestion) Question() string {
	return oq.question
}

// HistorySummary 返回历史摘要
func (oq *OriginalQuestion) HistorySummary() string {
	return oq.historySummary
}

// IsBlank 判断原始问题是否为空
func (oq *OriginalQuestion) IsBlank() bool {
	return oq.question == ""
}

// IsExplicitMultiQuestion 判断是否为显式多问题
//
// 判定依据（满足任一即视为多问题）：
//   - 问号（?？）数量 ≥ 2
//   - 包含分号（；;）
//   - 存在多个非空行（\n 分隔）
//   - 匹配编号列表（如 1. 2. 或 a) b)）
//   - 包含“分别”
func (oq *OriginalQuestion) IsExplicitMultiQuestion() bool {
	if oq.question == "" {
		return false
	}

	questionMarkCount := strings.Count(oq.question, "?") + strings.Count(oq.question, "？")
	if questionMarkCount >= 2 {
		return true
	}

	if strings.Contains(oq.question, "；") || strings.Contains(oq.question, ";") {
		return true
	}

	if multiLinePattern.MatchString(oq.question) {
		nonBlankLines := slices.DeleteFunc(strings.Split(oq.question, "\n"), func(item string) bool {
			return strutil.IsBlank(item)
		})
		if len(nonBlankLines) >= 2 {
			return true
		}
	}

	if numberedMultiQuestionPattern.MatchString(oq.question) {
		return true
	}

	return strings.Contains(oq.question, "分别")
}

// NeedsRewrite 判断是否需要 LLM 改写
//
// 规则：
//   - 无历史时：问题长度 < shortLen 或为显式多问题
//   - 有历史时：问题长度 < longLen 或为显式多问题
func (oq *OriginalQuestion) NeedsRewrite(shortLen, longLen int) bool {
	if oq.IsBlank() {
		return false
	}
	if oq.historySummary == "" {
		return utils.Len(oq.question) < shortLen || oq.IsExplicitMultiQuestion()
	}
	return utils.Len(oq.question) < longLen || oq.IsExplicitMultiQuestion()
}

// SplitByRules 基于规则拆分问题（按 ?？；;\n 分割）
func (oq *OriginalQuestion) SplitByRules(maxSubQuestions int) []string {
	parts := splitPattern.Split(oq.question, -1)
	result := utils.FilterMapUniqueLimit(parts, maxSubQuestions, func(item string) (string, string, bool) {
		trim := strings.TrimSpace(item)
		return trim, trim, trim != ""
	})

	if len(result) == 0 {
		return []string{oq.question}
	}
	return result
}
