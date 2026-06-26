package signal

import (
	"regexp"
	"strings"

	"github.com/duke-git/lancet/v2/strutil"

	"github.com/swiftbit/know-agent/internal/domain/document/model/vo"
)

var singleLevelDigitPattern = regexp.MustCompile(`^(\d+)\s*[、.]\s*(.+)$`)

type SingleLevelDigitDetector struct {
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

func (d *SingleLevelDigitDetector) Name() string {
	return "single-level-digit"
}

func (d *SingleLevelDigitDetector) Order() int {
	return 100
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

	return &vo.DocumentStructureSignal{
		Kind:       vo.SignalKindListItem,
		NodeCode:   strings.TrimSpace(matches[1]),
		Title:      title,
		ItemIndex:  itemIndex,
		Reasons:    []string{"single-digit-list"},
		Confidence: 0.88,
	}
}

func (d *SingleLevelDigitDetector) parseLooseNumber(text string) int {
	normalized := strutil.Trim(text)
	if normalized == "" {
		return 0
	}

	for _, c := range normalized {
		if c < '0' || c > '9' {
			return 0
		}
	}

	var num int
	for _, c := range normalized {
		num = num*10 + int(c-'0')
	}
	return num
}
