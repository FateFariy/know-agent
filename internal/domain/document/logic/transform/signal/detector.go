package signal

import (
	"github.com/swiftbit/know-agent/internal/domain/document/model/vo"
)

type DetectorContext struct {
	DocumentTitle string
	LineFrequency map[string]int
}

type LineClassifier interface {
	Classify(text string) LineClassification
}

type LineClassification struct {
	IsHeading bool
}

type Detector interface {
	Detect(detCtx *DetectorContext, text string) *vo.DocumentStructureSignal
	Order() int
}

type DetectorsManager interface {
	Register(detector Detector)
	Detect(detCtx *DetectorContext, text string) *vo.DocumentStructureSignal
}

func NewDetectorContext(documentTitle string, lineFrequency map[string]int) *DetectorContext {
	return &DetectorContext{
		DocumentTitle: documentTitle,
		LineFrequency: lineFrequency,
	}
}
