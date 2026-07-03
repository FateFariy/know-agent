package domain

import (
	"github.com/google/wire"

	chatLogic "github.com/swiftbit/know-agent/internal/domain/chat/logic"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/chat"
	documentadapter "github.com/swiftbit/know-agent/internal/domain/document/adapter"
	documentLogic "github.com/swiftbit/know-agent/internal/domain/document/logic"
	knowledgeLogic "github.com/swiftbit/know-agent/internal/domain/knowledge/logic"
)

var ProviderSet = wire.NewSet(
	documentLogic.NewLifecycleLogicImpl,
	wire.Bind(new(documentLogic.LifecycleLogic), new(*documentLogic.LifecycleLogicImpl)),
	documentLogic.NewAsyncProcessingLogicImpl,
	wire.Bind(new(documentLogic.AsyncProcessingLogic), new(*documentLogic.AsyncProcessingLogicImpl)),
	documentLogic.NewStructureNodeLogicImpl,
	wire.Bind(new(documentLogic.StructureNodeLogic), new(*documentLogic.StructureNodeLogicImpl)),
	documentLogic.NewChunkStrategyLogicImpl,
	wire.Bind(new(documentLogic.ChunkStrategyLogic), new(*documentLogic.ChunkStrategyLogicImpl)),
	documentLogic.NewTextPreProcessLogicImpl,
	wire.Bind(new(documentLogic.TextPreProcessLogic), new(*documentLogic.TextPreProcessLogicImpl)),
	documentadapter.NewDocumentPort,
	chat.NewChatLogic,
	wire.Bind(new(chatLogic.ChatLogic), new(*chat.LogicImpl)),
	knowledgeLogic.NewKnowledgeRouteLogicImpl,
	wire.Bind(new(knowledgeLogic.KnowledgeRouteLogic), new(*knowledgeLogic.KnowledgeRouteLogicImpl)),
)
