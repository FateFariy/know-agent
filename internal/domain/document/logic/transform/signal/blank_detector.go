package signal

import (
	"github.com/swiftbit/know-agent/internal/domain/document/model/vo"
)

type BlankDetector struct{}

func (d *BlankDetector) Name() string {
	return "blank"
}

func (d *BlankDetector) Order() int {
	return 0
}

func (d *BlankDetector) Detect(detCtx *DetectorContext, text string) *vo.DocumentStructureSignal {
	if text == "" {
		return &vo.DocumentStructureSignal{
			Kind:       vo.SignalKindBlank,
			Confidence: 1.0,
		}
	}
	return nil
}
