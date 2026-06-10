package ingestion

import (
	"context"

	"github.com/cloudwego/eino/components/document"
	"github.com/cloudwego/eino/schema"
)

type LoaderNode struct {
}

var _ document.Loader = (*LoaderNode)(nil)

func NewLoaderNode() *LoaderNode {
	return &LoaderNode{}
}

func (l *LoaderNode) Load(ctx context.Context, src document.Source, opts ...document.LoaderOption) ([]*schema.Document, error) {
	// TODO implement me
	panic("implement me")
}
