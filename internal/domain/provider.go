package domain

import (
	"github.com/google/wire"

	chatlogic "github.com/swiftbit/know-agent/internal/domain/chat/logic"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/conversation"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/conversation/executor"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/graph"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/intent"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/memory"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/memory/strategy"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/orchestrator"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/prompt"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/rag"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/rag/channel"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/recommend"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/rewrite"
	documentadapter "github.com/swiftbit/know-agent/internal/domain/document/adapter"
	documentlogic "github.com/swiftbit/know-agent/internal/domain/document/logic"
	"github.com/swiftbit/know-agent/internal/domain/document/logic/process"
	"github.com/swiftbit/know-agent/internal/domain/document/logic/process/transform"
	knowledgelogic "github.com/swiftbit/know-agent/internal/domain/knowledge/logic"
	"github.com/swiftbit/know-agent/internal/domain/knowledge/logic/route"
	"github.com/swiftbit/know-agent/internal/infrastructure/observability"
)

var ProviderSet = wire.NewSet(
	chatProviderSet,
	documentProviderSet,
	knowledgeProviderSet,
)

var chatProviderSet = wire.NewSet(
	conversation.NewChatLogic,
	wire.Bind(new(chatlogic.ConversationLogic), new(*conversation.LogicImpl)),
	rewrite.NewQueryRewriter,
	wire.Bind(new(chatlogic.QueryRewriter), new(*rewrite.QueryRewriter)),
	recommend.NewQuestionRecommender,
	wire.Bind(new(chatlogic.QuestionRecommender), new(*recommend.QuestionRecommender)),
	rag.NewRetrievalImpl,
	wire.Bind(new(chatlogic.RagRetriever), new(*rag.RetrievalImpl)),
	prompt.NewPromptTemplateLogicImpl,
	wire.Bind(new(chatlogic.PromptRenderer), new(*prompt.TemplateRenderer)),
	orchestrator.NewChatPreparationOrchestratorImpl,
	wire.Bind(new(chatlogic.ChatPreparationOrchestratorLogic), new(*orchestrator.PreparationOrchestratorImpl)),
	memory.NewSessionMemoryLogicImpl,
	wire.Bind(new(chatlogic.MemoryManager), new(*memory.SessionMemoryLogicImpl)),
	intent.NewDocumentQuestionRouterImpl,
	wire.Bind(new(chatlogic.DocumentRouter), new(*intent.DocumentQuestionRouterImpl)),
	graph.NewDefaultStructureGraphQuerier,
	wire.Bind(new(chatlogic.GraphQuerier), new(*graph.DefaultStructureGraphQuerier)),
	intent.NewDefaultNavigationIndexService,
	wire.Bind(new(intent.NavigationIndexService), new(*intent.DefaultNavigationIndexService)),
	graph.NewDefaultAnswerRender,
	wire.Bind(new(graph.AnswerRender), new(*graph.DefaultAnswerRender)),
	rag.NewPromptAssembler,
	channel.NewKeywordRetrievalChannel,
	channel.NewVectorRetrievalChannel,
	strategy.NewSummaryCompressionStrategy,
	wire.Bind(new(memory.Strategy), new(*strategy.SummaryCompressionStrategy)),
	executor.NewRagChatExecutor,
	executor.NewGraphOnlyExecutor,
	executor.NewGraphThenEvidenceExecutor,
	executor.NewClarificationExecutor,
	observability.NewConversationTraceRecorder,
	NewExecutorRegistry,
	NewRetrievalChannels,
	wire.Bind(new(conversation.RagPromptAssembler), new(*rag.PromptAssembler)),
)

var documentProviderSet = wire.NewSet(
	process.NewAsyncProcessImpl,
	wire.Bind(new(process.AsyncProcessor), new(*process.AsyncProcessImpl)),
	process.NewChunkCoordinateImpl,
	wire.Bind(new(process.ChunkCoordinator), new(*process.ChunkCoordinateImpl)),
	documentlogic.NewProfileLogicImpl,
	wire.Bind(new(documentlogic.ProfileLogic), new(*documentlogic.ProfileLogicImpl)),
	documentlogic.NewLifecycleLogicImpl,
	wire.Bind(new(documentlogic.LifecycleLogic), new(*documentlogic.LifecycleLogicImpl)),
	process.NewStructureNodeManager,
	wire.Bind(new(process.StructureNodeManager), new(*process.StructureNodeManageImpl)),
	process.NewTextPreprocessImpl,
	wire.Bind(new(process.TextPreprocessor), new(*process.TextPreprocessImpl)),
	process.NewProfileGenerateImpl,
	wire.Bind(new(process.ProfileGenerator), new(*process.ProfileGenerateImpl)),
	documentadapter.NewDocumentPort,
	transform.NewAmbiguityResolver,
	transform.NewHierarchyResolver,
	transform.NewSignalExtractor,
	transform.NewTreeValidator,
)

var knowledgeProviderSet = wire.NewSet(
	route.NewKnowledgeRouteImpl,
	wire.Bind(new(route.KnowledgeRouter), new(*route.KnowledgeRouteImpl)),
	knowledgelogic.NewKnowledgeLogicImpl,
	wire.Bind(new(knowledgelogic.KnowledgeLogic), new(*knowledgelogic.KnowledgeLogicImpl)),
	ProvideKnowledgeOptions,
)

// NewExecutorRegistry 组合四种 executor 为执行器注册表。
func NewExecutorRegistry(
	rag *executor.RagChatExecutor,
	graphOnly *executor.GraphOnlyExecutor,
	graphThen *executor.GraphThenEvidenceExecutor,
	clarification *executor.ClarificationExecutor,
) *conversation.ExecutorRegistry {
	return conversation.NewExecutorRegistry(rag, graphOnly, graphThen, clarification)
}

// ProvideKnowledgeOptions 提供知识路由的可选项（目前为空），
// 供 NewKnowledgeRouteImpl 消费。
func ProvideKnowledgeOptions() []route.Option {
	return nil
}

func NewRetrievalChannels(ch1 *channel.VectorRetrievalChannel, ch2 *channel.KeywordRetrievalChannel) []channel.Retrieval {
	return []channel.Retrieval{ch1, ch2}
}
