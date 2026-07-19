// Package milvus 抽取向量检索与关键词检索共享的 Milvus 基础设施。
//
// 设计要点：
//  1. 领域端口（VectorRetriever / KeywordRetriever）保持独立，按能力而非后端切分。
//  2. Base 持有 Milvus 客户端与 collection，同时提供过滤表达式构建、metadata 转换等
//     同后端实现共享的逻辑，避免 vector/milvus.go 与 keyword/milvus.go 重复。
//  3. 未来 keyword 切到 Elasticsearch 时只需新增 keyword/es.go 直接实现
//     KeywordRetriever 接口，不依赖 Base，实现真正的"同能力可换后端"。
package milvus

import (
	"context"
	"fmt"
	"strings"

	retrievermilvus "github.com/cloudwego/eino-ext/components/retriever/milvus2"
	"github.com/cloudwego/eino/components/retriever"
	"github.com/cloudwego/eino/schema"
	"github.com/duke-git/lancet/v2/convertor"
	"github.com/duke-git/lancet/v2/slice"
	"github.com/milvus-io/milvus/client/v2/milvusclient"

	"github.com/swiftbit/know-agent/common/utils"
	cvo "github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
	"github.com/swiftbit/know-agent/internal/svc"
)

// Base Milvus 共享基础设施
type Base struct {
	client     *milvusclient.Client
	retriever  retriever.Retriever
	collection string
}

func NewBase(svcCtx *svc.ServiceContext, retriever retriever.Retriever) *Base {
	return &Base{
		client:     svcCtx.Milvus,
		retriever:  retriever,
		collection: svcCtx.Config.Milvus.Collection,
	}
}

func (b *Base) DeleteByDocumentId(ctx context.Context, documentId int64) error {
	expr := fmt.Sprintf("document_id == %d", documentId)
	_, err := b.client.Delete(ctx, milvusclient.NewDeleteOption(b.collection).WithExpr(expr))
	return err
}

// Search 检索
func (b *Base) Search(ctx context.Context, query *cvo.DocumentRetrieve) ([]*cvo.DocumentChunk, error) {
	filterExpr := b.buildFilterExpr(query)
	retrievedDocs, err := b.retriever.Retrieve(ctx, query.RetrievalQuery, retriever.WithTopK(query.TopK), retrievermilvus.WithFilter(filterExpr))
	if err != nil {
		return nil, err
	}
	return b.convertToDocumentChunks(retrievedDocs), nil
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
//
// 说明：Milvus 的 like 区分大小写；hint 在拼接前统一转小写以贴近 Java 版本 LOWER() 语义。
func (b *Base) buildFilterExpr(query *cvo.DocumentRetrieve) string {
	var sb strings.Builder
	sb.WriteString("document_id in ")
	sb.WriteString(utils.Join(query.DocumentIds, "[", "]", ", "))
	sb.WriteString(" AND task_id in ")
	sb.WriteString(utils.Join(query.TaskIds, "[", "]", ", "))

	if query.Filters == nil {
		return sb.String()
	}

	// sectionPathHints（多个 hint 之间用 OR 拼接，模糊匹配 section_path）
	if clause := b.buildSectionFilter(query.Filters.SectionPathHints); clause != "" {
		sb.WriteString(clause)
	}
	// structureNodeIdHints / canonicalPathHints / itemIndexHints
	if clause := b.buildStructureFilter(query.Filters); clause != "" {
		sb.WriteString(clause)
	}

	return sb.String()
}

// buildSectionFilter 拼接 section_path hints 对应的过滤片段
//
// 返回形如 ` AND (section_path like "%h1%" OR section_path like "%h2%")`；无 hint 时返回空串。
func (b *Base) buildSectionFilter(hints []string) string {
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
func (b *Base) buildStructureFilter(filters *cvo.DocumentRetrieveFilters) string {
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
		ids := make([]int64, len(filters.ItemIndexHints))
		for i, v := range filters.ItemIndexHints {
			ids[i] = int64(v)
		}
		sb.WriteString(utils.Join(ids, "[", "]", ", "))
	}

	return sb.String()
}

// convertToDocumentChunks 将 Milvus 返回的 *schema.Document 列表转换为聊天域 DocumentChunk 列表
func (b *Base) convertToDocumentChunks(retrievedDocs []*schema.Document) []*cvo.DocumentChunk {
	return slice.Map(retrievedDocs, func(_ int, doc *schema.Document) *cvo.DocumentChunk {
		meta := doc.MetaData
		return &cvo.DocumentChunk{
			ID:                doc.ID,
			Content:           doc.Content,
			OriginalSnippet:   doc.Content,
			SourceType:        "DOCUMENT",
			Channel:           cvo.RetrievalChannelVector,
			Score:             doc.Score(),
			TaskId:            b.metaToInt(meta[cvo.MetaTaskID]),
			DocumentId:        b.metaToInt(meta[cvo.MetaDocumentID]),
			ChunkNo:           int(b.metaToInt(meta[cvo.MetaChunkNo])),
			ParentBlockId:     b.metaToInt(meta[cvo.MetaParentBlockID]),
			SectionPath:       convertor.ToString(meta[cvo.MetaSectionPath]),
			StructureNodeId:   b.metaToInt(meta[cvo.MetaStructureNodeID]),
			StructureNodeType: int(b.metaToInt(meta[cvo.MetaStructureNodeType])),
			CanonicalPath:     convertor.ToString(meta[cvo.MetaCanonicalPath]),
			ItemIndex:         int(b.metaToInt(meta[cvo.MetaItemIndex])),
		}
	})
}

// metaToInt 将 metadata 中的数值字段安全转换为 int64，转换失败返回 0
func (b *Base) metaToInt(v any) int64 {
	value, _ := convertor.ToInt(v)
	return value
}
