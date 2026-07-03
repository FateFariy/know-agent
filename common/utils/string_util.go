package utils

import (
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/duke-git/lancet/v2/strutil"
)

func EqualsIgnoreCase(a, b string) bool {
	return strings.EqualFold(a, b)
}

func BlankToDefault(s string, defaultValue string) string {
	if s == "" {
		return defaultValue
	}
	return s
}

// ClipHead 截取头部
func ClipHead(text string, maxChars int) string {
	normalized := strutil.Trim(text)
	if utf8.RuneCountInString(normalized) <= maxChars {
		return normalized
	}
	if maxChars <= 1 {
		return ""
	}
	return string(normalized[:maxChars-1]) + "…"
}

// ClipTail 截取尾部
func ClipTail(text string, maxChars int) string {
	normalized := strutil.Trim(text)
	if utf8.RuneCountInString(normalized) <= maxChars {
		return normalized
	}
	if maxChars <= 1 {
		return ""
	}
	start := max(0, len(normalized)-maxChars+1)
	return "…" + string(normalized[start:])
}

// JoinNonBlank 连接非空字符串
func JoinNonBlank(left, right, delimiter string) string {
	left = strutil.Trim(left)
	right = strutil.Trim(right)
	if strutil.IsBlank(left) {
		return right
	}
	if strutil.IsBlank(right) {
		return left
	}
	return left + delimiter + right
}

// ParseChineseNumber 解析松散格式的数字（支持阿拉伯数字和中文数字）
func ParseChineseNumber(text string) int {
	normalized := strutil.Trim(text)
	if normalized == "" {
		return 0
	}

	// 优先尝试解析阿拉伯数字
	if toInt, err := strconv.Atoi(normalized); err == nil {
		return toInt
	}

	// 中文数字映射表
	digitMap := map[rune]int{
		'一': 1, '二': 2, '三': 3, '四': 4, '五': 5,
		'六': 6, '七': 7, '八': 8, '九': 9, '十': 10,
	}

	runeStr := []rune(normalized)
	// 处理中文数字：十、十五、二十、二十五
	if len(runeStr) == 2 && strings.HasPrefix(normalized, "十") {
		return 10 + digitMap[runeStr[1]]
	}
	if len(runeStr) == 2 && strings.HasSuffix(normalized, "十") {
		return digitMap[runeStr[0]] * 10
	}
	if len(runeStr) == 3 && strings.Contains(normalized, "十") {
		return digitMap[runeStr[0]]*10 + digitMap[runeStr[2]]
	}

	// 单个中文数字
	return digitMap[runeStr[0]]
}
