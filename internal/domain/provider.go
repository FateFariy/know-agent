package domain

import (
	"github.com/google/wire"

	chatlogic "github.com/swiftbit/know-agent/internal/domain/chat/logic"
	executor2 "github.com/swiftbit/know-agent/internal/domain/chat/logic/executor"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/graph"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/intent"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/memory"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/memory/strategy"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/preparation"
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
	chatlogic.NewConversationLogicImpl,
	wire.Bind(new(chatlogic.ConversationLogic), new(*chatlogic.LogicImpl)),
	rewrite.NewQueryRewriteImpl,
	wire.Bind(new(rewrite.QueryRewriter), new(*rewrite.QueryRewriteImpl)),
	recommend.NewQuestionRecommendImpl,
	wire.Bind(new(recommend.QuestionRecommender), new(*recommend.QuestionRecommendImpl)),
	rag.NewRetrievalImpl,
	wire.Bind(new(rag.Retriever), new(*rag.RetrievalImpl)),
	prompt.NewRendererImpl,
	wire.Bind(new(prompt.Renderer), new(*prompt.RendererImpl)),
	preparation.NewChatPreparationOrchestratorImpl,
	wire.Bind(new(preparation.ConversationPreOrchestrator), new(*preparation.ConversationPreOrchestratorImpl)),
	memory.NewSessionMemoryManageImpl,
	wire.Bind(new(memory.SessionMemoryManager), new(*memory.SessionMemoryManageImpl)),
	intent.NewDocumentQuestionRouteImpl,
	wire.Bind(new(intent.DocumentRouter), new(*intent.DocumentQuestionRouteImpl)),
	graph.NewDefaultStructureGraphQuerier,
	wire.Bind(new(graph.GraphQuerier), new(*graph.DefaultStructureGraphQuerier)),
	intent.NewDefaultNavigationIndexService,
	wire.Bind(new(intent.NavigationIndexService), new(*intent.DefaultNavigationIndexService)),
	graph.NewDefaultAnswerRender,
	wire.Bind(new(graph.AnswerRender), new(*graph.DefaultAnswerRender)),
	rag.NewPromptAssembler,
	channel.NewKeywordRetrievalChannel,
	channel.NewVectorRetrievalChannel,
	strategy.NewSummaryCompressionStrategy,
	wire.Bind(new(strategy.Memory), new(*strategy.SummaryCompressionStrategy)),
	executor2.NewRagChatExecutor,
	executor2.NewGraphOnlyExecutor,
	executor2.NewGraphThenEvidenceExecutor,
	executor2.NewClarificationExecutor,
	observability.NewConversationTraceRecorder,
	NewExecutorRegistry,
	NewRetrievalChannels,
	wire.Bind(new(executor2.RagPromptAssembler), new(*rag.PromptAssembler)),
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
	rag *executor2.RagChatExecutor,
	graphOnly *executor2.GraphOnlyExecutor,
	graphThen *executor2.GraphThenEvidenceExecutor,
	clarification *executor2.ClarificationExecutor,
) *executor2.Registry {
	return executor2.NewExecutorRegistry(rag, graphOnly, graphThen, clarification)
}

// ProvideKnowledgeOptions 提供知识路由的可选项（目前为空），
// 供 NewKnowledgeRouteImpl 消费。
func ProvideKnowledgeOptions() []route.Option {
	return nil
}

func NewRetrievalChannels(ch1 *channel.VectorRetrievalChannel, ch2 *channel.KeywordRetrievalChannel) []channel.Retrieval {
	return []channel.Retrieval{ch1, ch2}
}
