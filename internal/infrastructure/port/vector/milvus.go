package vector

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	indexmilvus "github.com/cloudwego/eino-ext/components/indexer/milvus2"
	retrievermilvus "github.com/cloudwego/eino-ext/components/retriever/milvus2"
	"github.com/cloudwego/eino-ext/components/retriever/milvus2/search_mode"
	"github.com/cloudwego/eino/components/embedding"
	"github.com/cloudwego/eino/components/indexer"
	"github.com/cloudwego/eino/components/retriever"
	"github.com/cloudwego/eino/schema"
	"github.com/duke-git/lancet/v2/strutil"
	"github.com/milvus-io/milvus/client/v2/milvusclient"
	"github.com/zeromicro/go-zero/core/logx"

	"github.com/swiftbit/know-agent/internal/domain/document/adapter"
	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/document/model/vo"
	"github.com/swiftbit/know-agent/internal/svc"
)

type MilvusVector struct {
	indexer    indexer.Indexer
	retriever  retriever.Retriever
	model      string
	client     *milvusclient.Client
	collection string
}

var _ adapter.VectorDB = (*MilvusVector)(nil)

func NewMilvusVector(svcCtx *svc.ServiceContext) adapter.VectorDB {
	ctx := context.Background()
	client, err := milvusclient.New(ctx, &milvusclient.ClientConfig{Address: svcCtx.Config.Milvus.Addr})
	if err != nil {
		panic(err)
	}
	return &MilvusVector{
		indexer:    newIndexer(svcCtx, ctx, svcCtx.Emb),
		retriever:  newRetriever(svcCtx, ctx, svcCtx.Emb),
		model:      svcCtx.Config.Embedding.Model,
		client:     client,
		collection: svcCtx.Config.Milvus.Collection,
	}
}

func (m *MilvusVector) Vectorize(ctx context.Context, chunks []*entity.DocumentChunk) error {
	docs := m.toDocument(chunks)
	if len(docs) == 0 {
		return nil
	}
	_, err := m.indexer.Store(ctx, docs)
	if err != nil {
		return err
	}
	m.markSuccess(chunks)
	logx.Infof("向量生成成功, chunkCount:%d, model:%s", len(docs), m.model)
	return nil
}

// DeleteVectorByDocumentId 根据文档ID删除向量
func (m *MilvusVector) DeleteVectorByDocumentId(ctx context.Context, documentId int64) error {
	expr := fmt.Sprintf("metadata['documentId'] == %d", documentId)
	_, err := m.client.Delete(ctx, milvusclient.NewDeleteOption(m.collection).WithExpr(expr))
	return err
}

// markSuccess 批量标记分片向量生成成功
func (m *MilvusVector) markSuccess(chunks []*entity.DocumentChunk) {
	for _, chunk := range chunks {
		if chunk != nil && strutil.IsNotBlank(chunk.ChunkText) {
			chunk.VectorId = strconv.FormatInt(chunk.ID, 10)
			chunk.VectorStoreType = vo.VectorStoreTypeMilvus
			chunk.VectorStatus = vo.VectorStatusVectorSuccess
		}
	}
}

// toDocument 转换为文档
func (m *MilvusVector) toDocument(chunks []*entity.DocumentChunk) []*schema.Document {
	result := make([]*schema.Document, 0, len(chunks))
	for _, chunk := range chunks {
		if chunk != nil && strutil.IsNotBlank(chunk.ChunkText) {
			result = append(result, &schema.Document{
				ID:      strconv.FormatInt(chunk.ID, 10),
				Content: chunk.ChunkText,
				MetaData: map[string]any{
					"documentId":        chunk.DocumentId,
					"taskId":            chunk.TaskId,
					"planId":            chunk.PlanId,
					"parentBlockId":     chunk.ParentBlockId,
					"chunkNo":           chunk.ChunkNo,
					"sourceType":        chunk.SourceType,
					"sectionPath":       chunk.SectionPath,
					"structureNodeId":   chunk.StructureNodeId,
					"structureNodeType": chunk.StructureNodeType,
					"canonicalPath":     chunk.CanonicalPath,
					"itemIndex":         chunk.ItemIndex,
					"charCount":         chunk.CharCount,
					"tokenCount":        chunk.TokenCount,
					"embeddingModel":    m.model,
					"createTime":        time.Now().Format("2006-01-02 15:04:05"),
					"updateTime":        time.Now().Format("2006-01-02 15:04:05"),
					"status":            1,
				},
			})
		}
	}

	return result
}

// 创建索引器
func newIndexer(svcCtx *svc.ServiceContext, ctx context.Context, emb embedding.Embedder) indexer.Indexer {
	vecIndexer, err := indexmilvus.NewIndexer(ctx, &indexmilvus.IndexerConfig{
		ClientConfig: &milvusclient.ClientConfig{
			Address: svcCtx.Config.Milvus.Addr,
		},
		Collection: svcCtx.Config.Milvus.Collection,
		Vector: &indexmilvus.VectorConfig{
			Dimension:    int64(svcCtx.Config.Embedding.Dimensions),
			MetricType:   indexmilvus.MetricType(strings.ToUpper(svcCtx.Config.Milvus.MetricType)),
			IndexBuilder: indexmilvus.NewHNSWIndexBuilder().WithM(16).WithEfConstruction(200),
		},
		Embedding: emb,
	})
	if err != nil {
		panic(err)
	}
	return vecIndexer
}

// 创建向量检索器
func newRetriever(svcCtx *svc.ServiceContext, ctx context.Context, emb embedding.Embedder) retriever.Retriever {
	metricType := retrievermilvus.MetricType(strings.ToUpper(svcCtx.Config.Milvus.MetricType))
	vecRetriever, err := retrievermilvus.NewRetriever(ctx, &retrievermilvus.RetrieverConfig{
		ClientConfig: &milvusclient.ClientConfig{
			Address: svcCtx.Config.Milvus.Addr,
		},
		Collection: svcCtx.Config.Milvus.Collection,
		TopK:       10,
		SearchMode: search_mode.NewApproximate(metricType),
		Embedding:  emb,
	})
	if err != nil {
		panic(err)
	}
	return vecRetriever
}
