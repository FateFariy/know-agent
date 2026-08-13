package rank

import "github.com/swiftbit/know-agent/internal/domain/knowledge/model/vo"

type Context struct {
	Question                 string
	RewriteQuestion          string
	RoutingText              string
	QueryEmbedding           []float64
	SelectedKnowledgeBaseIds []int64
	AllowedDocumentIds       []int64
	Diagnostics              map[string]struct{}
	ScopeCandidates          []*vo.ScopeRouteCandidate
	TopicCandidates          []*vo.TopicRouteCandidate
	DocumentCandidates       []*vo.DocumentRouteCandidate
}
