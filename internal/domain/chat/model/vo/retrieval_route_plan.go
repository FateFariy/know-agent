package vo

import (
	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/enum"
)

// RetrievalRoutePlan 检索路由计划
type RetrievalRoutePlan struct {
	Source                   string                     `json:"source"`                   // 来源
	Status                   string                     `json:"status"`                   // 状态
	Confidence               float64                    `json:"confidence"`               // 置信度
	Degraded                 bool                       `json:"degraded"`                 // 是否降级
	DegradedReasons          []string                   `json:"degradedReasons"`          // 降级原因
	TopDocumentHintId        int64                      `json:"topDocumentHintId"`        // 推荐文档ID
	TopTaskHintId            int64                      `json:"topTaskHintId"`            // 推荐任务ID
	AuthorizationMode        string                     `json:"authorizationMode"`        // 授权模式
	ScopeAuthorizationReason string                     `json:"scopeAuthorizationReason"` // 范围授权原因
	AuthorizedDocumentIds    []int64                    `json:"authorizedDocumentIds"`    // 授权文档ID列表
	AuthorizedTaskIds        []int64                    `json:"authorizedTaskIds"`        // 授权任务ID列表
	Candidates               []*RetrievalRouteCandidate `json:"candidates"`               // 候选列表
}

// RetrievalRouteCandidate 检索路由候选
type RetrievalRouteCandidate struct {
	CandidateType string             `json:"candidateType"` // 候选类型
	CandidateId   string             `json:"candidateId"`   // 候选ID
	DocumentId    int64              `json:"documentId"`    // 文档ID
	TaskId        int64              `json:"taskId"`        // 任务ID
	DisplayName   string             `json:"displayName"`   // 显示名称
	Score         float64            `json:"score"`         // 分数
	Reason        string             `json:"reason"`        // 原因
	Source        string             `json:"source"`        // 来源
	Features      map[string]float64 `json:"features"`      // 特征
}

// RetrievalCandidate 检索候选（简化版）
type RetrievalCandidate struct {
	DocumentId  int64   `json:"documentId"`  // 文档ID
	TaskId      int64   `json:"taskId"`      // 任务ID
	DisplayName string  `json:"displayName"` // 显示名称
	Score       float64 `json:"score"`       // 分数
	Reason      string  `json:"reason"`      // 原因
	Source      string  `json:"source"`      // 来源
}

// BuildRetrievalRoutePlan 构建检索路由计划
func BuildRetrievalRoutePlan(input *AssemblyInput) *RetrievalRoutePlan {
	source := "ALLOWED_DOCUMENT_SCOPE"
	documentIds := utils.Copy(input.DocumentScope)
	taskIds := utils.Copy(input.TaskScope)

	candidates := make([]*RetrievalRouteCandidate, 0)
	pairCount := min(len(documentIds), len(taskIds))
	for i := 0; i < pairCount; i++ {
		reason := "Allowed scope expansion"
		if input.ChatMode == enum.ChatQueryModeDocument {
			source = "EXPLICIT_DOCUMENT_SCOPE"
			reason = "Explicit document scope"
		}
		candidates = append(candidates, &RetrievalRouteCandidate{
			CandidateType: "DOCUMENT",
			DocumentId:    documentIds[i],
			TaskId:        taskIds[i],
			DisplayName:   "",
			Score:         0,
			Reason:        reason,
			Source:        source,
			Features:      map[string]float64{},
		})
	}

	degraded := input.ChatMode == enum.ChatQueryModeAutoDocument
	status := "SUCCESS"
	if degraded {
		status = "LOW_CONFIDENCE"
	}
	degradedReasons := []string{}
	if degraded {
		degradedReasons = append(degradedReasons, "No scored route candidate; expanded inside allowed document scope")
	}
	confidence := 1.0
	if degraded {
		confidence = 0.0
	}

	authorizationMode := resolveAuthorizationMode(input)

	return &RetrievalRoutePlan{
		Source:                   source,
		Status:                   status,
		Confidence:               confidence,
		Degraded:                 degraded,
		DegradedReasons:          degradedReasons,
		AuthorizationMode:        authorizationMode,
		ScopeAuthorizationReason: utils.BlankToDefault("", ""),
		AuthorizedDocumentIds:    utils.Copy(input.AllowedDocumentIds),
		AuthorizedTaskIds:        utils.Copy(input.TaskScope),
		Candidates:               candidates,
	}
}

// resolveAuthorizationMode 解析授权模式
func resolveAuthorizationMode(input *AssemblyInput) string {
	if input == nil {
		return "KNOWLEDGE_BASE_ALLOWED_SCOPE"
	}
	switch input.ChatMode {
	case enum.ChatQueryModeDocument:
		return "EXPLICIT_DOCUMENT"
	case enum.ChatQueryModeAutoDocument:
		return "KNOWLEDGE_BASE_ALLOWED_SCOPE"
	default:
		return "KNOWLEDGE_BASE_ALLOWED_SCOPE"
	}
}
