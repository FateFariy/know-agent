package infrastructure

import (
	"github.com/google/wire"

	chatadapter "github.com/swiftbit/know-agent/internal/domain/chat/adapter"
	documentadapter "github.com/swiftbit/know-agent/internal/domain/document/adapter"
	"github.com/swiftbit/know-agent/internal/infrastructure/persistence"
	"github.com/swiftbit/know-agent/internal/infrastructure/port/mq"
	"github.com/swiftbit/know-agent/internal/infrastructure/port/storage"
	"github.com/swiftbit/know-agent/internal/infrastructure/port/vector"
)

var ProviderSet = wire.NewSet(
	persistence.NewDocumentRepository,
	wire.Bind(new(documentadapter.DocumentRepository), new(*persistence.DocumentRepositoryImpl)),
	persistence.NewChatRepository,
	wire.Bind(new(chatadapter.ChatRepository), new(*persistence.ChatRepositoryImpl)),
	storage.NewMinioStorage,
	wire.Bind(new(documentadapter.Storage), new(*storage.MinioStorage)),
	mq.NewMockMessageProducer,
	wire.Bind(new(documentadapter.MessageProducer), new(*mq.MockMessageProducer)),
	vector.NewMilvusVector,
	wire.Bind(new(documentadapter.VectorDB), new(*vector.MilvusVector)),
)
