package ingestion

import (
	"context"

	"github.com/cloudwego/eino/components/indexer"
	"github.com/cloudwego/eino/schema"
)

type IndexerNode struct {
}

var _ indexer.Indexer = (*IndexerNode)(nil)

func NewIndexerNode() *IndexerNode {
	return &IndexerNode{}
}

func (i *IndexerNode) Store(ctx context.Context, docs []*schema.Document, opts ...indexer.Option) (ids []string, err error) {
	// TODO implement me
	panic("implement me")
}
