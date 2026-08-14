package vo

import (
	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/enum"
)

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
func (i *StructureNavigationIntent) IsConfident(confidenceThreshold float64) bool {
	if i == nil || len(i.Operations) == 0 {
		return false
	}
	return i.NormalizeConfidence() >= confidenceThreshold
}

// ResolveAction 解析结构导航动作
func (i *StructureNavigationIntent) ResolveAction(threshold float64) string {
	if i == nil || !i.IsConfident(threshold) {
		return ""
	}

	contains := func(ops ...enum.StructureNavigationOperation) bool {
		return utils.ContainsAny(i.Operations, ops...)
	}
	// 目录展开
	if contains(enum.SectionWithChildren, enum.DirectChildren) {
		return enum.DocumentNavigationActionChildSectionDescend
	}
	// 相邻章节
	if contains(enum.SectionWithSiblings, enum.PreviousSibling, enum.NextSibling) {
		return enum.DocumentNavigationActionSectionAdjacencyLookup
	}
	// 父章节
	if contains(enum.ParentSection) {
		return enum.DocumentNavigationActionAncestorSectionReturn
	}
	// 当前章节
	if contains(enum.CurrentSection) {
		return enum.DocumentNavigationActionFreshTopic
	}
	return ""
}

// NormalizeConfidence 将模型返回的可能超出 0-1 的置信度规范化到 [0, 1)，再由调用方与阈值比较
func (i *StructureNavigationIntent) NormalizeConfidence() float64 {
	if i.Confidence > 1 {
		return i.Confidence / 100.0
	}
	return max(0, i.Confidence)
}
