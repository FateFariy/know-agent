package rank

import (
	"context"
	"math"

	"github.com/swiftbit/know-agent/common/logx"
	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/knowledge/adapter"
	"github.com/swiftbit/know-agent/internal/domain/knowledge/model/vo"
)

const (
	routeEmbeddingBatchSize = 10
)

type base struct {
	embedder     adapter.Embedder
	lexicalIndex adapter.RouteLexicalIndex
}

func newBaseRanker(embedder adapter.Embedder, lexicalIndex adapter.RouteLexicalIndex) base {
	return base{
		embedder:     embedder,
		lexicalIndex: lexicalIndex,
	}
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
func (b *base) searchLexicalScores(ctx context.Context, routingText, entityType string, size int) (map[int64]float64, error) {
	if b.lexicalIndex == nil {
		return nil, nil
	}
	hits, err := b.lexicalIndex.Search(ctx, routingText, entityType, size)
	if err != nil || len(hits) == 0 {
		return nil, err
	}
	keyFunc := func(hit *vo.RouteLexicalHit) (int64, float64) {
		return hit.EntityId, hit.Score
	}
	return utils.MapBy(hits, keyFunc), nil
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
