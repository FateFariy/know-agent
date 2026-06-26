package signal

import (
	"github.com/swiftbit/know-agent/internal/domain/document/model/vo"
)

type BodyDetector struct{}

func (d *BodyDetector) Name() string {
	return "body"
}

func (d *BodyDetector) Order() int {
	return 1000
}

func (d *BodyDetector) Detect(detCtx *DetectorContext, text string) *vo.DocumentStructureSignal {
	return &vo.DocumentStructureSignal{
		Kind:       vo.SignalKindBody,
		Reasons:    []string{"body"},
		Confidence: 1.0,
	}
}
