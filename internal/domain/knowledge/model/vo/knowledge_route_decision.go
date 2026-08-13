package vo

import "fmt"

// KnowledgeRouteDecision 知识路由决策
type KnowledgeRouteDecision struct {
	RouteStatus     string
	Confidence      float64
	Scopes          []*ScopeRouteCandidate
	Topics          []*TopicRouteCandidate
	Documents       []*DocumentRouteCandidate
	Source          string
	Reason          string
	IsDegraded      bool
	DegradedReasons []string
}

// ScopeRouteCandidate 知识范围（scope）路由候选
type ScopeRouteCandidate struct {
	ScopeId   int64              `json:"scopeId"`   // 知识范围ID
	ScopeName string             `json:"scopeName"` // 知识范围名称
	Score     float64            `json:"score"`     // 分数
	Reason    string             `json:"reason"`    // 原因
	Source    string             `json:"source"`    // 源
	Features  map[string]float64 `json:"features"`  // 特征
}

// TopicRouteCandidate 主题（topic）路由候选
type TopicRouteCandidate struct {
	TopicId   string             `json:"topicId"`   // 主题ID
	TopicName string             `json:"topicName"` // 主题名称
	ScopeId   int64              `json:"scopeId"`   // 知识范围ID
	Reason    string             `json:"reason"`    // 原因
	Source    string             `json:"source"`    // 源
	Features  map[string]float64 `json:"features"`  // 特征
}

// DocumentRouteCandidate 文档路由候选
type DocumentRouteCandidate struct {
	DocumentId      int64              `json:"documentId"`      // 文档ID
	DocumentName    string             `json:"documentName"`    // 文档名称
	LastIndexTaskId int64              `json:"lastIndexTaskId"` // 最后索引任务ID
	Score           float64            `json:"score"`           // 分数
	Reason          string             `json:"reason"`          // 原因
	Source          string             `json:"source"`          // 源
	Features        map[string]float64 `json:"features"`        // 特征
}

// ResolveHitSelectedDocument 当 selectedDocumentId 有效时，判断其是否在候选前三
func (k *KnowledgeRouteDecision) ResolveHitSelectedDocument(selectedDocumentId int64) int {
	if selectedDocumentId == 0 || len(k.Documents) == 0 {
		return 0
	}
	for idx := 0; idx < 3; idx++ {
		if k.Documents[idx].DocumentId == selectedDocumentId {
			return 1
		}
	}
	return 0
}

// ResolveConfidence 计算整体置信度：以 top1 分数/(top1+top2+5) 归一化
func (k *KnowledgeRouteDecision) ResolveConfidence() float64 {
	if k == nil || len(k.Documents) == 0 {
		return 0
	}
	top1 := k.Documents[0].Score
	top2 := 0.0
	if len(k.Documents) > 1 {
		top2 = k.Documents[1].Score
	}
	return top1 / max(10, top1+top2+5)
}

// ResolveDecisionReason 根据候选与置信度生成决策原因
func (k *KnowledgeRouteDecision) ResolveDecisionReason(confidence float64) string {
	if len(k.Documents) == 0 {
		return "没有找到可用候选文档"
	}
	top := k.Documents[0]
	if confidence >= 0.80 {
		return fmt.Sprintf("高置信度路由到《%s》，置信度 %.2f", top.DocumentName, confidence)
	} else if confidence >= 0.55 {
		return fmt.Sprintf("中等置信度路由到《%s》，置信度 %.2f", top.DocumentName, confidence)
	}
	return fmt.Sprintf("低置信度，前 %d 个候选得分接近，建议澄清", min(3, len(k.Documents)))
}
