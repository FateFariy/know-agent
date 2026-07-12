package adapter

import (
	"context"
	"log"
	"testing"

	"github.com/cloudwego/eino-ext/components/embedding/ark"
	"github.com/cloudwego/eino-ext/components/indexer/milvus2"
	"github.com/cloudwego/eino/schema"
	"github.com/milvus-io/milvus/client/v2/milvusclient"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
)

func TestIndex(t *testing.T) {
	// 获取环境变量
	addr := "10.104.1.173:19530"
	arkApiKey := ""
	arkModel := "doubao-embedding-vision-251215"

	ctx := context.Background()

	// 创建 embedding 模型
	emb, err := ark.NewEmbedder(ctx, &ark.EmbeddingConfig{
		APIKey:  arkApiKey,
		Model:   arkModel,
		APIType: utils.Pointer(ark.APITypeMultiModal),
	})
	if err != nil {
		log.Fatalf("Failed to create embedding: %v", err)
		return
	}

	// 创建索引器
	indexer, err := milvus2.NewIndexer(ctx, &milvus2.IndexerConfig{
		ClientConfig: &milvusclient.ClientConfig{
			Address: addr,
		},
		Collection: "test_duplicated",

		Vector: &milvus2.VectorConfig{
			Dimension:    2048, // 与 embedding 模型维度匹配
			MetricType:   milvus2.COSINE,
			IndexBuilder: milvus2.NewHNSWIndexBuilder().WithM(16).WithEfConstruction(200),
		},
		Embedding: emb,
	})
	if err != nil {
		log.Fatalf("Failed to create indexer: %v", err)
		return
	}
	log.Printf("Indexer created successfully")

	// 存储文档
	docs := []*schema.Document{
		{
			ID:      "doc1",
			Content: "Milvus is an open-source vector database",
			MetaData: map[string]any{
				"category": "database",
				"year":     2021,
			},
		},
		{
			ID:      "doc2",
			Content: "EINO is a framework for building AI applications",
		},
	}
	ids, err := indexer.Store(ctx, docs)
	if err != nil {
		log.Fatalf("Failed to store: %v", err)
		return
	}
	log.Printf("Store success, ids: %v", ids)
}

func TestQuery(t *testing.T) {
	debugTrace := vo.NewChatDebugTrace(nil)
	debugTrace.AddUsedChannels("embedding")
	log.Println(debugTrace.Serialize())
}
