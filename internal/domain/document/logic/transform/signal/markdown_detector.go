package signal

import (
	"regexp"

	"github.com/duke-git/lancet/v2/strutil"

	"github.com/swiftbit/know-agent/internal/domain/document/model/vo"
)

var markdownHeadingPattern = regexp.MustCompile(`^(#+)\s+(.+)$`)

type MarkdownHeadingDetector struct {
	BaseDetector
}

func (d *MarkdownHeadingDetector) Detect(detCtx *DetectorContext, text string) *vo.DocumentStructureSignal {
	if text == "" {
		return nil
	}

	matches := markdownHeadingPattern.FindStringSubmatch(text)
	if len(matches) != 3 {
		return nil
	}

	title := strutil.Trim(matches[2])
	if d.sameDocumentTitle(detCtx.DocumentTitle, title) {
		return &vo.DocumentStructureSignal{
			Kind:       vo.SignalKindNoise,
			Title:      title,
			Reasons:    []string{"duplicate-document-title"},
			Confidence: 0.99,
		}
	}

	nodeCode := d.extractCode(title)
	return &vo.DocumentStructureSignal{
		Kind:        vo.SignalKindHeading,
		NodeCode:    nodeCode,
		Title:       title,
		LevelHint:   len(matches[1]),
		NumericPath: d.extractNumericPath(nodeCode),
		Reasons:     []string{"markdown-heading"},
		Confidence:  0.98,
	}
}

func (d *MarkdownHeadingDetector) Order() int {
	return 20
}
