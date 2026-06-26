package signal

import (
	"regexp"

	"github.com/duke-git/lancet/v2/strutil"

	"github.com/swiftbit/know-agent/internal/domain/document/model/vo"
)

var (
	checkboxPattern = regexp.MustCompile(`^\[[ xX]]\s+(.+)$`)
	bulletPattern   = regexp.MustCompile(`^([-*+•])\s+(.+)$`)
)

type ListItemDetector struct{}

func (d *ListItemDetector) Detect(detCtx *DetectorContext, text string) *vo.DocumentStructureSignal {
	if text == "" {
		return nil
	}

	if matches := checkboxPattern.FindStringSubmatch(text); len(matches) == 2 {
		return &vo.DocumentStructureSignal{
			Kind:       vo.SignalKindListItem,
			Title:      strutil.Trim(matches[1]),
			Reasons:    []string{"checkbox-list"},
			Confidence: 0.92,
		}
	}

	if matches := bulletPattern.FindStringSubmatch(text); len(matches) == 3 {
		return &vo.DocumentStructureSignal{
			Kind:       vo.SignalKindListItem,
			Title:      strutil.Trim(matches[2]),
			Reasons:    []string{"bullet-list"},
			Confidence: 0.90,
		}
	}

	return nil
}

func (d *ListItemDetector) Order() int {
	return 90
}
