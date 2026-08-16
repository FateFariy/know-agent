package vo

import (
	"github.com/swiftbit/know-agent/internal/domain/knowledge/model/enum"
)

// KnowledgeRouteDecision 知识路由决策
type KnowledgeRouteDecision struct {
	Scopes          []*ScopeRouteCandidate    // 候选Scope列表
	Topics          []*TopicRouteCandidate    // 候选Topic列表
	Documents       []*DocumentRouteCandidate // 候选文档列表
	Confidence      float64                   // 路由决策置信度
	RouteStatus     enum.RouteStatus          // 路由状态（如SUCCESS/FAILED）
	Reason          string                    // 决策原因
	Source          string                    // 决策来源
	Degraded        bool                      // 是否降级
	DegradedReasons []string                  // 降级原因列表
}

func NewUnavailableRouteDecision(degradedReasons ...string) *KnowledgeRouteDecision {
	return &KnowledgeRouteDecision{
		RouteStatus:     enum.RouteStatusFailed,
		Reason:          "Route advice is unavailable; ordinary retrieval keeps the explicit knowledge scope",
		Source:          enum.KbSelectionModeNone,
		Degraded:        true,
		DegradedReasons: degradedReasons,
	}
}

// SelectRecommendedCandidate 从候选文档中选出推荐主文档
func (d *KnowledgeRouteDecision) SelectRecommendedCandidate(candidates []*DocumentRouteCandidate, threshold float64) *DocumentRouteCandidate {
	if d == nil || d.Confidence <= 0 || d.RouteStatus != enum.RouteStatusSuccess || len(candidates) == 0 {
		return nil
	}

	if d.Confidence < threshold || d.Confidence > 1.0 {
		return nil
	}

	top := candidates[0]
	if top == nil || !top.IsValidScore() || !d.IsOriginalTop(top) {
		return nil
	}
	return top
}

// IsOriginalTop 判断候选是否与原始路由决策的 top 候选一致
func (d *KnowledgeRouteDecision) IsOriginalTop(candidate *DocumentRouteCandidate) bool {
	if d == nil || len(d.Documents) == 0 || candidate == nil {
		return false
	}

	originalTop := d.Documents[0]
	return originalTop != nil && originalTop.SameDocumentTask(candidate)
}

// ScopeRouteCandidate 知识范围（scope）路由候选
type ScopeRouteCandidate struct {
	ScopeCode string  `json:"scopeCode"` // 知识范围代码
	ScopeName string  `json:"scopeName"` // 知识范围名称
	Score     float64 `json:"score"`     // 分数
	Reason    string  `json:"reason"`    // 原因
}

// TopicRouteCandidate 主题（topic）路由候选
type TopicRouteCandidate struct {
	TopicCode string  `json:"topicCode"` // 主题代码
	TopicName string  `json:"topicName"` // 主题名称
	ScopeCode string  `json:"scopeCode"` // 知识范围代码
	Score     float64 `json:"score"`     // 分数
	Reason    string  `json:"reason"`    // 原因
}

// DocumentRouteCandidate 文档路由候选
type DocumentRouteCandidate struct {
	DocumentId         int64   `json:"documentId"`      // 文档ID
	DocumentName       string  `json:"documentName"`    // 文档名称
	LastIndexTaskId    int64   `json:"lastIndexTaskId"` // 最后索引任务ID
	KnowledgeScopeCode string  `json:"-"`               // 知识范围代码
	KnowledgeScopeName string  `json:"-"`               // 知识范围名称
	BusinessCategory   string  `json:"-"`               // 业务类别
	DocumentTags       string  `json:"-"`               // 文档标签
	Score              float64 `json:"score"`           // 分数
	Reason             string  `json:"reason"`          // 原因
}

func (d *DocumentRouteCandidate) IsValidScore() bool {
	return d != nil && d.Score > 0
}

// SameDocumentTask 判断两个文档候选是否指向同一文档与索引任务
func (d *DocumentRouteCandidate) SameDocumentTask(other *DocumentRouteCandidate) bool {
	if d == nil || other == nil {
		return false
	}
	return d.DocumentId == other.DocumentId && d.LastIndexTaskId == other.LastIndexTaskId
}
