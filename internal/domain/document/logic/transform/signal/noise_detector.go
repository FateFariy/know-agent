package signal

import (
	"regexp"
	"strings"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/document/model/vo"
)

var (
	pageNoisePattern      = regexp.MustCompile(`^(?:第\s*\d+\s*页|Page\s*\d+|\d+\s*/\s*\d+)$`)
	copyrightNoisePattern = regexp.MustCompile(`.*(?:版权所有|未经授权|内部使用|copyright|all rights reserved|保密).*`)
	versionFooterPattern  = regexp.MustCompile(`.*(?:\bV\d+(?:\.\d+)*\b|版本|修订|Rev\.?\s*\d+).*`)
)

// NoiseDetector 噪声检测器
type NoiseDetector struct {
	BaseDetector
}

func (d *NoiseDetector) Detect(detCtx *DetectorContext, text string) *vo.DocumentStructureSignal {
	if text == "" {
		return nil
	}

	frequency := utils.Ternary(detCtx.LineFrequency == nil, 0, detCtx.LineFrequency[text])
	if frequency >= 2 {
		noise := &vo.DocumentStructureSignal{Kind: vo.SignalKindNoise, Confidence: 0.99}
		if d.sameDocumentTitle(detCtx.DocumentTitle, text) {
			noise.Reasons = []string{"duplicate-document-title"}
			return noise
		}

		if copyrightNoisePattern.MatchString(text) {
			noise.Reasons = []string{"copyright-noise"}
			return noise
		}

		if frequency >= 3 && len(text) <= 120 && (versionFooterPattern.MatchString(text) || strings.Contains(text, "|")) {
			noise.Reasons = []string{"version-footer-noise"}
			return noise
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

func (d *NoiseDetector) Order() int {
	return 10
}
