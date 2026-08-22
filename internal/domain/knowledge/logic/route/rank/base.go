package rank

import (
	"context"
	"math"

	"github.com/swiftbit/know-agent/common"
	"github.com/swiftbit/know-agent/common/logx"
	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/knowledge/adapter"
	"github.com/swiftbit/know-agent/internal/domain/knowledge/logic/route/score"
	"github.com/swiftbit/know-agent/internal/domain/knowledge/model/vo"
)

const (
	routeEmbeddingBatchSize = 10
)

type base struct {
	docGateway adapter.DocumentGateway
	*options
}

func newBaseRanker(docGateway adapter.DocumentGateway, opts ...Option) *base {
	b := &base{
		docGateway: docGateway,
		options: &options{
			scorer: score.NewDefaultScorer(),
		},
	}
	b.options = common.GetImplSpecificOptions(b.options, opts...)
	return b
}

// computeSemanticScores 批量计算 routingText 与每个候选文本的余弦相似度；embedder 未配置时返回全 0 长度相同
func (b *base) computeSemanticScores(ctx context.Context, rankCtx *Context, routeTexts []string) []float64 {
	scores := make([]float64, len(routeTexts))
	if len(rankCtx.QueryEmbedding) == 0 || len(routeTexts) == 0 {
		return scores
	}
	if b.embedder == nil {
		rankCtx.Diagnostics["SEMANTIC_ROUTE_NOT_CONFIGURED"] = struct{}{}
		return scores
	}

	for start := 0; start < len(routeTexts); start += routeEmbeddingBatchSize {
		end := min(start+routeEmbeddingBatchSize, len(routeTexts))
		batch := routeTexts[start:end]
		embeddings, err := b.embedder.EmbedStrings(ctx, batch...)
		if err != nil {
			logx.Warnf("知识路由批量向量计算失败: batchStart=%d, size=%d, err=%v", start, len(batch), err)
			rankCtx.Diagnostics["SEMANTIC_CANDIDATE_EMBEDDING_UNAVAILABLE"] = struct{}{}
			return make([]float64, len(routeTexts))
		}
		if len(embeddings) != len(batch) {
			rankCtx.Diagnostics["SEMANTIC_CANDIDATE_EMBEDDING_INVALID"] = struct{}{}
			return make([]float64, len(routeTexts))
		}
		for idx, emb := range embeddings {
			scores[start+idx] = cosineSimilarity(rankCtx.QueryEmbedding, emb)
		}
	}
	return scores
}

// searchLexicalScores 调用外部词面索引；未配置或失败时回退到本地计算
func (b *base) searchLexicalScores(ctx context.Context, rankCtx *Context, entityType string, size int) map[int64]float64 {
	if b.lexicalIndex == nil {
		rankCtx.Diagnostics["ROUTE_INDEX_NOT_CONFIGURED"] = struct{}{}
		return nil
	}
	input := &EsSearchInput{
		RoutingText: rankCtx.RoutingText,
		EntityType:  entityType,
		Size:        size,
		KbIds:       rankCtx.SelectedKnowledgeBaseIds,
	}
	hits, err := b.lexicalIndex.Search(ctx, input)

	if err != nil {
		rankCtx.Diagnostics["ROUTE_INDEX_UNAVAILABLE"] = struct{}{}
		return nil
	}
	if len(hits) == 0 {
		return nil
	}
	if len(rankCtx.SelectedKnowledgeBaseIds) != 0 {
		hits = utils.Filter(hits, func(hit *vo.RouteLexicalHit) bool {
			return utils.ContainsAny(rankCtx.SelectedKnowledgeBaseIds, hit.KnowledgeBaseId)
		})
	}
	if len(rankCtx.AllowedDocumentIds) != 0 {
		hits = utils.Filter(hits, func(hit *vo.RouteLexicalHit) bool {
			return utils.ContainsAny(rankCtx.AllowedDocumentIds, hit.DocumentId)
		})
	}
	keyFunc := func(hit *vo.RouteLexicalHit) (int64, float64) {
		return hit.EntityId, hit.Score
	}
	return utils.MapBy(hits, keyFunc)
}

// listRetrievableDocuments 查询可检索的文档
func (b *base) listRetrievableDocuments(ctx context.Context, rankCtx *Context) ([]*vo.DocumentMetadata, error) {
	var docs []*vo.DocumentMetadata
	var err error
	if len(rankCtx.AllowedDocumentIds) != 0 {
		docs, err = b.docGateway.FindRetrieveDocumentByIds(ctx, rankCtx.AllowedDocumentIds...)
		if err != nil {
			logx.Warnf("查询可检索文档失败: %v", err)
		}
	}
	if len(docs) == 0 {
		docs, err = b.docGateway.FindRetrievableByKbIds(ctx, rankCtx.SelectedKnowledgeBaseIds)
		if err != nil {
			logx.Warnf("查询可检索文档失败: %v", err)
		}
	}
	return docs, err
}

// cosineSimilarity 计算两个等长向量的余弦相似度
func cosineSimilarity(left, right []float64) float64 {
	if len(left) == 0 || len(right) == 0 || len(left) != len(right) {
		return 0
	}
	var dot, lNorm, rNorm float64
	for i := 0; i < len(left); i++ {
		dot += left[i] * right[i]
		lNorm += left[i] * left[i]
		rNorm += right[i] * right[i]
	}
	if lNorm <= 0 || rNorm <= 0 {
		return 0
	}
	return dot / (math.Sqrt(lNorm) * math.Sqrt(rNorm))
}
