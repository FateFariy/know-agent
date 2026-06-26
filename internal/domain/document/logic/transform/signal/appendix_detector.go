package signal

import (
	"regexp"

	"github.com/duke-git/lancet/v2/strutil"

	"github.com/swiftbit/know-agent/internal/domain/document/model/vo"
)

var appendixPattern = regexp.MustCompile(`^(附录\s*([A-Za-z一二三四五六七八九十百\d]+))(?:\s+(.+))?$`)

type AppendixHeadingDetector struct{}

func (d *AppendixHeadingDetector) Name() string {
	return "appendix-heading"
}

func (d *AppendixHeadingDetector) Order() int {
	return 50
}

func (d *AppendixHeadingDetector) Detect(detCtx *DetectorContext, text string) *vo.DocumentStructureSignal {
	if text == "" {
		return nil
	}

	matches := appendixPattern.FindStringSubmatch(text)
	if len(matches) < 3 {
		return nil
	}

	nodeCode := strutil.Trim(matches[1])
	title := strutil.Trim(matches[3])
	if title == "" {
		title = nodeCode
	}

	return &vo.DocumentStructureSignal{
		Kind:       vo.SignalKindHeading,
		NodeCode:   nodeCode,
		Title:      title,
		LevelHint:  1,
		Reasons:    []string{"appendix-heading"},
		Confidence: 0.92,
	}
}
