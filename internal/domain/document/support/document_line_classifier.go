package support

import (
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/duke-git/lancet/v2/strutil"
)

type LineKind = int

const (
	lineKindHeading LineKind = iota
	lineKindListItem
	lineKindBody
)

type LineClassification struct {
	Kind    LineKind
	Level   int
	Title   string
	RawText string
}

func newLineClassification(kind LineKind, level int, title, rawText string) *LineClassification {
	return &LineClassification{
		Kind:    kind,
		Level:   level,
		Title:   title,
		RawText: rawText,
	}
}

func (c LineClassification) IsHeading() bool {
	return c.Kind == lineKindHeading
}

func (c LineClassification) IsListItem() bool {
	return c.Kind == lineKindListItem
}

var (
	markdownHeadingPattern        = regexp.MustCompile(`^(#{1,6})\s+(.+)$`)
	multiLevelDigitHeadingPattern = regexp.MustCompile(`^(\d+(?:\.\d+)+)\s*[、.]?\s*(.+)$`)
	singleLevelDigitLinePattern   = regexp.MustCompile(`^(\d+)\s*[、.]\s*(.+)$`)
	chineseChapterPattern         = regexp.MustCompile(`^(第[一二三四五六七八九十百\d]+[章节条部分])\s*(.+)$`)
	chineseOutlinePattern         = regexp.MustCompile(`^([一二三四五六七八九十百]+)[、.]\s*(.+)$`)
	appendixPattern               = regexp.MustCompile(`^(附录\s*[A-Za-z一二三四五六七八九十百\d]+)(?:\s+(.+))?$`)
	explicitStepPattern           = regexp.MustCompile(`^(?:第\s*([0-9一二三四五六七八九十百]+)\s*步|步骤\s*([0-9一二三四五六七八九十百]+))\s*[:：、.]?\s*(.+)$`)
)

const (
	maxHeadingLength = 24 // 标题最大长度
)

type DocumentLineClassifier struct{}

func (c *DocumentLineClassifier) Classify(line string) *LineClassification {
	normalized := strutil.Trim(line)
	if normalized == "" {
		return newLineClassification(lineKindBody, 0, normalized, normalized)
	}

	if matches := markdownHeadingPattern.FindStringSubmatch(normalized); len(matches) > 3 {
		level := len(matches[1])
		return c.heading(level, matches[2], normalized)
	}

	if matches := appendixPattern.FindStringSubmatch(normalized); len(matches) > 0 {
		return c.heading(1, normalized, normalized)
	}

	if matches := explicitStepPattern.FindStringSubmatch(normalized); len(matches) > 0 {
		return c.listItem(normalized)
	}

	if matches := chineseChapterPattern.FindStringSubmatch(normalized); len(matches) > 0 {
		return c.heading(2, normalized, normalized)
	}

	if matches := multiLevelDigitHeadingPattern.FindStringSubmatch(normalized); len(matches) > 1 {
		prefix := matches[1]
		level := len(strings.Split(prefix, "."))
		return c.heading(level, normalized, normalized)
	}

	if matches := chineseOutlinePattern.FindStringSubmatch(normalized); len(matches) > 3 {
		content := strutil.Trim(matches[2])
		if c.looksLikeHeadingContent(content) {
			return c.heading(1, normalized, normalized)
		}
		return c.listItem(normalized)
	}

	if matches := singleLevelDigitLinePattern.FindStringSubmatch(normalized); len(matches) > 3 {
		content := strutil.Trim(matches[2])
		if c.looksLikeHeadingContent(content) {
			return c.heading(1, normalized, normalized)
		}
		return c.listItem(normalized)
	}
	if strutil.HasPrefixAny(normalized, []string{"- ", "* ", "+ ", "- [", "* [", "+ ["}) {
		return c.listItem(normalized)
	}

	return newLineClassification(lineKindBody, 0, normalized, normalized)
}

func (c *DocumentLineClassifier) heading(level int, title, rawText string) *LineClassification {
	return newLineClassification(lineKindHeading, max(1, level), strutil.Trim(title), strutil.Trim(rawText))
}

func (c *DocumentLineClassifier) listItem(rawText string) *LineClassification {
	return newLineClassification(lineKindListItem, 0, strutil.Trim(rawText), strutil.Trim(rawText))
}

func (c *DocumentLineClassifier) looksLikeHeadingContent(content string) bool {
	normalized := strutil.Trim(content)
	if normalized == "" {
		return false
	}

	charLen := utf8.RuneCountInString(normalized)
	if strings.ContainsAny(normalized[charLen-1:], "。！？；.!?;") || charLen > maxHeadingLength {
		return false
	}
	return !strings.ContainsAny(normalized, "，；。：")
}
