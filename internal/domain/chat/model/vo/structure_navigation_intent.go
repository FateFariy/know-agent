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
