package infrastructure

import (
	"github.com/google/wire"

	chatadapter "github.com/swiftbit/know-agent/internal/domain/chat/adapter"
	documentadapter "github.com/swiftbit/know-agent/internal/domain/document/adapter"
	knowledgeadapter "github.com/swiftbit/know-agent/internal/domain/knowledge/adapter"
	"github.com/swiftbit/know-agent/internal/infrastructure/persistence"
	"github.com/swiftbit/know-agent/internal/infrastructure/port/indexer"
	"github.com/swiftbit/know-agent/internal/infrastructure/port/keyword"
	"github.com/swiftbit/know-agent/internal/infrastructure/port/lock"
	"github.com/swiftbit/know-agent/internal/infrastructure/port/mq"
	"github.com/swiftbit/know-agent/internal/infrastructure/port/reranker"
	"github.com/swiftbit/know-agent/internal/infrastructure/port/storage"
)

var ProviderSet = wire.NewSet(
	persistence.NewDocumentRepository,
	wire.Bind(new(documentadapter.DocumentRepository), new(*persistence.DocumentRepositoryImpl)),
	persistence.NewChatRepository,
	wire.Bind(new(chatadapter.ChatRepository), new(*persistence.ChatRepositoryImpl)),
	persistence.NewKnowledgeRepository,
	wire.Bind(new(knowledgeadapter.KnowledgeRepository), new(*persistence.KnowledgeRepositoryImpl)),
	storage.NewMinioStorage,
	wire.Bind(new(documentadapter.Storage), new(*storage.MinioStorage)),
	mq.NewRocketMQMessageProducer,
	wire.Bind(new(documentadapter.MessageProducer), new(*mq.RocketMQMessageProducer)),
	indexer.NewMilvusVector,
	wire.Bind(new(documentadapter.VectorIndexer), new(*indexer.MilvusVector)),
	wire.Bind(new(chatadapter.Retriever), new(*indexer.MilvusVector)),
	keyword.NewMilvusKeyword,
	wire.Bind(new(documentadapter.KeywordIndexer), new(*keyword.MilvusKeyword)),
	wire.Bind(new(chatadapter.KeywordRetriever), new(*keyword.MilvusKeyword)),
	lock.NewRedisMutexLock,
	wire.Bind(new(chatadapter.DistributedLock), new(*lock.RedisMutexLock)),
	reranker.NewDashScope,
	wire.Bind(new(chatadapter.Reranker), new(*reranker.DashScope)),
)
