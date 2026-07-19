package vector

import (
	"context"
	"strconv"
	"strings"
	"time"

	indexmilvus "github.com/cloudwego/eino-ext/components/indexer/milvus2"
	retrievermilvus "github.com/cloudwego/eino-ext/components/retriever/milvus2"
	"github.com/cloudwego/eino-ext/components/retriever/milvus2/search_mode"
	"github.com/cloudwego/eino/components/indexer"
	"github.com/cloudwego/eino/components/retriever"
	"github.com/cloudwego/eino/schema"
	"github.com/duke-git/lancet/v2/strutil"
	"github.com/milvus-io/milvus/client/v2/milvusclient"
	"github.com/zeromicro/go-zero/core/logx"

	cvo "github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
	dvo "github.com/swiftbit/know-agent/internal/domain/document/model/vo"
	"github.com/swiftbit/know-agent/internal/infrastructure/port/milvus"
	"github.com/swiftbit/know-agent/internal/svc"
)

// MilvusVector 关键词检索实现（sparse BM25）
type MilvusVector struct {
	*milvus.Base
	indexer indexer.Indexer
	model   string
}

func NewMilvusVector(svcCtx *svc.ServiceContext) *MilvusVector {
	return &MilvusVector{
		Base:    milvus.NewBase(svcCtx, newRetriever(svcCtx)),
		indexer: newIndexer(svcCtx),
		model:   svcCtx.Config.Embedding.Model,
	}
}

// BuildVectors 生成向量
func (m *MilvusVector) BuildVectors(ctx context.Context, chunks []*entity.DocumentChunk) error {
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

func (m *MilvusVector) SearchByVector(ctx context.Context, query *cvo.DocumentRetrieve) ([]*cvo.DocumentChunk, error) {
	return m.Search(ctx, query)
}

// markSuccess 批量标记分片向量生成成功
func (m *MilvusVector) markSuccess(chunks []*entity.DocumentChunk) {
	for _, chunk := range chunks {
		if chunk != nil && strutil.IsNotBlank(chunk.ChunkText) {
			chunk.VectorId = strconv.FormatInt(chunk.ID, 10)
			chunk.VectorStoreType = dvo.VectorStoreTypeMilvus
			chunk.VectorStatus = dvo.VectorStatusVectorSuccess
		}
	}
}

// toDocument 转换为文档（向量写入专用）
func (m *MilvusVector) toDocument(chunks []*entity.DocumentChunk) []*schema.Document {
	result := make([]*schema.Document, 0, len(chunks))
	for _, chunk := range chunks {
		if chunk != nil && strutil.IsNotBlank(chunk.ChunkText) {
			result = append(result, &schema.Document{
				ID:      strconv.FormatInt(chunk.ID, 10),
				Content: chunk.ChunkText,
				MetaData: map[string]any{
					cvo.MetaDocumentID:        chunk.DocumentId,
					cvo.MetaTaskID:            chunk.TaskId,
					cvo.MetaPlanID:            chunk.PlanId,
					cvo.MetaParentBlockID:     chunk.ParentBlockId,
					cvo.MetaChunkNo:           int32(chunk.ChunkNo),
					cvo.MetaSourceType:        int32(chunk.SourceType),
					cvo.MetaSectionPath:       chunk.SectionPath,
					cvo.MetaStructureNodeID:   chunk.StructureNodeId,
					cvo.MetaStructureNodeType: int32(chunk.StructureNodeType),
					cvo.MetaCanonicalPath:     chunk.CanonicalPath,
					cvo.MetaItemIndex:         int32(chunk.ItemIndex),
					"charCount":               int32(chunk.CharCount),
					"tokenCount":              int32(chunk.TokenCount),
					"embeddingModel":          m.model,
					"createTime":              time.Now(),
					"updateTime":              time.Now(),
					"status":                  int8(0),
				},
			})
		}
	}

	return result
}

// 创建索引器
func newIndexer(svcCtx *svc.ServiceContext) indexer.Indexer {
	indexerConfig := &indexmilvus.IndexerConfig{
		ClientConfig: &milvusclient.ClientConfig{
			Address: svcCtx.Config.Milvus.Addr,
		},
		Collection: svcCtx.Config.Milvus.Collection,
		Vector: &indexmilvus.VectorConfig{
			Dimension:    int64(svcCtx.Config.Embedding.Dimensions),
			MetricType:   indexmilvus.MetricType(strings.ToUpper(svcCtx.Config.Milvus.MetricType)),
			IndexBuilder: indexmilvus.NewHNSWIndexBuilder().WithM(16).WithEfConstruction(200),
		},
		Sparse:    &indexmilvus.SparseVectorConfig{},
		Embedding: svcCtx.Emb,
	}
	indexerConfig.DocumentConverter = DocumentConverter(indexerConfig.Vector, indexerConfig.Sparse)
	vecIndexer, err := indexmilvus.NewIndexer(context.Background(), indexerConfig)
	if err != nil {
		panic(err)
	}
	return vecIndexer
}

// 创建向量检索器
func newRetriever(svcCtx *svc.ServiceContext) retriever.Retriever {
	metricType := retrievermilvus.MetricType(strings.ToUpper(svcCtx.Config.Milvus.MetricType))
	vecRetriever, err := retrievermilvus.NewRetriever(context.Background(), &retrievermilvus.RetrieverConfig{
		ClientConfig: &milvusclient.ClientConfig{
			Address: svcCtx.Config.Milvus.Addr,
		},
		Collection: svcCtx.Config.Milvus.Collection,
		TopK:       10,
		SearchMode: search_mode.NewApproximate(metricType),
		Embedding:  svcCtx.Emb,
	})
	if err != nil {
		panic(err)
	}
	return vecRetriever
}
