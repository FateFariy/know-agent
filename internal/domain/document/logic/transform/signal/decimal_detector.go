package signal

import (
	"regexp"
	"strings"

	"github.com/duke-git/lancet/v2/convertor"
	"github.com/duke-git/lancet/v2/strutil"

	"github.com/swiftbit/know-agent/internal/domain/document/model/vo"
)

var decimalHeadingPattern = regexp.MustCompile(`^(\d+(?:\.\d+)+)\s*[、.]?\s*(.+)$`)

type DecimalHeadingDetector struct{}

func (d *DecimalHeadingDetector) Name() string {
	return "decimal-heading"
}

func (d *DecimalHeadingDetector) Order() int {
	return 60
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

	levelHint := len(strings.Split(nodeCode, "."))
	if levelHint < 1 {
		levelHint = 1
	}

	return &vo.DocumentStructureSignal{
		Kind:        vo.SignalKindHeading,
		NodeCode:    nodeCode,
		Title:       title,
		LevelHint:   levelHint,
		NumericPath: d.extractNumericPath(nodeCode),
		Reasons:     []string{"decimal-heading"},
		Confidence:  0.95,
	}
}

func (d *DecimalHeadingDetector) extractNumericPath(code string) []int {
	normalized := strutil.Trim(code)
	if normalized == "" {
		return nil
	}
	if !strings.Contains(normalized, ".") {
		return nil
	}

	var path []int
	for _, segment := range strings.Split(normalized, ".") {
		if toInt, err := convertor.ToInt(segment); err == nil {
			path = append(path, int(toInt))
		} else {
			return nil
		}
	}
	return path
}
