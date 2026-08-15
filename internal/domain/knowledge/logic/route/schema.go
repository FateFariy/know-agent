package route

import (
	"context"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/knowledge/adapter"
)

// Context 路由上下文
type Context struct {
	ExchangeId                 int64
	ConversationId             string
	Question                   string
	RewriteQuestion            string
	KnowledgeBaseSelectionMode string
	SelectedDocumentId         int64
	RoutingText                string
	QueryEmbedding             []float64
	SelectedKnowledgeBaseIds   []int64
	SelectedKnowledgeBaseNames []string
	AllowedDocumentIds         []int64
	Diagnostics                map[string]struct{}
}

func NewRouteContext(question, rewriteQuestion string, selectedKnowledgeBaseIds []int64, allowedDocumentIds []int64) *Context {
	r := &Context{
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

func (r *Context) Embedding(ctx context.Context, embedder adapter.Embedder) {
	if embedder == nil {
		r.Diagnostics["SEMANTIC_ROUTE_NOT_CONFIGURED"] = struct{}{}
	}
	if embedder != nil && r.RoutingText != "" {
		vectors, err := embedder.EmbedStrings(ctx, r.RoutingText)
		if err != nil || len(vectors) == 0 {
			r.Diagnostics["SEMANTIC_QUERY_EMBEDDING_UNAVAILABLE"] = struct{}{}
			return
		}
		r.QueryEmbedding = vectors[0]
	}
	return
}

// buildRoutingText 将原始问题与改写文本拼接；两文本相同则返回其一
func (r *Context) buildRoutingText() string {
	if r.Question == "" {
		return r.RewriteQuestion
	}
	if r.RewriteQuestion == "" || r.Question == r.RewriteQuestion {
		return r.Question
	}
	return r.Question + " " + r.RewriteQuestion
}
