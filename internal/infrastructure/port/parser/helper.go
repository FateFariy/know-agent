package parser

import (
	"bytes"
	"regexp"
	"strings"
	"unicode/utf8"

	"golang.org/x/net/html/charset"

	"github.com/swiftbit/know-agent/common/utils"
)

var (
	// 匹配连续空行：换行 + 任意空白 + 至少一个换行
	emptyLinesRegex = regexp.MustCompile(`\n\s*\n+`)

	// 匹配文档序号开头：中文序号、第X章节条、多级数字编号
	headingPrefixRegex = regexp.MustCompile(`^([一二三四五六七八九十]+、|第[一二三四五六七八九十0-9]+[章节条]|[0-9]+(\.[0-9]+)*[、.])`)

	// 匹配标题开头：# 开头，1-6 个 # 号
	titleRegex = regexp.MustCompile(`^#{1,6}\s+`)
)

// decodeText decodes bytes to string, trying UTF-8, UTF-8-SIG, then GB18030
func decodeText(content []byte) string {
	// Try UTF-8-SIG (with BOM)
	if len(content) >= 3 && content[0] == 0xEF && content[1] == 0xBB && content[2] == 0xBF {
		return string(content[3:])
	}
	// Try UTF-8
	if utf8.Valid(content) {
		return string(content)
	}
	// Try GB18030
	reader, err := charset.NewReaderLabel("gb18030", bytes.NewReader(content))
	if err == nil {
		var buf bytes.Buffer
		_, err = buf.ReadFrom(reader)
		if err == nil {
			return buf.String()
		}
	}
	// Fallback
	return string(content)
}

// classifyTextBlock 判断文本块类型
func classifyTextBlock(text string) string {
	stripped := strings.TrimSpace(text)
	if stripped == "" {
		return "Text"
	}
	if utils.Len(stripped) <= 80 {
		// 正则匹配中文序号、章节条、数字序号
		if headingPrefixRegex.MatchString(stripped) {
			return "TITLE"
		}
		// 以章、节、：、: 结尾
		if strings.HasSuffix(stripped, "章") ||
			strings.HasSuffix(stripped, "节") ||
			strings.HasSuffix(stripped, "：") ||
			strings.HasSuffix(stripped, ":") {
			return "TITLE"
		}
	}
	return "Text"
}
