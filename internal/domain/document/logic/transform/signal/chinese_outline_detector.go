package signal

import (
	"regexp"
	"strings"

	"github.com/duke-git/lancet/v2/strutil"

	"github.com/swiftbit/know-agent/internal/domain/document/model/vo"
)

var chineseOutlinePattern = regexp.MustCompile(`^([一二三四五六七八九十百]+)[、.]\s*(.+)$`)

type ChineseOutlineDetector struct{}

func (d *ChineseOutlineDetector) Name() string {
	return "chinese-outline"
}

func (d *ChineseOutlineDetector) Order() int {
	return 110
}

func (d *ChineseOutlineDetector) Detect(detCtx *DetectorContext, text string) *vo.DocumentStructureSignal {
	if text == "" {
		return nil
	}

	matches := chineseOutlinePattern.FindStringSubmatch(text)
	if len(matches) != 3 {
		return nil
	}

	title := strutil.Trim(matches[2])
	itemIndex := d.parseLooseNumber(matches[1])

	return &vo.DocumentStructureSignal{
		Kind:       vo.SignalKindListItem,
		NodeCode:   strings.TrimSpace(matches[1]),
		Title:      title,
		ItemIndex:  itemIndex,
		Reasons:    []string{"chinese-outline-list"},
		Confidence: 0.86,
	}
}

func (d *ChineseOutlineDetector) parseLooseNumber(text string) int {
	normalized := strutil.Trim(text)
	if normalized == "" {
		return 0
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
