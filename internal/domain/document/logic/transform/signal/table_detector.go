package signal

import (
	"regexp"
	"strings"

	"github.com/swiftbit/know-agent/internal/domain/document/model/vo"
)

var (
	tableSplitPattern  = regexp.MustCompile(`\|`)
	specialLinePattern = regexp.MustCompile(`^[:\-\\s|]+$`)
)

type TableRowDetector struct{}

func (d *TableRowDetector) Name() string {
	return "table-row"
}

func (d *TableRowDetector) Order() int {
	return 70
}

func (d *TableRowDetector) Detect(detCtx *DetectorContext, text string) *vo.DocumentStructureSignal {
	if text == "" {
		return nil
	}

	if d.isTableRow(text) {
		return &vo.DocumentStructureSignal{
			Kind:       vo.SignalKindTableRow,
			Reasons:    []string{"table-row"},
			Confidence: 0.90,
		}
	}

	return nil
}

func (d *TableRowDetector) isTableRow(text string) bool {
	if strings.HasPrefix(text, "|") && strings.HasSuffix(text, "|") {
		return true
	}
	if strings.Contains(text, "\t") {
		return true
	}
	if len(tableSplitPattern.Split(text, -1)) >= 3 && strings.Contains(text, "|") {
		return true
	}
	return specialLinePattern.MatchString(text)
}
