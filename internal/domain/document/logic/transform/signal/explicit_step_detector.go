package signal

import (
	"regexp"

	"github.com/duke-git/lancet/v2/strutil"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/document/model/vo"
)

var explicitStepPattern = regexp.MustCompile(`^(?:第\s*([0-9一二三四五六七八九十百]+)\s*步|步骤\s*([0-9一二三四五六七八九十百]+))\s*[:：、.]?\s*(.+)$`)

type ExplicitStepDetector struct {
	BaseDetector
}

func (d *ExplicitStepDetector) Detect(detCtx *DetectorContext, text string) *vo.DocumentStructureSignal {
	if text == "" {
		return nil
	}

	matches := explicitStepPattern.FindStringSubmatch(text)
	if len(matches) != 4 {
		return nil
	}
	itemIndex := d.parseLooseNumber(utils.BlankToDefault(strutil.Trim(matches[1]), strutil.Trim(matches[2])))
	return &vo.DocumentStructureSignal{
		Kind:       vo.SignalKindStepItem,
		Title:      strutil.Trim(matches[3]),
		ItemIndex:  itemIndex,
		Reasons:    []string{"explicit-step"},
		Confidence: 0.96,
	}
}

func (d *ExplicitStepDetector) Order() int {
	return 30
}
