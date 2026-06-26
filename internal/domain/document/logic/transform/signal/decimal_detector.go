package signal

import (
	"regexp"
	"strings"

	"github.com/duke-git/lancet/v2/strutil"

	"github.com/swiftbit/know-agent/internal/domain/document/model/vo"
)

var decimalHeadingPattern = regexp.MustCompile(`^(\d+(?:\.\d+)+)\s*[、.]?\s*(.+)$`)

type DecimalHeadingDetector struct {
	BaseDetector
}

func (d *DecimalHeadingDetector) Detect(detCtx *DetectorContext, text string) *vo.DocumentStructureSignal {
	if text == "" {
		return nil
	}

	matches := decimalHeadingPattern.FindStringSubmatch(text)
	if len(matches) != 3 {
		return nil
	}

	nodeCode := strutil.Trim(matches[1])
	title := strutil.Trim(matches[2])

	return &vo.DocumentStructureSignal{
		Kind:        vo.SignalKindHeading,
		NodeCode:    nodeCode,
		Title:       title,
		LevelHint:   max(len(strings.Split(nodeCode, ".")), 1),
		NumericPath: d.extractNumericPath(nodeCode),
		Reasons:     []string{"decimal-heading"},
		Confidence:  0.95,
	}
}

func (d *DecimalHeadingDetector) Order() int {
	return 60
}
