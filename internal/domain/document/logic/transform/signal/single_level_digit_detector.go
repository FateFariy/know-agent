package signal

import (
	"regexp"

	"github.com/duke-git/lancet/v2/strutil"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/document/model/vo"
)

var singleLevelDigitPattern = regexp.MustCompile(`^(\d+)\s*[、.]\s*(.+)$`)

type SingleLevelDigitDetector struct {
	BaseDetector
	maxPlainHeadingChars int
}

func NewSingleLevelDigitDetector(maxPlainHeadingChars int) *SingleLevelDigitDetector {
	if maxPlainHeadingChars <= 0 {
		maxPlainHeadingChars = 80
	}
	return &SingleLevelDigitDetector{
		maxPlainHeadingChars: maxPlainHeadingChars,
	}
}

func (d *SingleLevelDigitDetector) Detect(detCtx *DetectorContext, text string) *vo.DocumentStructureSignal {
	if text == "" {
		return nil
	}

	matches := singleLevelDigitPattern.FindStringSubmatch(text)
	if len(matches) != 3 {
		return nil
	}
	title := strutil.Trim(matches[2])
	itemIndex := d.parseLooseNumber(matches[1])
	sequential := d.isNeighborSequence(itemIndex, arabicSingle, detCtx)
	introducedByLeadIn := d.previousIntroducesList(detCtx.previousNonBlank)
	headingLike := !sequential && !introducedByLeadIn && d.looksLikePlainHeading(title, detCtx)

	var kind vo.DocumentStructureSignalKind
	var reasons []string
	var confidence float64

	if headingLike {
		kind = vo.SignalKindHeadingCandidate
		reasons = []string{"single-digit-ambiguous-heading"}
		confidence = 0.62
	} else if sequential {
		kind = vo.SignalKindListItem
		reasons = []string{"single-digit-sequence-list"}
		confidence = 0.93
	} else {
		kind = vo.SignalKindListItem
		reasons = []string{"single-digit-list"}
		confidence = 0.88
	}

	return &vo.DocumentStructureSignal{
		Kind:        kind,
		NodeCode:    strutil.Trim(matches[1]),
		Title:       title,
		ItemIndex:   itemIndex,
		NumericPath: utils.Ternary(headingLike && itemIndex > 0, []int{itemIndex}, nil),
		Reasons:     reasons,
		Confidence:  confidence,
	}
}

func (d *SingleLevelDigitDetector) Order() int {
	return 100
}
