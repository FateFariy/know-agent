package infrastructure

import (
	"github.com/cloudwego/eino/schema"
	"github.com/google/wire"

	"github.com/swiftbit/know-agent/internal/domain/chat/adapter/checkpoint"
	lock2 "github.com/swiftbit/know-agent/internal/domain/chat/adapter/lock"
	"github.com/swiftbit/know-agent/internal/domain/chat/adapter/model"
	reranker2 "github.com/swiftbit/know-agent/internal/domain/chat/adapter/reranker"
	"github.com/swiftbit/know-agent/internal/domain/chat/adapter/retriever"
	parse2 "github.com/swiftbit/know-agent/internal/domain/document/logic/process/parse"
	"github.com/swiftbit/know-agent/internal/infrastructure/port/check"
	"github.com/swiftbit/know-agent/internal/infrastructure/port/llm"
	"github.com/swiftbit/know-agent/internal/infrastructure/port/parser"
	"github.com/swiftbit/know-agent/internal/infrastructure/port/parser/markdown"

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
	persistence.NewTableRepository,
	wire.Bind(new(documentadapter.TableRepository), new(*persistence.TableRepositoryImpl)),
	persistence.NewChatRepository,
	wire.Bind(new(chatadapter.ChatRepository), new(*persistence.ChatRepositoryImpl)),
	persistence.NewKnowledgeRepository,
	wire.Bind(new(knowledgeadapter.KnowledgeRepository), new(*persistence.KnowledgeRepositoryImpl)),
	storage.NewMinioStorage,
	wire.Bind(new(documentadapter.Storage), new(*storage.MinioStorage)),
	mq.NewRocketMQMessageProducer,
	wire.Bind(new(documentadapter.MessageProducer), new(*mq.RocketMQMessageProducer)),
	keyword.NewMilvusKeyword,
	wire.Bind(new(documentadapter.KeywordIndexer), new(*keyword.MilvusKeyword)),
	wire.Bind(new(retriever.KeywordRetriever), new(*keyword.MilvusKeyword)),
	vector.NewMilvusVector,
	wire.Bind(new(documentadapter.VectorIndexer), new(*vector.MilvusVector)),
	wire.Bind(new(retriever.VectorRetriever), new(*vector.MilvusVector)),
	lock.NewRedisMutexLock,
	wire.Bind(new(lock2.DistributedLock), new(*lock.RedisMutexLock)),
	reranker.NewDashScope,
	wire.Bind(new(reranker2.Reranker), new(*reranker.DashScope)),
	llm.NewChatModelImpl,
	wire.Bind(new(model.ChatModel), new(*llm.ChatModelImpl[*schema.AgenticMessage])),
	check.NewMemoryCheckPointStore,
	wire.Bind(new(checkpoint.Store), new(*check.MemoryCheckPointStore)),
	NewParserRegistry,
)

func NewParserRegistry() *parse2.Registry {
	fallbackParser := &parser.TextParser{}
	parsers := []parse2.Parser{
		&parser.HTMLParser{},
		&parser.TextParser{},
		&parser.PDFParser{},
		&markdown.GoldmarkParser{},
	}
	return parse2.NewRegistry(fallbackParser, parsers...)
}
