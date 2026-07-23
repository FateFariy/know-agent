package utils

import (
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/duke-git/lancet/v2/convertor"
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
	if Len(normalized) <= maxChars {
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
	if Len(normalized) <= maxChars {
		return normalized
	}
	if maxChars <= 1 {
		return ""
	}
	start := max(0, len(normalized)-maxChars+1)
	return "…" + string(normalized[start:])
}

// JoinNonBlank 连接非空字符串
func JoinNonBlank(sep string, parts ...string) string {
	result := make([]string, 0, len(parts))
	for i := 0; i < len(parts); i++ {
		if strutil.IsNotBlank(parts[i]) {
			result = append(result, strutil.Trim(parts[i]))
		}
	}
	return strings.Join(result, sep)
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

// Join 连接字符串
func Join[T any](values []T, prefix, suffix, sep string) string {
	var sb strings.Builder
	sb.WriteString(prefix)
	for i, id := range values {
		if i > 0 {
			sb.WriteString(sep)
		}
		sb.WriteString(convertor.ToString(id))
	}
	sb.WriteString(suffix)
	return sb.String()
}

var (
	englishPattern = regexp.MustCompile(`[A-Za-z]`) // 匹配英文字母
)

func EstimateTokens(content string) int {
	englishCount, chineseCount := 0, 0

	// 统计英文单词数量
	for _, word := range strings.Fields(content) {
		if englishPattern.MatchString(word) {
			englishCount++
		}
	}

	// 统计中文字符数量
	for _, r := range content {
		if unicode.Is(unicode.Han, r) {
			chineseCount++
		}
	}

	// 非中英文字符按 4 字符折算 1 Token
	baseToken := max(1, (Len(content)-chineseCount-englishCount)/4)

	return chineseCount + englishCount + baseToken
}

func Len(str string) int {
	return utf8.RuneCountInString(str)
}
