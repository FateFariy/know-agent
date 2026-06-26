package signal

import (
	"github.com/swiftbit/know-agent/internal/domain/document/model/vo"
)

type QuoteDetector struct{}

func (d *QuoteDetector) Name() string {
	return "quote"
}

func (d *QuoteDetector) Order() int {
	return 80
}

func (d *QuoteDetector) Detect(detCtx *DetectorContext, text string) *vo.DocumentStructureSignal {
	if text == "" {
		return nil
	}

	if len(text) > 0 && text[0] == '>' {
		return &vo.DocumentStructureSignal{
			Kind:       vo.SignalKindQuote,
			Reasons:    []string{"quote"},
			Confidence: 0.88,
		}
	}

	return nil
}
