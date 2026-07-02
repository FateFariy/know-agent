package vo

// KnowledgeRouteDecision 知识路由决策
type KnowledgeRouteDecision struct {
	RouteStatus           string
	Confidence            float64
	Scopes                []*ScopeRouteCandidate
	Topics                []*TopicRouteCandidate
	Documents             []*DocumentRouteCandidate
	Reason                string
	RequiresClarification bool
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
