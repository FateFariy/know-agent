package support

import (
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/duke-git/lancet/v2/strutil"

	"github.com/swiftbit/know-agent/internal/domain/document/model/vo"
)

var (
	titleHashPrefixRegex = regexp.MustCompile(`^#+\s*`)              // Markdown 标题前缀（如 ###）
	titleExtRegex        = regexp.MustCompile(`\.[A-Za-z0-9]{1,6}$`) // 文件扩展名
	titleSpaceRegex      = regexp.MustCompile(`\s+`)                 // 空白字符
	tableBorderRegex     = regexp.MustCompile(`^[\-=_]{3,}$`)        // 表格分割线（如 ----）
	nonContentRegex      = regexp.MustCompile(`^[:\-\\s|]+$`)        // 非内容行（纯分隔符）
)

// NormalizeComparableTitle 标准化标题用于比较, 去除 Markdown 前缀、文件扩展名、空白字符，转为小写
func NormalizeComparableTitle(text string) string {
	normalized := strutil.Trim(text)
	if normalized == "" {
		return ""
	}
	normalized = titleHashPrefixRegex.ReplaceAllString(normalized, "")
	normalized = titleExtRegex.ReplaceAllString(normalized, "")
	normalized = titleSpaceRegex.ReplaceAllString(normalized, "")
	return strings.ToLower(normalized)
}

// LooksLikePlainHeading 启发式判断一行是否像「朴素标题」（无编号/无符号的纯文字标题）
/*
  判断要点：
  1. 文本非空且字符数 ≤ maxPlainHeadingChars
  2. 不以句末标点结尾（排除完整句子）
  3. 不含 http(s):// 前缀（排除链接行）
  4. 不以 | 开头或结尾（排除表格行）
  5. 不是纯分割线
  6. 上下文判断：前后至少一侧有空行，且下一行看起来像正文内容
  7. 名词性特征：不含内部中英文逗号/分号/句号等
*/
func LooksLikePlainHeading(lineContext *vo.LineContext, text string, maxPlainHeadingChars int) bool {
	normalized := strutil.Trim(text)
	if normalized == "" {
		return false
	}

	charLen := utf8.RuneCountInString(normalized)
	// 长度超过阈值 → 不是标题
	if charLen > maxPlainHeadingChars {
		return false
	}

	// 以句子结束标点结尾 → 不是标题
	if strings.ContainsAny(normalized[charLen-1:], "。！？；.!?;") {
		return false
	}

	// 包含 URL 前缀 → 不是标题
	lower := strings.ToLower(normalized)
	if strings.Contains(lower, "http://") || strings.Contains(lower, "https://") {
		return false
	}

	// 以 | 开头或结尾 → 表格内容
	if strings.HasPrefix(normalized, "|") || strings.HasSuffix(normalized, "|") {
		return false
	}

	// 纯分割线（如 ====, ----, ____）→ 不是标题
	if tableBorderRegex.MatchString(normalized) {
		return false
	}

	// 上下文判断：前后有空白行且下一行有内容
	isolated := lineContext.BlankBefore || lineContext.BlankAfter

	nextLooksContent := lineContext.NextNonBlank != nil &&
		strutil.IsNotBlank(lineContext.NextNonBlank.NormalizedText) &&
		!nonContentRegex.MatchString(lineContext.NextNonBlank.NormalizedText)

	// 名词性特征：不含内部标点（，；。：）
	nounLike := !strings.ContainsAny(normalized, "，；。：") && !strings.HasPrefix(lower, "http")

	return isolated && nextLooksContent && nounLike
}
