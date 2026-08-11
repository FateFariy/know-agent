package chat

import (
	"github.com/google/wire"

	chatlogic "github.com/swiftbit/know-agent/internal/domain/chat/logic"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/executor"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/graph"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/intent"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/memory"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/memory/strategy"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/preparation"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/rag"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/rag/channel"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/recommend"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/rewrite"
	"github.com/swiftbit/know-agent/internal/infrastructure/observability"
)

var ProviderSet = wire.NewSet(
	chatlogic.NewConversationLogicImpl,
	wire.Bind(new(chatlogic.ConversationLogic), new(*chatlogic.ConversationLogicImpl)),
	rewrite.NewQueryRewriteImpl,
	wire.Bind(new(rewrite.QueryRewriter), new(*rewrite.QueryRewriteImpl)),
	recommend.NewQuestionRecommendImpl,
	wire.Bind(new(recommend.QuestionRecommender), new(*recommend.QuestionRecommendImpl)),
	rag.NewRetrievalImpl,
	wire.Bind(new(rag.Retriever), new(*rag.RetrievalImpl)),
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
	executor.NewRagChatExecutor,
	executor.NewGraphOnlyExecutor,
	executor.NewGraphThenEvidenceExecutor,
	executor.NewClarificationExecutor,
	observability.NewConversationTraceRecorder,
	NewExecutorRegistry,
	NewRetrievalChannels,
	wire.Bind(new(executor.RagPromptAssembler), new(*rag.PromptAssembler)),
)

func NewRetrievalChannels(ch1 *channel.VectorRetrievalChannel, ch2 *channel.KeywordRetrievalChannel) []channel.Retrieval {
	return []channel.Retrieval{ch1, ch2}
}

// NewExecutorRegistry 组合四种 executor 为执行器注册表。
func NewExecutorRegistry(
	rag *executor.RagChatExecutor,
	graphOnly *executor.GraphOnlyExecutor,
	graphThen *executor.GraphThenEvidenceExecutor,
	clarification *executor.ClarificationExecutor,
) *executor.Registry {
	return executor.NewExecutorRegistry(rag, graphOnly, graphThen, clarification)
}
