package document

import (
	"github.com/google/wire"

	"github.com/swiftbit/know-agent/internal/domain/document/adapter"
	"github.com/swiftbit/know-agent/internal/domain/document/logic"
	"github.com/swiftbit/know-agent/internal/domain/document/logic/process"
	"github.com/swiftbit/know-agent/internal/domain/document/logic/process/chunk"
	"github.com/swiftbit/know-agent/internal/domain/document/logic/process/transform"
)

var ProviderSet = wire.NewSet(
	process.NewAsyncProcessImpl,
	wire.Bind(new(process.AsyncProcessor), new(*process.AsyncProcessImpl)),
	logic.NewProfileLogicImpl,
	wire.Bind(new(logic.ProfileLogic), new(*logic.ProfileLogicImpl)),
	logic.NewLifecycleLogicImpl,
	wire.Bind(new(logic.LifecycleLogic), new(*logic.LifecycleLogicImpl)),
	process.NewTextPreprocessImpl,
	wire.Bind(new(process.TextPreprocessor), new(*process.TextPreprocessImpl)),
	process.NewProfileGenerateImpl,
	wire.Bind(new(process.ProfileGenerator), new(*process.ProfileGenerateImpl)),
	adapter.NewDocumentPort,
	transform.NewAmbiguityResolver,
	transform.NewHierarchyResolver,
	transform.NewSignalExtractor,
	transform.NewTreeValidator,
	chunk.NewChunkStrategyRegistry,
)
