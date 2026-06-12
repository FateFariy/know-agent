package domain

import (
	"github.com/google/wire"

	"github.com/swiftbit/know-agent/internal/domain/document/adapter"
	"github.com/swiftbit/know-agent/internal/domain/document/logic"
)

var ProviderSet = wire.NewSet(
	logic.NewDocumentLifecycleLogicImpl,
	wire.Bind(new(logic.DocumentLifecycleLogic), new(*logic.DocumentLifecycleLogicImpl)),
	adapter.NewDocumentPort,
)
