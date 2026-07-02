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
