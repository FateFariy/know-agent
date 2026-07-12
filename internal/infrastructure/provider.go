package infrastructure

import (
	"github.com/google/wire"

	chatadapter "github.com/swiftbit/know-agent/internal/domain/chat/adapter"
	documentadapter "github.com/swiftbit/know-agent/internal/domain/document/adapter"
	knowledgeadapter "github.com/swiftbit/know-agent/internal/domain/knowledge/adapter"
	"github.com/swiftbit/know-agent/internal/infrastructure/persistence"
	"github.com/swiftbit/know-agent/internal/infrastructure/port/keyword"
	"github.com/swiftbit/know-agent/internal/infrastructure/port/lock"
	"github.com/swiftbit/know-agent/internal/infrastructure/port/mq"
	"github.com/swiftbit/know-agent/internal/infrastructure/port/reranker"
	"github.com/swiftbit/know-agent/internal/infrastructure/port/storage"
	"github.com/swiftbit/know-agent/internal/infrastructure/port/vector"
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
	mq.NewMockMessageProducer,
	wire.Bind(new(documentadapter.MessageProducer), new(*mq.MockMessageProducer)),
	vector.NewMilvusVector,
	wire.Bind(new(documentadapter.VectorDB), new(*vector.MilvusVector)),
	wire.Bind(new(chatadapter.VectorDB), new(*vector.MilvusVector)),
	keyword.NewKeywordDB,
	wire.Bind(new(documentadapter.KeywordDB), new(*keyword.KeywordDB)),
	wire.Bind(new(chatadapter.KeywordDB), new(*keyword.KeywordDB)),
	lock.NewRedisMutexLock,
	wire.Bind(new(chatadapter.DistributedLock), new(*lock.RedisMutexLock)),
	reranker.NewDashScope,
	wire.Bind(new(chatadapter.Reranker), new(*reranker.DashScope)),
)
