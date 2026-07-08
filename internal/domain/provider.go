package domain

import (
	"github.com/google/wire"

	chatlogic "github.com/swiftbit/know-agent/internal/domain/chat/logic"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/conversation"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/rag"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/recommend"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/rewrite"
	documentadapter "github.com/swiftbit/know-agent/internal/domain/document/adapter"
	documentlogic "github.com/swiftbit/know-agent/internal/domain/document/logic"
	knowledgelogic "github.com/swiftbit/know-agent/internal/domain/knowledge/logic"
)

var ProviderSet = wire.NewSet(
	chatProviderSet,
	documentProviderSet,
	knowledgeProviderSet,
)

var chatProviderSet = wire.NewSet(
	conversation.NewChatLogic,
	wire.Bind(new(chatlogic.ChatLogic), new(*conversation.LogicImpl)),
	rewrite.NewQueryRewriteLogicImpl,
	wire.Bind(new(chatlogic.QueryRewriteLogic), new(*rewrite.QueryRewriteLogicImpl)),
	recommend.NewRecommendationLogicImpl,
	wire.Bind(new(chatlogic.RecommendationLogic), new(*recommend.RecommendationLogicImpl)),
	rag.NewRetrievalImpl,
	wire.Bind(new(chatlogic.RagRetrieveLogic), new(*rag.RetrievalImpl)),
)

var documentProviderSet = wire.NewSet(
	documentlogic.NewLifecycleLogicImpl,
	wire.Bind(new(documentlogic.LifecycleLogic), new(*documentlogic.LifecycleLogicImpl)),
	documentlogic.NewAsyncProcessingLogicImpl,
	wire.Bind(new(documentlogic.AsyncProcessingLogic), new(*documentlogic.AsyncProcessingLogicImpl)),
	documentlogic.NewStructureNodeLogicImpl,
	wire.Bind(new(documentlogic.StructureNodeLogic), new(*documentlogic.StructureNodeLogicImpl)),
	documentlogic.NewChunkStrategyLogicImpl,
	wire.Bind(new(documentlogic.ChunkStrategyLogic), new(*documentlogic.ChunkStrategyLogicImpl)),
	documentlogic.NewTextPreProcessLogicImpl,
	wire.Bind(new(documentlogic.TextPreProcessLogic), new(*documentlogic.TextPreProcessLogicImpl)),
	documentadapter.NewDocumentPort,
)

var knowledgeProviderSet = wire.NewSet(
	knowledgelogic.NewKnowledgeRouteLogicImpl,
	wire.Bind(new(knowledgelogic.KnowledgeRouteLogic), new(*knowledgelogic.KnowledgeRouteLogicImpl)),
	knowledgelogic.NewKnowledgeLogic,
	wire.Bind(new(knowledgelogic.KnowledgeLogic), new(*knowledgelogic.KnowledgeLogicImpl)),
)
