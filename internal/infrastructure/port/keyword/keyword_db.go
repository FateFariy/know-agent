package keyword

import (
	"context"
	"fmt"

	"github.com/duke-git/lancet/v2/strutil"
	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"

	cadapter "github.com/swiftbit/know-agent/internal/domain/chat/adapter"
	cvo "github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
	dadapter "github.com/swiftbit/know-agent/internal/domain/document/adapter"
	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
	"github.com/swiftbit/know-agent/internal/svc"
)

type KeywordDB struct {
	db *gorm.DB
}

var _ dadapter.KeywordDB = (*KeywordDB)(nil)
var _ cadapter.KeywordDB = (*KeywordDB)(nil)

func NewKeywordDB(svcCtx *svc.ServiceContext) *KeywordDB {
	return &KeywordDB{db: svcCtx.Db}
}

// IndexChunks 批量索引分片（文档域）。分片文本已写入 document_chunk，此处仅标记可检索状态
func (k *KeywordDB) IndexChunks(ctx context.Context, chunks []*entity.DocumentChunk) error {
	if len(chunks) == 0 {
		return nil
	}
	count := 0
	for _, chunk := range chunks {
		if chunk != nil && strutil.IsNotBlank(chunk.ChunkText) {
			count++
		}
	}
	logx.Infof("关键词索引完成, chunkCount:%d", count)
	return nil
}

// DeleteIndexByDocumentId 根据文档ID删除关键词索引（文档域）
func (k *KeywordDB) DeleteIndexByDocumentId(ctx context.Context, documentId int64) error {
	logx.Infof("关键词索引删除完成, documentId:%d", documentId)
	return nil
}

// SearchByKeyword 基于关键词进行检索（聊天域）
func (k *KeywordDB) SearchByKeyword(ctx context.Context, query *cvo.DocumentRetrieve) ([]*cvo.DocumentChunk, error) {
	if query == nil || strutil.IsBlank(query.RetrievalQuery) {
		return nil, fmt.Errorf("invaild value")
	}

	limit := query.TopK
	if limit <= 0 {
		limit = 10
	}

	type row struct {
		ID                int64
		ChunkText         string
		DocumentId        int64
		TaskId            int64
		ChunkNo           int
		ParentBlockId     int64
		SectionPath       string
		StructureNodeId   int64
		StructureNodeType int
		CanonicalPath     string
		ItemIndex         int
	}

	var rows []*row
	search := "%" + query.RetrievalQuery + "%"
	err := k.db.WithContext(ctx).
		Table("document_chunk").
		Select("id, chunk_text, document_id, task_id, chunk_no, parent_block_id, section_path, structure_node_id, structure_node_type, canonical_path, item_index").
		Where("chunk_text LIKE ?", search).
		Limit(limit).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	result := make([]*cvo.DocumentChunk, 0, len(rows))
	for _, r := range rows {
		result = append(result, &cvo.DocumentChunk{
			ID:                fmt.Sprintf("%d", r.ID),
			Content:           r.ChunkText,
			OriginalSnippet:   r.ChunkText,
			SourceType:        "DOCUMENT",
			Channel:           cvo.RetrievalChannelKeyword,
			Score:             0,
			TaskId:            r.TaskId,
			DocumentId:        r.DocumentId,
			ChunkNo:           r.ChunkNo,
			ParentBlockId:     r.ParentBlockId,
			SectionPath:       r.SectionPath,
			StructureNodeId:   r.StructureNodeId,
			StructureNodeType: r.StructureNodeType,
			CanonicalPath:     r.CanonicalPath,
			ItemIndex:         r.ItemIndex,
		})
	}
	return result, nil
}
