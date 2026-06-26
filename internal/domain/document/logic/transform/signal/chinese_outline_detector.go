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
	sequential := d.isNeighborSequence(itemIndex, chineseOutline, context)
	introducedByLeadIn := d.previousIntroducesList(context.previousNonBlank)
	headingLike := !sequential && !introducedByLeadIn && d.looksLikePlainHeading(title, context)

	var kind vo.DocumentStructureSignalKind
	var reasons []string
	var confidence float64

	if headingLike {
		kind = vo.SignalKindHeadingCandidate
		reasons = []string{"chinese-outline-ambiguous-heading"}
		confidence = 0.60
	} else if sequential {
		kind = vo.SignalKindListItem
		reasons = []string{"chinese-outline-sequence-list"}
		confidence = 0.92
	} else {
		kind = vo.SignalKindListItem
		reasons = []string{"chinese-outline-list"}
		confidence = 0.86
	}

	baseSignal.Kind = kind
	baseSignal.Reasons = reasons
	baseSignal.Confidence = confidence

	if headingLike && itemIndex != nil && *itemIndex > 0 {
		baseSignal.NumericPath = []int{*itemIndex}
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
