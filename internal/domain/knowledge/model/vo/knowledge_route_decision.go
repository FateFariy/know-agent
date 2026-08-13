package vo

import (
	"github.com/swiftbit/know-agent/common/utils"
)

// KnowledgeRouteDecision 知识路由决策
type KnowledgeRouteDecision struct {
	RouteStatus     string                  // 路由状态（SUCCESS/FAILED）
	Confidence      float64                 // 置信度（0-1）
	Scopes          []*ScopeRouteCandidate  // 知识范围（scope）路由候选
	Topics          []*TopicRouteCandidate  // 主题（topic）路由候选
	Documents       DocumentRouteCandidates // 文档路由候选
	Source          string                  // 来源（SEMANTIC/ROUTE_INDEX/PERSISTED_RELATION/COMPOSITE）
	Reason          string                  // 原因
	IsDegraded      bool                    // 是否降级
	DegradedReasons []string                // 降级原因
}

// Resolve 填充决策对象的置信度、来源、路由状态、原因
func (d *KnowledgeRouteDecision) Resolve(lowConfidenceThreshold float64) {
	if d == nil {
		return
	}
	// 计算置信度
	d.Confidence = d.ResolveConfidence()
	// 解析决策来源
	d.Source = d.ResolveSource()
	// 解析路由状态
	d.RouteStatus = d.ResolveRouteStatus(lowConfidenceThreshold)
	// 解析决策原因
	d.Reason = d.ResolveReason(lowConfidenceThreshold)
}

// ResolveSource 从文档候选中解析决策来源
func (d *KnowledgeRouteDecision) ResolveSource() string {
	if d == nil || len(d.Documents) == 0 {
		return "NONE"
	}
	sourceSet := utils.FilterMapUniqueLimit(d.Documents, -1, func(c *DocumentRouteCandidate) (string, string, bool) {
		if c != nil && utils.IsNotBlank(c.Source) {
			return c.Source, c.Source, true
		}
		return "", "", false
	})
	if len(sourceSet) == 1 {
		return sourceSet[0]
	}
	if len(sourceSet) > 1 {
		return "COMPOSITE"
	}
	return "NONE"
}

// ResolveHitSelectedDocument 当 selectedDocumentId 有效时，判断其是否在候选前三
func (d *KnowledgeRouteDecision) ResolveHitSelectedDocument(selectedDocumentId int64) int {
	if d == nil || selectedDocumentId == 0 || len(d.Documents) == 0 {
		return 0
	}
	for idx := 0; idx < 3; idx++ {
		if d.Documents[idx].DocumentId == selectedDocumentId {
			return 1
		}
	}
	return 0
}

// ResolveConfidence 计算整体置信度：以 top1 分数/(top1+top2+5) 归一化
func (d *KnowledgeRouteDecision) ResolveConfidence() float64 {
	if d == nil || len(d.Documents) == 0 {
		return 0
	}
	return d.Documents.ComputeConfidence()
}

// ResolveReason 根据候选与置信度生成决策原因
func (d *KnowledgeRouteDecision) ResolveReason(lowConfidenceThreshold float64) string {
	if d == nil || len(d.Documents) == 0 {
		return "没有找到可用候选文档"
	}
	top := d.Documents[0]
	if d.Confidence < lowConfidenceThreshold {
		return utils.BlankToDefault(top.Reason, "低置信度，已进入保守扩范围候选")
	}
	return top.Reason
}

func (d *KnowledgeRouteDecision) ResolveRouteStatus(lowConfidenceThreshold float64) string {
	if len(d.Documents) == 0 {
		return "FAILED"
	} else if d.Confidence < lowConfidenceThreshold {
		return "LOW_CONFIDENCE"
	} else {
		return "SUCCESS"
	}
}

// ScopeRouteCandidate 知识范围（scope）路由候选
type ScopeRouteCandidate struct {
	ScopeId   int64              `json:"scopeId"`   // 知识范围ID
	ScopeName string             `json:"scopeName"` // 知识范围名称
	Score     float64            `json:"score"`     // 分数
	Reason    string             `json:"reason"`    // 原因
	Source    string             `json:"source"`    // 来源（SEMANTIC/ROUTE_INDEX/PERSISTED_RELATION/COMPOSITE）
	Features  map[string]float64 `json:"features"`  // 特征分明细
}

// TopicRouteCandidate 主题（topic）路由候选
type TopicRouteCandidate struct {
	TopicId   int64              `json:"topicId"`   // 主题ID
	TopicName string             `json:"topicName"` // 主题名称
	ScopeId   int64              `json:"scopeId"`   // 知识范围ID
	Score     float64            `json:"score"`     // 分数
	Reason    string             `json:"reason"`    // 原因
	Source    string             `json:"source"`    // 来源
	Features  map[string]float64 `json:"features"`  // 特征分明细
}

// DocumentRouteCandidate 文档路由候选
type DocumentRouteCandidate struct {
	DocumentId      int64              `json:"documentId"`      // 文档ID
	DocumentName    string             `json:"documentName"`    // 文档名称
	LastIndexTaskId int64              `json:"lastIndexTaskId"` // 最后索引任务ID
	Score           float64            `json:"score"`           // 分数
	Reason          string             `json:"reason"`          // 原因
	Source          string             `json:"source"`          // 来源
	Features        map[string]float64 `json:"features"`        // 特征分明细
}

type DocumentRouteCandidates []*DocumentRouteCandidate

// ComputeConfidence 从候选列表计算置信度：top1/(top1+top2+5) 归一化
func (d DocumentRouteCandidates) ComputeConfidence() float64 {
	if len(d) == 0 {
		return 0
	}
	top := d[0].Score
	second := 0.0
	if len(d) > 1 {
		second = d[1].Score
	}
	return top / max(10, top+second+5)
}
