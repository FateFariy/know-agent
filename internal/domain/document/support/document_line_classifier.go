package support

import (
	"regexp"
	"strings"
)

const (
	lineKindHeading LineKind = iota
	lineKindListItem
	lineKindBody
)

type LineKind int

func (k LineKind) IsHeading() bool {
	return k == lineKindHeading
}

func (k LineKind) IsListItem() bool {
	return k == lineKindListItem
}

type LineClassification struct {
	Kind    LineKind
	Level   int
	Title   string
	RawText string
}

func (c LineClassification) IsHeading() bool {
	return c.Kind.IsHeading()
}

func (c LineClassification) IsListItem() bool {
	return c.Kind.IsListItem()
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

type DocumentLineClassifier struct{}

func NewDocumentLineClassifier() *DocumentLineClassifier {
	return &DocumentLineClassifier{}
}

func (c *DocumentLineClassifier) Classify(line string) LineClassification {
	normalized := safeText(line)
	if normalized == "" {
		return newLineClassification(lineKindBody, 0, normalized, normalized)
	}

	if matches := markdownHeadingPattern.FindStringSubmatch(normalized); len(matches) == 3 {
		level := len(matches[1])
		return c.heading(level, safeText(matches[2]), normalized)
	}

	if matches := appendixPattern.FindStringSubmatch(normalized); len(matches) >= 1 {
		return c.heading(1, normalized, normalized)
	}

	if matches := explicitStepPattern.FindStringSubmatch(normalized); len(matches) >= 1 {
		return c.listItem(normalized)
	}

	if matches := chineseChapterPattern.FindStringSubmatch(normalized); len(matches) == 3 {
		return c.heading(2, normalized, normalized)
	}

	if matches := multiLevelDigitHeadingPattern.FindStringSubmatch(normalized); len(matches) == 3 {
		prefix := matches[1]
		level := len(strings.Split(prefix, "."))
		return c.heading(level, normalized, normalized)
	}

	if matches := chineseOutlinePattern.FindStringSubmatch(normalized); len(matches) == 3 {
		content := safeText(matches[2])
		if c.looksLikeHeadingContent(content) {
			return c.heading(1, normalized, normalized)
		}
		return c.listItem(normalized)
	}

	if matches := singleLevelDigitLinePattern.FindStringSubmatch(normalized); len(matches) == 3 {
		content := safeText(matches[2])
		if c.looksLikeHeadingContent(content) {
			return c.heading(1, normalized, normalized)
		}
		return c.listItem(normalized)
	}

	if strings.HasPrefix(normalized, "- ") ||
		strings.HasPrefix(normalized, "* ") ||
		strings.HasPrefix(normalized, "+ ") ||
		strings.HasPrefix(normalized, "- [") ||
		strings.HasPrefix(normalized, "* [") ||
		strings.HasPrefix(normalized, "+ [") {
		return c.listItem(normalized)
	}

	return newLineClassification(lineKindBody, 0, normalized, normalized)
}

func (c *DocumentLineClassifier) heading(level int, title, rawText string) LineClassification {
	if level < 1 {
		level = 1
	}
	return newLineClassification(lineKindHeading, level, safeText(title), safeText(rawText))
}

func (c *DocumentLineClassifier) listItem(rawText string) LineClassification {
	return newLineClassification(lineKindListItem, 0, safeText(rawText), safeText(rawText))
}

func (c *DocumentLineClassifier) looksLikeHeadingContent(content string) bool {
	normalized := safeText(content)
	if normalized == "" {
		return false
	}

	if c.endsWithSentencePunctuation(normalized) {
		return false
	}
	if len(normalized) > 24 {
		return false
	}
	return !strings.Contains(normalized, "，") &&
		!strings.Contains(normalized, "；") &&
		!strings.Contains(normalized, "。") &&
		!strings.Contains(normalized, "：")
}

func (c *DocumentLineClassifier) endsWithSentencePunctuation(text string) bool {
	return strings.HasSuffix(text, "。") ||
		strings.HasSuffix(text, "！") ||
		strings.HasSuffix(text, "？") ||
		strings.HasSuffix(text, "；") ||
		strings.HasSuffix(text, ".") ||
		strings.HasSuffix(text, "!") ||
		strings.HasSuffix(text, "?") ||
		strings.HasSuffix(text, ";")
}

func safeText(text string) string {
	if text == "" {
		return ""
	}
	return strings.TrimSpace(text)
}

func newLineClassification(kind LineKind, level int, title, rawText string) LineClassification {
	return LineClassification{
		Kind:    kind,
		Level:   level,
		Title:   title,
		RawText: rawText,
	}
}
