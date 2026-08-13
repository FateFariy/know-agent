package route

import (
	"context"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
	"github.com/swiftbit/know-agent/internal/domain/knowledge/adapter"
)

// RouteContext 路由上下文
type RouteContext struct {
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

func NewQueryContext(question, rewriteQuestion string, selectedKnowledgeBaseIds []int64, allowedDocumentIds []int64) *RouteContext {
	r := &RouteContext{
		Question:        utils.Trim(question),
		RewriteQuestion: utils.Trim(rewriteQuestion),
		Diagnostics:     make(map[string]struct{}),
	}
	r.RoutingText = r.buildRoutingText()
	keyOf := func(id int64) (int64, bool) {
		return id, id > 0
	}
	r.SelectedKnowledgeBaseIds = utils.FilterUniqueLimit(selectedKnowledgeBaseIds, -1, keyOf)
	r.AllowedDocumentIds = utils.FilterUniqueLimit(allowedDocumentIds, -1, keyOf)
	return r
}

func (r *RouteContext) Embedding(ctx context.Context, embedder adapter.Embedder) error {
	if embedder != nil && r.RoutingText != "" {
		vectors, err := embedder.EmbedStrings(ctx, r.RoutingText)
		if err != nil || len(vectors) == 0 {
			return err
		}
		r.QueryEmbedding = vectors[0]
	}
	return nil
}

// buildRoutingText 将原始问题与改写文本拼接；两文本相同则返回其一
func (r *RouteContext) buildRoutingText() string {
	if r.Question == "" {
		return r.RewriteQuestion
	}
	if r.RewriteQuestion == "" || r.Question == r.RewriteQuestion {
		return r.Question
	}
	return r.Question + " " + r.RewriteQuestion
}
