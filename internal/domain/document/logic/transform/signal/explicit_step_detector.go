package signal

import (
	"regexp"
	"strings"

	"github.com/duke-git/lancet/v2/convertor"
	"github.com/duke-git/lancet/v2/strutil"

	"github.com/swiftbit/know-agent/internal/domain/document/model/vo"
)

var explicitStepPattern = regexp.MustCompile(`^(?:第\s*([0-9一二三四五六七八九十百]+)\s*步|步骤\s*([0-9一二三四五六七八九十百]+))\s*[:：、.]?\s*(.+)$`)

type ExplicitStepDetector struct{}

func (d *ExplicitStepDetector) Name() string {
	return "explicit-step"
}

func (d *ExplicitStepDetector) Order() int {
	return 30
}

func (d *ExplicitStepDetector) Detect(detCtx *DetectorContext, text string) *vo.DocumentStructureSignal {
	if text == "" {
		return nil
	}

	matches := explicitStepPattern.FindStringSubmatch(text)
	if len(matches) != 4 {
		return nil
	}

	group1 := strutil.Trim(matches[1])
	group2 := strutil.Trim(matches[2])
	title := strutil.Trim(matches[3])

	itemIndex := d.parseLooseNumber(group1)
	if itemIndex == 0 && group2 != "" {
		itemIndex = d.parseLooseNumber(group2)
	}

	return &vo.DocumentStructureSignal{
		Kind:       vo.SignalKindStepItem,
		Title:      title,
		ItemIndex:  itemIndex,
		Reasons:    []string{"explicit-step"},
		Confidence: 0.96,
	}
}

func (d *ExplicitStepDetector) parseLooseNumber(text string) int {
	normalized := strutil.Trim(text)
	if normalized == "" {
		return 0
	}

	if toInt, err := convertor.ToInt(normalized); err == nil {
		return int(toInt)
	}

	digitMap := map[rune]int{
		'一': 1, '二': 2, '三': 3, '四': 4, '五': 5,
		'六': 6, '七': 7, '八': 8, '九': 9, '十': 10,
	}

	if val, ok := digitMap[rune(normalized[0])]; ok {
		return val
	}
	if len(normalized) == 2 && strings.HasPrefix(normalized, "十") {
		return 10 + digitMap[rune(normalized[1])]
	}
	if len(normalized) == 2 && strings.HasSuffix(normalized, "十") {
		return digitMap[rune(normalized[0])] * 10
	}
	if len(normalized) == 3 && strings.Contains(normalized, "十") {
		return digitMap[rune(normalized[0])]*10 + digitMap[rune(normalized[2])]
	}

	return 0
}
