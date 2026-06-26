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
	Name() string
	Order() int
	Detect(detCtx *DetectorContext, text string) *vo.DocumentStructureSignal
}

type DetectorsManager interface {
	Register(detector Detector)
	Detect(text string, ctx *DetectorContext) *vo.DocumentStructureSignal
	GetDetectors() []Detector
}

func NewDetectorContext(documentTitle string, lineFrequency map[string]int) *DetectorContext {
	return &DetectorContext{
		DocumentTitle: documentTitle,
		LineFrequency: lineFrequency,
	}
}
