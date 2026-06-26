package signal

import (
	"regexp"
	"strings"

	"github.com/duke-git/lancet/v2/strutil"

	"github.com/swiftbit/know-agent/internal/domain/document/model/vo"
)

var (
	pageNoisePattern      = regexp.MustCompile(`^(?:第\s*\d+\s*页|Page\s*\d+|\d+\s*/\s*\d+)$`)
	copyrightNoisePattern = regexp.MustCompile(`.*(?:版权所有|未经授权|内部使用|copyright|all rights reserved|保密).*`)
	versionFooterPattern  = regexp.MustCompile(`.*(?:\bV\d+(?:\.\d+)*\b|版本|修订|Rev\.?\s*\d+).*`)
)

type NoiseDetector struct{}

func (d *NoiseDetector) Name() string {
	return "noise"
}

func (d *NoiseDetector) Order() int {
	return 10
}

func (d *NoiseDetector) Detect(detCtx *DetectorContext, text string) *vo.DocumentStructureSignal {
	if detCtx == nil || text == "" {
		return nil
	}

	frequency := 0
	if detCtx.LineFrequency != nil {
		frequency = detCtx.LineFrequency[text]
	}

	if frequency >= 2 {
		if d.sameDocumentTitle(detCtx.DocumentTitle, text) {
			return &vo.DocumentStructureSignal{
				Kind:       vo.SignalKindNoise,
				Reasons:    []string{"duplicate-document-title"},
				Confidence: 0.99,
			}
		}

		if copyrightNoisePattern.MatchString(text) {
			return &vo.DocumentStructureSignal{
				Kind:       vo.SignalKindNoise,
				Reasons:    []string{"copyright-noise"},
				Confidence: 0.99,
			}
		}

		if frequency >= 3 && len(text) <= 120 && (versionFooterPattern.MatchString(text) || strings.Contains(text, "|")) {
			return &vo.DocumentStructureSignal{
				Kind:       vo.SignalKindNoise,
				Reasons:    []string{"version-footer-noise"},
				Confidence: 0.99,
			}
		}
	}

	if pageNoisePattern.MatchString(text) {
		return &vo.DocumentStructureSignal{
			Kind:       vo.SignalKindNoise,
			Reasons:    []string{"page-noise"},
			Confidence: 0.98,
		}
	}

	return nil
}

func (d *NoiseDetector) sameDocumentTitle(documentTitle, candidate string) bool {
	if documentTitle == "" || candidate == "" {
		return false
	}
	left := d.normalizeComparableTitle(documentTitle)
	right := d.normalizeComparableTitle(candidate)
	return left == right
}

func (d *NoiseDetector) normalizeComparableTitle(text string) string {
	normalized := strutil.Trim(text)
	if normalized == "" {
		return ""
	}
	normalized = regexp.MustCompile(`^#+\s*`).ReplaceAllString(normalized, "")
	normalized = regexp.MustCompile(`\.[A-Za-z0-9]{1,6}$`).ReplaceAllString(normalized, "")
	normalized = regexp.MustCompile(`\s+`).ReplaceAllString(normalized, "")
	return strings.ToLower(normalized)
}
