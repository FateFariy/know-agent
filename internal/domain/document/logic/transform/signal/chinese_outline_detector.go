package signal

import (
	"regexp"
	"strings"

	"github.com/duke-git/lancet/v2/strutil"

	"github.com/swiftbit/know-agent/internal/domain/document/model/vo"
)

var chineseOutlinePattern = regexp.MustCompile(`^([一二三四五六七八九十百]+)[、.]\s*(.+)$`)

type ChineseOutlineDetector struct {
	BaseDetector
}

func (d *ChineseOutlineDetector) Name() string {
	return "chinese-outline"
}

func (d *ChineseOutlineDetector) Order() int {
	return 110
}

func (d *ChineseOutlineDetector) Detect(text string, ctx *DetectorContext) *vo.DocumentStructureSignal {
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
