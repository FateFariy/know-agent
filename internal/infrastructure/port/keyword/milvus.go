package keyword

import (
	"context"

	retrievermilvus "github.com/cloudwego/eino-ext/components/retriever/milvus2"
	"github.com/cloudwego/eino-ext/components/retriever/milvus2/search_mode"
	"github.com/cloudwego/eino/components/retriever"
	"github.com/milvus-io/milvus/client/v2/milvusclient"

	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
	"github.com/swiftbit/know-agent/internal/infrastructure/port/milvus"
	"github.com/swiftbit/know-agent/internal/svc"
)

// MilvusKeyword 关键词检索实现（sparse BM25）
type MilvusKeyword struct {
	*milvus.Base
}

func NewMilvusKeyword(svcCtx *svc.ServiceContext) *MilvusKeyword {
	ctx := context.Background()
	return &MilvusKeyword{
		Base: milvus.NewBase(svcCtx, newRetriever(svcCtx, ctx)),
	}
}

func (m *MilvusKeyword) BuildIndexes(ctx context.Context, chunks []*entity.DocumentChunk) error {
	// todo 向量、关键字均使用Milus，暂时不用实现
	return nil
}

func (m *MilvusKeyword) SearchByKeyword(ctx context.Context, query *vo.DocumentRetrieve) ([]*vo.DocumentChunk, error) {
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
