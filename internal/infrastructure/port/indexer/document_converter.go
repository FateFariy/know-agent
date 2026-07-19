package indexer

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudwego/eino-ext/components/indexer/milvus2"
	"github.com/cloudwego/eino/schema"
	"github.com/milvus-io/milvus/client/v2/column"

	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
)

func DocumentConverter(vector *milvus2.VectorConfig, _ *milvus2.SparseVectorConfig) func(ctx context.Context, docs []*schema.Document, vectors [][]float64) ([]column.Column, error) {
	return func(ctx context.Context, docs []*schema.Document, vectors [][]float64) ([]column.Column, error) {
		ids := make([]string, 0, len(docs))
		contents := make([]string, 0, len(docs))
		vecs := make([][]float32, 0, len(docs))
		documentIds := make([]int64, 0, len(docs))
		taskIds := make([]int64, 0, len(docs))
		planIds := make([]int64, 0, len(docs))
		parentBlockIds := make([]int64, 0, len(docs))
		chunkNos := make([]int32, 0, len(docs))
		sourceTypes := make([]int32, 0, len(docs))
		sectionPaths := make([]string, 0, len(docs))
		structureNodeIds := make([]int64, 0, len(docs))
		structureNodeTypes := make([]int32, 0, len(docs))
		canonicalPaths := make([]string, 0, len(docs))
		itemIndices := make([]int32, 0, len(docs))
		charCounts := make([]int32, 0, len(docs))
		tokenCounts := make([]int32, 0, len(docs))
		embeddingModels := make([]string, 0, len(docs))
		createTimes := make([]time.Time, 0, len(docs))
		updateTimes := make([]time.Time, 0, len(docs))
		statuses := make([]int8, 0, len(docs))

		// Determine if we need to handle dense vectors
		denseVectorField := ""
		if vector != nil {
			denseVectorField = vector.VectorField
		}

		for idx, doc := range docs {
			ids = append(ids, doc.ID)
			contents = append(contents, doc.Content)
			documentIds = append(documentIds, doc.MetaData[vo.MetaDocumentID].(int64))
			taskIds = append(taskIds, doc.MetaData[vo.MetaTaskID].(int64))
			planIds = append(planIds, doc.MetaData[vo.MetaPlanID].(int64))
			parentBlockIds = append(parentBlockIds, doc.MetaData[vo.MetaParentBlockID].(int64))
			chunkNos = append(chunkNos, doc.MetaData[vo.MetaChunkNo].(int32))
			sourceTypes = append(sourceTypes, doc.MetaData[vo.MetaSourceType].(int32))
			sectionPaths = append(sectionPaths, doc.MetaData[vo.MetaSectionPath].(string))
			structureNodeIds = append(structureNodeIds, doc.MetaData[vo.MetaStructureNodeID].(int64))
			structureNodeTypes = append(structureNodeTypes, doc.MetaData[vo.MetaStructureNodeType].(int32))
			canonicalPaths = append(canonicalPaths, doc.MetaData["canonicalPath"].(string))
			itemIndices = append(itemIndices, doc.MetaData["itemIndex"].(int32))
			charCounts = append(charCounts, doc.MetaData["charCount"].(int32))
			tokenCounts = append(tokenCounts, doc.MetaData["tokenCount"].(int32))
			embeddingModels = append(embeddingModels, doc.MetaData["embeddingModel"].(string))
			createTimes = append(createTimes, doc.MetaData["createTime"].(time.Time))
			updateTimes = append(updateTimes, doc.MetaData["updateTime"].(time.Time))
			statuses = append(statuses, doc.MetaData["status"].(int8))

			var sourceVec []float64
			if len(vectors) == len(docs) {
				sourceVec = vectors[idx]
			} else {
				sourceVec = doc.DenseVector()
			}

			// Dense indexer is required when vectorField is set (dense-only or hybrid mode).
			if denseVectorField != "" {
				if len(sourceVec) == 0 {
					return nil, fmt.Errorf("indexer data missing for document %d (id: %s)", idx, doc.ID)
				}
				vec := make([]float32, len(sourceVec))
				for i, v := range sourceVec {
					vec[i] = float32(v)
				}
				vecs = append(vecs, vec)
			}

		}

		columns := []column.Column{
			column.NewColumnVarChar("id", ids),
			column.NewColumnVarChar("content", contents),
			column.NewColumnInt64("document_id", documentIds),
			column.NewColumnInt64("task_id", taskIds),
			column.NewColumnInt64("plan_id", planIds),
			column.NewColumnInt64("parent_block_id", parentBlockIds),
			column.NewColumnInt32("chunk_no", chunkNos),
			column.NewColumnInt32("source_type", sourceTypes),
			column.NewColumnVarChar("section_path", sectionPaths),
			column.NewColumnInt64("structure_node_id", structureNodeIds),
			column.NewColumnInt32("structure_node_type", structureNodeTypes),
			column.NewColumnVarChar("canonical_path", canonicalPaths),
			column.NewColumnInt32("item_index", itemIndices),
			column.NewColumnInt32("char_count", charCounts),
			column.NewColumnInt32("token_count", tokenCounts),
			column.NewColumnVarChar("embedding_model", embeddingModels),
			column.NewColumnTimestamptz("create_time", createTimes),
			column.NewColumnTimestamptz("update_time", updateTimes),
			column.NewColumnInt8("status", statuses),
		}

		if denseVectorField != "" {
			dim := 0
			if len(vecs) > 0 {
				dim = len(vecs[0])
			}
			columns = append(columns, column.NewColumnFloatVector(denseVectorField, dim, vecs))
		}

		return columns, nil
	}
}
