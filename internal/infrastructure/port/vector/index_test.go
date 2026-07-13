package vector

import (
	"context"
	"log"
	"testing"

	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
)

func TestIndex(t *testing.T) {
	indexer := NewMilvusVector(svcCtx)
	ctx := context.Background()

	chunks := []*entity.DocumentChunk{
		{
			ID:                 1,
			DocumentId:         10001,
			TaskId:             5001,
			PlanId:             2001,
			ParentBlockId:      0,
			ChunkNo:            1,
			SourceType:         1,
			SectionPath:        "第一章/第一节",
			StructureNodeId:    8001,
			StructureNodeType:  1,
			CanonicalPath:      "/doc/10001/chapter1/section1",
			ItemIndex:          0,
			ChunkText:          "大模型RAG检索优化方案：采用分层分块策略，提升向量匹配精准度，减少幻觉问题。",
			CharCount:          62,
			TokenCount:         18,
			VectorStatus:       2,
			VectorStoreType:    1,
			VectorId:           "vec_10001_001",
			ParentBlockNo:      0,
			ParentChildCount:   5,
			ParentStartChunkNo: 1,
			ParentEndChunkNo:   5,
			SourceTypeName:     "文档正文",
			VectorStatusName:   "向量生成完成",
		},
		{
			ID:                 2,
			DocumentId:         10001,
			TaskId:             5001,
			PlanId:             2001,
			ParentBlockId:      1,
			ChunkNo:            2,
			SourceType:         2,
			SectionPath:        "第一章/第一节/数据表",
			StructureNodeId:    8002,
			StructureNodeType:  2,
			CanonicalPath:      "/doc/10001/chapter1/section1/table1",
			ItemIndex:          1,
			ChunkText:          "分块尺寸对照表：短块200token、标准块500token、长块1000token，适配不同检索场景。",
			CharCount:          58,
			TokenCount:         16,
			VectorStatus:       1,
			VectorStoreType:    2,
			VectorId:           "",
			ParentBlockNo:      1,
			ParentChildCount:   5,
			ParentStartChunkNo: 1,
			ParentEndChunkNo:   5,
			SourceTypeName:     "表格内容",
			VectorStatusName:   "等待向量化",
		},
	}
	err := indexer.BuildVectors(ctx, chunks)
	if err != nil {
		log.Fatalf("Failed to store: %v", err)
		return
	}
}
