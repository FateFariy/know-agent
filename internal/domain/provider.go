package domain

import (
	"github.com/google/wire"

	chatLogic "github.com/swiftbit/know-agent/internal/domain/chat/logic"
	documentadapter "github.com/swiftbit/know-agent/internal/domain/document/adapter"
	documentLogic "github.com/swiftbit/know-agent/internal/domain/document/logic"
)

var ProviderSet = wire.NewSet(
	documentLogic.NewLifecycleLogicImpl,
	wire.Bind(new(documentLogic.LifecycleLogic), new(*documentLogic.LifecycleLogicImpl)),
	documentLogic.NewAsyncProcessingLogic,
	wire.Bind(new(documentLogic.AsyncProcessingLogic), new(*documentLogic.AsyncProcessingLogicImpl)),
	documentLogic.NewStructureNodeLogicImpl,
	wire.Bind(new(documentLogic.StructureNodeLogic), new(*documentLogic.StructureNodeLogicImpl)),
	documentLogic.NewStrategyLogicImpl,
	wire.Bind(new(documentLogic.StrategyLogic), new(*documentLogic.StrategyLogicImpl)),
	documentadapter.NewDocumentPort,
	chatLogic.NewChatLogic,
	wire.Bind(new(chatLogic.ChatLogic), new(*chatLogic.ChatLogicImpl)),
)
