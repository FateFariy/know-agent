package ingestion

import (
	"context"

	"github.com/cloudwego/eino/components/document"
	"github.com/cloudwego/eino/schema"
)

type ChunkerNode struct {
}

var _ document.Transformer = (*ChunkerNode)(nil)

func (c *ChunkerNode) Transform(ctx context.Context, src []*schema.Document, opts ...document.TransformerOption) ([]*schema.Document, error) {
	// TODO implement me
	panic("implement me")
}
