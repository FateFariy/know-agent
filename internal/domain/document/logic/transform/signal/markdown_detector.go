package signal

import (
	"regexp"
	"strings"

	"github.com/duke-git/lancet/v2/strutil"

	"github.com/swiftbit/know-agent/internal/domain/document/model/vo"
)

var markdownHeadingPattern = regexp.MustCompile(`^(#+)\s+(.+)$`)

type MarkdownHeadingDetector struct{}

func (d *MarkdownHeadingDetector) Name() string {
	return "markdown-heading"
}

func (d *MarkdownHeadingDetector) Order() int {
	return 20
}

func (d *MarkdownHeadingDetector) Detect(detCtx *DetectorContext, text string) *vo.DocumentStructureSignal {
	if text == "" {
		return nil
	}

	matches := markdownHeadingPattern.FindStringSubmatch(text)
	if len(matches) != 3 {
		return nil
	}

	title := strings.TrimSpace(matches[2])
	if d.sameDocumentTitle(detCtx.DocumentTitle, title) {
		return &vo.DocumentStructureSignal{
			Kind:       vo.SignalKindNoise,
			Title:      title,
			Reasons:    []string{"duplicate-document-title"},
			Confidence: 0.99,
		}
	}

	return &vo.DocumentStructureSignal{
		Kind:        vo.SignalKindHeading,
		NodeCode:    d.extractCode(title),
		Title:       title,
		LevelHint:   len(matches[1]),
		NumericPath: d.extractNumericPath(title),
		Reasons:     []string{"markdown-heading"},
		Confidence:  0.98,
	}
}

func (d *MarkdownHeadingDetector) sameDocumentTitle(documentTitle, candidate string) bool {
	if documentTitle == "" || candidate == "" {
		return false
	}
	left := d.normalizeComparableTitle(documentTitle)
	right := d.normalizeComparableTitle(candidate)
	return left == right
}

func (d *MarkdownHeadingDetector) normalizeComparableTitle(text string) string {
	normalized := strutil.Trim(text)
	if normalized == "" {
		return ""
	}
	normalized = regexp.MustCompile(`^#+\s*`).ReplaceAllString(normalized, "")
	normalized = regexp.MustCompile(`\.[A-Za-z0-9]{1,6}$`).ReplaceAllString(normalized, "")
	normalized = regexp.MustCompile(`\s+`).ReplaceAllString(normalized, "")
	return strings.ToLower(normalized)
}

func (d *MarkdownHeadingDetector) extractCode(title string) string {
	decimalPattern := regexp.MustCompile(`^(\d+(?:\.\d+)+)\s*[、.]?\s*(.+)$`)
	if matches := decimalPattern.FindStringSubmatch(title); len(matches) == 3 {
		return strutil.Trim(matches[1])
	}
	return ""
}

func (d *MarkdownHeadingDetector) extractNumericPath(text string) []int {
	normalized := strutil.Trim(text)
	if normalized == "" {
		return nil
	}
	if strings.Contains(normalized, ".") {
		var path []int
		for _, segment := range strings.Split(normalized, ".") {
			for _, c := range segment {
				if c < '0' || c > '9' {
					return nil
				}
			}
			var num int
			for _, c := range segment {
				num = num*10 + int(c-'0')
			}
			path = append(path, num)
		}
		return path
	}
	return nil
}
