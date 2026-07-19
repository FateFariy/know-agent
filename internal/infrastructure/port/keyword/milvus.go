package keyword

import (
	"context"

	retrievermilvus "github.com/cloudwego/eino-ext/components/retriever/milvus2"
	"github.com/cloudwego/eino-ext/components/retriever/milvus2/search_mode"
	"github.com/cloudwego/eino/components/retriever"
	"github.com/milvus-io/milvus/client/v2/milvusclient"

	cadapter "github.com/swiftbit/know-agent/internal/domain/chat/adapter"
	cvo "github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
	dadapter "github.com/swiftbit/know-agent/internal/domain/document/adapter"
	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
	"github.com/swiftbit/know-agent/internal/infrastructure/port/milvus"
	"github.com/swiftbit/know-agent/internal/svc"
)

// MilvusKeyword 关键词检索实现（sparse BM25）
type MilvusKeyword struct {
	*milvus.Base
}

var _ dadapter.KeywordIndexer = (*MilvusKeyword)(nil)
var _ cadapter.KeywordRetriever = (*MilvusKeyword)(nil)

func NewMilvusKeyword(svcCtx *svc.ServiceContext) *MilvusKeyword {
	ctx := context.Background()
	return &MilvusKeyword{
		Base: milvus.NewBase(svcCtx, newRetriever(svcCtx, ctx)),
	}
}

// BuildIndexes 批量索引分片（文档域）。分片文本已写入 document_chunk，此处仅标记可检索状态
func (m *MilvusKeyword) BuildIndexes(ctx context.Context, chunks []*entity.DocumentChunk) error {
	return nil
}

// SearchByKeyword 基于关键词进行检索
func (m *MilvusKeyword) SearchByKeyword(ctx context.Context, query *cvo.DocumentRetrieve) ([]*cvo.DocumentChunk, error) {
	return m.Search(ctx, query)
}

// 创建关键词检索器
func newRetriever(svcCtx *svc.ServiceContext, ctx context.Context) retriever.Retriever {
	keyRetriever, err := retrievermilvus.NewRetriever(ctx, &retrievermilvus.RetrieverConfig{
		ClientConfig: &milvusclient.ClientConfig{
			Address: svcCtx.Config.Milvus.Addr,
		},
		Collection: svcCtx.Config.Milvus.Collection,
		TopK:       10,
		SearchMode: search_mode.NewSparse(retrievermilvus.BM25),
	})
	if err != nil {
		panic(err)
	}
	return keyRetriever
}
