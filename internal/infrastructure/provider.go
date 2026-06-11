package infrastructure

import (
	"github.com/google/wire"

	"github.com/swiftbit/know-agent/internal/domain/document/adapter"
	"github.com/swiftbit/know-agent/internal/infrastructure/persistence"
	"github.com/swiftbit/know-agent/internal/infrastructure/port/storage"
)

var ProviderSet = wire.NewSet(
	persistence.NewDocumentRepository,
	wire.Bind(new(adapter.DocumentRepository), new(*persistence.DocumentRepositoryImpl)),
	storage.NewMinioStorage,
	wire.Bind(new(adapter.Storage), new(*storage.MinioStorage)),
)
