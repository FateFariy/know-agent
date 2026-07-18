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
	"github.com/duke-git/lancet/v2/convertor"
	"github.com/duke-git/lancet/v2/slice"
	"github.com/duke-git/lancet/v2/strutil"
	"github.com/milvus-io/milvus/client/v2/milvusclient"
	"github.com/zeromicro/go-zero/core/logx"

	"github.com/swiftbit/know-agent/common/utils"
	cadapter "github.com/swiftbit/know-agent/internal/domain/chat/adapter"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
	dadapter "github.com/swiftbit/know-agent/internal/domain/document/adapter"
	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
	dvo "github.com/swiftbit/know-agent/internal/domain/document/model/vo"
	"github.com/swiftbit/know-agent/internal/svc"
)

type MilvusVector struct {
	indexer    indexer.Indexer
	retriever  retriever.Retriever
	model      string
	client     *milvusclient.Client
	collection string
}

var _ dadapter.VectorRetriever = (*MilvusVector)(nil)
var _ cadapter.VectorRetriever = (*MilvusVector)(nil)

func NewMilvusVector(svcCtx *svc.ServiceContext) *MilvusVector {
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

// Vectorize 生成向量
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
	expr := fmt.Sprintf("document_id == %d", documentId)
	_, err := m.client.Delete(ctx, milvusclient.NewDeleteOption(m.collection).WithExpr(expr))
	return err
}

// SearchByVector 根据向量搜索
func (m *MilvusVector) SearchByVector(ctx context.Context, query *vo.DocumentRetrieve) ([]*vo.DocumentChunk, error) {
	filterExpr := m.buildFilterExpr(query)
	retrievedDocs, err := m.retriever.Retrieve(ctx, query.RetrievalQuery, retriever.WithTopK(query.TopK), retrievermilvus.WithFilter(filterExpr))
	if err != nil {
		return nil, err
	}
	return m.convertToDocumentChunks(retrievedDocs), nil
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

// buildFilterExpr 构建 Milvus 过滤表达式
//
// 表达式结构：
//
//	document_id in [..] AND task_id in [..]
//	    [ AND (section_path like "%h1%" or ...) ]       ← sectionPathHints
//	    [ AND structure_node_id in [..] ]               ← structureNodeIdHints
//	    [ AND (canonical_path like "%h1%" or ...) ]     ← canonicalPathHints
//	    [ AND item_index in [..] ]                      ← itemIndexHints
func (m *MilvusVector) buildFilterExpr(query *vo.DocumentRetrieve) string {
	var sb strings.Builder
	sb.WriteString("document_id in ")
	sb.WriteString(utils.Join(query.DocumentIds, "[", "]", ", "))
	sb.WriteString(" AND task_id in ")
	sb.WriteString(utils.Join(query.TaskIds, "[", "]", ", "))

	if query.Filters == nil {
		return sb.String()
	}

	// sectionPathHints（多个 hint 之间用 OR 拼接，模糊匹配 section_path）
	if clause := m.buildSectionFilter(query.Filters.SectionPathHints); clause != "" {
		sb.WriteString(clause)
	}
	// structureNodeIdHints / canonicalPathHints / itemIndexHints
	if clause := m.buildStructureFilter(query.Filters); clause != "" {
		sb.WriteString(clause)
	}

	return sb.String()
}

// buildSectionFilter 拼接 section_path hints 对应的过滤片段
//
// 返回形如 ` AND (section_path like "%h1%" OR section_path like "%h2%")`；无 hint 时返回空串。
func (m *MilvusVector) buildSectionFilter(hints []string) string {
	if len(hints) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString(" AND (")
	for i, hint := range hints {
		if i > 0 {
			sb.WriteString(" OR ")
		}
		sb.WriteString(`section_path like "%`)
		sb.WriteString(strings.ToLower(hint))
		sb.WriteString(`%"`)
	}
	sb.WriteString(")")
	return sb.String()
}

// buildStructureFilter 拼接 structure_node_id / canonical_path / item_index hints 对应的过滤片段
//
// 返回形如 ` AND structure_node_id in [..] AND (...) AND item_index in [..]`；所有 hint 都为空时返回空串。
func (m *MilvusVector) buildStructureFilter(filters *vo.DocumentRetrieveFilters) string {
	if filters == nil {
		return ""
	}
	var sb strings.Builder

	if len(filters.StructureNodeIdHints) > 0 {
		sb.WriteString(" AND structure_node_id in ")
		sb.WriteString(utils.Join(filters.StructureNodeIdHints, "[", "]", ", "))
	}

	if len(filters.CanonicalPathHints) > 0 {
		sb.WriteString(" AND (")
		for i, hint := range filters.CanonicalPathHints {
			if i > 0 {
				sb.WriteString(" OR ")
			}
			sb.WriteString(`canonical_path like "%`)
			sb.WriteString(strings.ToLower(hint))
			sb.WriteString(`%"`)
		}
		sb.WriteString(")")
	}

	if len(filters.ItemIndexHints) > 0 {
		sb.WriteString(" AND item_index in ")
		sb.WriteString(utils.Join(filters.ItemIndexHints, "[", "]", ", "))
	}

	return sb.String()
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
					vo.MetaDocumentID:        chunk.DocumentId,
					vo.MetaTaskID:            chunk.TaskId,
					vo.MetaPlanID:            chunk.PlanId,
					vo.MetaParentBlockID:     chunk.ParentBlockId,
					vo.MetaChunkNo:           int32(chunk.ChunkNo),
					vo.MetaSourceType:        int32(chunk.SourceType),
					vo.MetaSectionPath:       chunk.SectionPath,
					vo.MetaStructureNodeID:   chunk.StructureNodeId,
					vo.MetaStructureNodeType: int32(chunk.StructureNodeType),
					vo.MetaCanonicalPath:     chunk.CanonicalPath,
					vo.MetaItemIndex:         int32(chunk.ItemIndex),
					"charCount":              int32(chunk.CharCount),
					"tokenCount":             int32(chunk.TokenCount),
					"embeddingModel":         m.model,
					"createTime":             time.Now(),
					"updateTime":             time.Now(),
					"status":                 int8(0),
				},
			})
		}
	}

	return result
}

// 转换为文档分片列表
func (m *MilvusVector) convertToDocumentChunks(retrievedDocs []*schema.Document) []*vo.DocumentChunk {
	return slice.Map(retrievedDocs, func(_ int, doc *schema.Document) *vo.DocumentChunk {
		meta := doc.MetaData
		return &vo.DocumentChunk{
			ID:                doc.ID,
			Content:           doc.Content,
			OriginalSnippet:   doc.Content,
			SourceType:        "DOCUMENT",
			Channel:           vo.RetrievalChannelVector,
			Score:             doc.Score(),
			TaskId:            metaToInt(meta[vo.MetaTaskID]),
			DocumentId:        metaToInt(meta[vo.MetaDocumentID]),
			ChunkNo:           int(metaToInt(meta[vo.MetaChunkNo])),
			ParentBlockId:     metaToInt(meta[vo.MetaParentBlockID]),
			SectionPath:       convertor.ToString(meta[vo.MetaSectionPath]),
			StructureNodeId:   metaToInt(meta[vo.MetaStructureNodeID]),
			StructureNodeType: int(metaToInt(meta[vo.MetaStructureNodeType])),
			CanonicalPath:     convertor.ToString(meta[vo.MetaCanonicalPath]),
			ItemIndex:         int(metaToInt(meta[vo.MetaItemIndex])),
		}
	})
}

// 创建索引器
func newIndexer(svcCtx *svc.ServiceContext, ctx context.Context, emb embedding.Embedder) indexer.Indexer {
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
		Embedding: emb,
	}
	indexerConfig.DocumentConverter = DocumentConverter(indexerConfig.Vector, indexerConfig.Sparse)
	vecIndexer, err := indexmilvus.NewIndexer(ctx, indexerConfig)
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

func metaToInt(v any) int64 {
	value, _ := convertor.ToInt(v)
	return value
}
