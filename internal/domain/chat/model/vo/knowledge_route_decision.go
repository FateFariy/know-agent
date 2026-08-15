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

func NewUnavailableRouteDecision(degradedReasons ...string) *KnowledgeRouteDecision {
	return &KnowledgeRouteDecision{
		RouteStatus:     enum.RouteStatusFailed,
		Reason:          "Route advice is unavailable; ordinary retrieval keeps the explicit knowledge scope",
		Source:          enum.KbSelectionModeNone,
		Degraded:        true,
		DegradedReasons: degradedReasons,
	}
}

// ResolveHitSelectedDocument 当 selectedDocumentId 有效时，判断其是否候选前三
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
