package domain

import (
	"github.com/google/wire"

	chatlogic "github.com/swiftbit/know-agent/internal/domain/chat/logic"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/conversation"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/intent"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/memory"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/memory/strategy"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/orchestrator"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/prompt"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/rag"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/rag/channel"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/recommend"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/rewrite"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/trace"
	documentadapter "github.com/swiftbit/know-agent/internal/domain/document/adapter"
	documentlogic "github.com/swiftbit/know-agent/internal/domain/document/logic"
	"github.com/swiftbit/know-agent/internal/domain/document/logic/transform"
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
	prompt.NewPromptTemplateLogicImpl,
	wire.Bind(new(chatlogic.PromptTemplateLogic), new(*prompt.TemplateLogicImpl)),
	orchestrator.NewChatPreparationOrchestratorImpl,
	wire.Bind(new(chatlogic.ChatPreparationOrchestratorLogic), new(*orchestrator.PreparationOrchestratorImpl)),
	memory.NewSessionMemoryLogicImpl,
	wire.Bind(new(chatlogic.SessionMemoryLogic), new(*memory.SessionMemoryLogicImpl)),
	intent.NewDocumentQuestionRouterImpl,
	wire.Bind(new(chatlogic.DocumentQuestionRouteLogic), new(*intent.DocumentQuestionRouterImpl)),
	chatlogic.NewChatModelImpl,
	rag.NewPromptBuilder,
	channel.NewKeywordRetrievalChannel,
	wire.Bind(new(rag.RetrievalChannel), new(*channel.KeywordRetrievalChannel)),
	channel.NewVectorRetrievalChannel,
	wire.Bind(new(rag.RetrievalChannel), new(*channel.VectorRetrievalChannel)),
	strategy.NewSummaryCompressionStrategy,
	wire.Bind(new(memory.Strategy), new(*strategy.SummaryCompressionStrategy)),
	trace.NewConversationTraceRecorder,
)

var documentProviderSet = wire.NewSet(
	documentlogic.NewAsyncProcessingLogicImpl,
	wire.Bind(new(documentlogic.AsyncProcessingLogic), new(*documentlogic.AsyncProcessingLogicImpl)),
	documentlogic.NewChunkStrategyLogicImpl,
	wire.Bind(new(documentlogic.ChunkStrategyLogic), new(*documentlogic.ChunkStrategyLogicImpl)),
	documentlogic.NewProfileLogicImpl,
	wire.Bind(new(documentlogic.ProfileLogic), new(*documentlogic.ProfileLogicImpl)),
	documentlogic.NewLifecycleLogicImpl,
	wire.Bind(new(documentlogic.LifecycleLogic), new(*documentlogic.LifecycleLogicImpl)),
	documentlogic.NewStructureNodeLogicImpl,
	wire.Bind(new(documentlogic.StructureNodeLogic), new(*documentlogic.StructureNodeLogicImpl)),
	documentlogic.NewTextPreProcessLogicImpl,
	wire.Bind(new(documentlogic.TextPreProcessLogic), new(*documentlogic.TextPreProcessLogicImpl)),
	documentadapter.NewDocumentPort,
	transform.NewAmbiguityResolver,
	transform.NewHierarchyResolver,
	transform.NewSignalExtractor,
	transform.NewTreeValidator,
)

var knowledgeProviderSet = wire.NewSet(
	knowledgelogic.NewKnowledgeRouteLogicImpl,
	wire.Bind(new(knowledgelogic.KnowledgeRouteLogic), new(*knowledgelogic.KnowledgeRouteLogicImpl)),
	knowledgelogic.NewKnowledgeLogicImpl,
	wire.Bind(new(knowledgelogic.KnowledgeLogic), new(*knowledgelogic.KnowledgeLogicImpl)),
)
