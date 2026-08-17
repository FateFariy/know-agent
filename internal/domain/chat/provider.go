package chat

import (
	"github.com/google/wire"

	"github.com/swiftbit/know-agent/internal/domain/chat/logic"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/conversation"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/executor"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/graph"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/memory"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/memory/strategy"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/rag"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/rag/channel"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/recommend"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/rewrite"
	"github.com/swiftbit/know-agent/internal/infrastructure/observability"
)

var ProviderSet = wire.NewSet(
	logic.NewConversationLogicImpl,
	wire.Bind(new(logic.ConversationLogic), new(*logic.ConversationLogicImpl)),
	rewrite.NewQueryRewriteImpl,
	wire.Bind(new(rewrite.QueryRewriter), new(*rewrite.QueryRewriteImpl)),
	recommend.NewQuestionRecommendImpl,
	wire.Bind(new(recommend.QuestionRecommender), new(*recommend.QuestionRecommendImpl)),
	rag.NewRetrievalEngine,
	wire.Bind(new(rag.Retriever), new(*rag.RetrievalEngine)),
	memory.NewSessionMemoryManageImpl,
	wire.Bind(new(memory.SessionMemoryManager), new(*memory.SessionMemoryManageImpl)),
	graph.NewDefaultStructureGraphQuerier,
	wire.Bind(new(graph.GraphQuerier), new(*graph.DefaultStructureGraphQuerier)),
	graph.NewDefaultAnswerRender,
	wire.Bind(new(graph.AnswerRender), new(*graph.DefaultAnswerRender)),
	conversation.NewEvidenceBudgetStage,
	channel.NewKeywordRetrievalChannel,
	channel.NewVectorRetrievalChannel,
	strategy.NewSummaryCompressionStrategy,
	wire.Bind(new(strategy.Memory), new(*strategy.SummaryCompressionStrategy)),
	executor.NewClarificationExecutor,
	observability.NewConversationTraceRecorder,
	NewExecutorRegistry,
	NewRetrievalChannels,
)

func NewRetrievalChannels(ch1 *channel.VectorRetrievalChannel, ch2 *channel.KeywordRetrievalChannel) []rag.Retrieval {
	return []rag.Retrieval{ch1, ch2}
}

// NewExecutorRegistry 组合四种 executor 为执行器注册表。
func NewExecutorRegistry(
	clarification *executor.ClarificationExecutor,
) *executor.Registry {
	return executor.NewExecutorRegistry(clarification)
}
