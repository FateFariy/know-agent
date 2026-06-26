package signal

import (
	"sort"

	"github.com/swiftbit/know-agent/internal/domain/document/model/vo"
)

type DefaultDetectorsManager struct {
	detectors []Detector
}

var _ DetectorsManager = (*DefaultDetectorsManager)(nil)

func NewDefaultDetectorsManager() *DefaultDetectorsManager {
	mgr := &DefaultDetectorsManager{
		detectors: make([]Detector, 0),
	}
	mgr.registerDefaultDetectors()
	return mgr
}

func (m *DefaultDetectorsManager) registerDefaultDetectors() {
	m.detectors = append(m.detectors, &BlankDetector{})
	m.detectors = append(m.detectors, &NoiseDetector{})
	m.detectors = append(m.detectors, &MarkdownHeadingDetector{})
	m.detectors = append(m.detectors, &ExplicitStepDetector{})
	m.detectors = append(m.detectors, &ChapterHeadingDetector{})
	m.detectors = append(m.detectors, &AppendixHeadingDetector{})
	m.detectors = append(m.detectors, &DecimalHeadingDetector{})
	m.detectors = append(m.detectors, &TableRowDetector{})
	m.detectors = append(m.detectors, &QuoteDetector{})
	m.detectors = append(m.detectors, &ListItemDetector{})
	m.detectors = append(m.detectors, NewSingleLevelDigitDetector(80))
	m.detectors = append(m.detectors, &ChineseOutlineDetector{})
	m.detectors = append(m.detectors, &BodyDetector{})

	sort.Slice(m.detectors, func(i, j int) bool {
		return m.detectors[i].Order() < m.detectors[j].Order()
	})
}

func (m *DefaultDetectorsManager) Register(detector Detector) {
	if detector == nil {
		return
	}
	m.detectors = append(m.detectors, detector)
	sort.Slice(m.detectors, func(i, j int) bool {
		return m.detectors[i].Order() < m.detectors[j].Order()
	})
}

func (m *DefaultDetectorsManager) Detect(detCtx *DetectorContext, text string) *vo.DocumentStructureSignal {
	for _, detector := range m.detectors {
		result := detector.Detect(detCtx, text)
		if result != nil {
			return result
		}
	}

	return &vo.DocumentStructureSignal{
		Kind:       vo.SignalKindBody,
		Reasons:    []string{"body"},
		Confidence: 1.0,
	}
}
