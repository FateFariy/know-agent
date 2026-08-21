package emb

import (
	"context"

	"github.com/cloudwego/eino/components/embedding"

	"github.com/swiftbit/know-agent/internal/svc"
)

type Embedder struct {
	emb embedding.Embedder
}

func NewEmbedder(svcCtx *svc.ServiceContext) *Embedder {
	return &Embedder{emb: svcCtx.Emb}
}

func (e *Embedder) EmbedStrings(ctx context.Context, texts ...string) ([][]float64, error) {
	return e.emb.EmbedStrings(ctx, texts)
}
