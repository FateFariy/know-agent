package utils

import (
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
