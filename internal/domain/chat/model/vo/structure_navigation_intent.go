package vo

import "github.com/swiftbit/know-agent/internal/domain/chat/model/enum"

// StructureNavigationIntent 结构导航意图
type StructureNavigationIntent struct {
	Operations            []enum.StructureNavigationOperation
	AnchorStructureNodeId int64
	AnchorSectionPath     string
	AnchorCanonicalPath   string
	SectionAnchors        []string
	Confidence            float64
	Source                string
}

// IsConfident 判断结构导航意图是否有效且置信度足够
func (intent *StructureNavigationIntent) IsConfident(confidenceThreshold float64) bool {
	if intent == nil || len(intent.Operations) == 0 {
		return false
	}
	return normalizeConfidence(intent.Confidence) >= confidenceThreshold
}

// normalizeConfidence 将模型返回的可能超出 0-1 的置信度规范化到 [0, 1)，再由调用方与阈值比较
func normalizeConfidence(confidence float64) float64 {
	if confidence > 1 {
		return confidence / 100.0
	}
	return max(0, confidence)
}
