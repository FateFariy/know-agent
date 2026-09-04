package main

import (
	"github.com/swiftbit/know-agent/internal/config"
	"github.com/swiftbit/know-agent/internal/domain/chat/adapter/model"
	chatlogic "github.com/swiftbit/know-agent/internal/domain/chat/logic"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/conversation"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/conversation/middleware"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/conversation/tool"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/evaluate"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/memory"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/memory/strategy"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/recommend"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/retrieval"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/retrieval/channel"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/retrieval/fuse"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/rewrite"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/route"
	"github.com/swiftbit/know-agent/internal/domain/document/adapter"
	doclogic "github.com/swiftbit/know-agent/internal/domain/document/logic"
	"github.com/swiftbit/know-agent/internal/domain/document/logic/process"
	"github.com/swiftbit/know-agent/internal/domain/document/logic/process/analysis"
	"github.com/swiftbit/know-agent/internal/domain/document/logic/process/chunk"
	chunkllm "github.com/swiftbit/know-agent/internal/domain/document/logic/process/chunk/llm"
	"github.com/swiftbit/know-agent/internal/domain/document/logic/process/chunk/recursive"
	"github.com/swiftbit/know-agent/internal/domain/document/logic/process/chunk/semantic"
	"github.com/swiftbit/know-agent/internal/domain/document/logic/process/index"
	"github.com/swiftbit/know-agent/internal/domain/document/model/enum"
	knowlogic "github.com/swiftbit/know-agent/internal/domain/knowledge/logic"
	knowroute "github.com/swiftbit/know-agent/internal/domain/knowledge/logic/route"
	"github.com/swiftbit/know-agent/internal/domain/knowledge/logic/route/rank"
	"github.com/swiftbit/know-agent/internal/infrastructure/observability"
	"github.com/swiftbit/know-agent/internal/infrastructure/persistence"
	"github.com/swiftbit/know-agent/internal/infrastructure/port/agent"
	"github.com/swiftbit/know-agent/internal/infrastructure/port/agent/check"
	"github.com/swiftbit/know-agent/internal/infrastructure/port/cache"
	portconfig "github.com/swiftbit/know-agent/internal/infrastructure/port/config"
	"github.com/swiftbit/know-agent/internal/infrastructure/port/emb"
	"github.com/swiftbit/know-agent/internal/infrastructure/port/gateway"
	"github.com/swiftbit/know-agent/internal/infrastructure/port/keyword"
	"github.com/swiftbit/know-agent/internal/infrastructure/port/llm"
	"github.com/swiftbit/know-agent/internal/infrastructure/port/lock"
	"github.com/swiftbit/know-agent/internal/infrastructure/port/mq"
	"github.com/swiftbit/know-agent/internal/infrastructure/port/prompt"
	"github.com/swiftbit/know-agent/internal/infrastructure/port/rerank"
	"github.com/swiftbit/know-agent/internal/infrastructure/port/storage"
	"github.com/swiftbit/know-agent/internal/infrastructure/port/tokenize"
	"github.com/swiftbit/know-agent/internal/infrastructure/port/vector"
	"github.com/swiftbit/know-agent/internal/server"
	"github.com/swiftbit/know-agent/internal/svc"
	"github.com/swiftbit/know-agent/internal/trigger/consumer"
	"github.com/swiftbit/know-agent/internal/trigger/handler"
)

func bootstrap(c *config.Config) *server.Server {
	svcCtx := svc.NewServiceContext(c)

	embedder := emb.NewOllamaEmbedder(svcCtx)
	checkPointStore := check.NewMemoryCheckPointStore()
	localConfig := portconfig.NewLocalConfig(svcCtx)
	milvusKeyword := keyword.NewMilvusKeyword(svcCtx)
	milvusVector := vector.NewMilvusVector(svcCtx, embedder)
	rocketMQMessageProducer := mq.NewRocketMQMessageProducer(svcCtx)
	redisMutexLock := lock.NewRedisMutexLock(svcCtx)
	chatModel := llm.NewChatModelImpl(svcCtx)
	ollamaModel := llm.NewOllamaModel(svcCtx)
	renderer := prompt.NewRendererImpl()
	dashScope := rerank.NewDashScope(svcCtx)
	minioStorage := storage.NewMinioStorage(svcCtx)
	elasticStorage := storage.NewElasticStorage(svcCtx)
	gseTokenizer := tokenize.NewGseTokenizer(svcCtx)
	semanticCacheStore := cache.NewSemanticCache(svcCtx, embedder)
	semanticCacheWriteConsumer := consumer.NewSemanticCacheWriteConsumer(svcCtx, semanticCacheStore)

	documentRepo := persistence.NewDocumentRepository(svcCtx, minioStorage, milvusVector)
	tableRepo := persistence.NewTableRepository(svcCtx)
	chatRepo := persistence.NewChatRepository(svcCtx)
	knowledgeRepo := persistence.NewKnowledgeRepository(svcCtx)
	documentForKnowledge := gateway.NewDocumentAdapterForKnowledge(documentRepo)
	documentForChat := gateway.NewDocumentAdapterForChat(documentRepo)
	observability.RegisterConversationTraceRecorder(chatRepo)
	observability.RegisterRagEvalMetricsHandler(chatRepo)
	observability.RegisterModelUsageHandler(svcCtx.Config.ChatModel)

	retrievalScopeLogicImpl := knowlogic.NewKnowledgeBaseRetrievalScopeLogicImpl(knowledgeRepo, documentForKnowledge, localConfig)
	opts := []rank.Option{rank.WithLexicalIndex(elasticStorage), rank.WithEmbedding(embedder)}
	rankers := []knowroute.Ranker{
		rank.NewDocumentRanker(knowledgeRepo, documentForKnowledge, opts...),
		rank.NewScopeRanker(knowledgeRepo, documentForKnowledge, opts...),
		rank.NewTopicRanker(knowledgeRepo, documentForKnowledge, opts...),
	}
	knowledgeRouter := knowroute.NewKnowledgeRouteImpl(knowledgeRepo, documentForKnowledge, rankers, knowroute.WithEmbedding(embedder))
	knowledgeLogicImpl := knowlogic.NewKnowledgeLogicImpl(knowledgeRepo, documentForKnowledge)
	knowledgeBaseLogicImpl := knowlogic.NewKnowledgeBaseLogicImpl(knowledgeRepo, documentForKnowledge)
	knowledgeAdapter := gateway.NewKnowledgeAdapter(retrievalScopeLogicImpl, knowledgeRepo, knowledgeRouter, localConfig)

	recommender := recommend.NewQuestionRecommendImpl(svcCtx, renderer, ollamaModel)
	channels := []retrieval.Retrieval{
		channel.NewVectorRetrievalChannel(svcCtx, milvusVector),
		channel.NewKeywordRetrievalChannel(milvusKeyword),
	}

	rrfFusion := fuse.NewRRFFusion()
	retrievalEngine := retrieval.NewRetrievalEngine(svcCtx, chatRepo, dashScope, channels, documentForChat, rrfFusion)

	compressionStrategy := strategy.NewSummaryCompressionStrategy(svcCtx, chatRepo, chatModel, renderer)
	memoryManageImpl := memory.NewSessionMemoryManageImpl(compressionStrategy)
	rewriteImpl := rewrite.NewQueryRewriteImpl(svcCtx, ollamaModel, renderer)
	chainRuntime := conversation.NewRuntimeRegistry()
	documentRouter := route.NewDocumentRouter(documentForChat, nil)
	contextInjectMiddleware := middleware.NewContextInjectMiddleware(svcCtx)
	structureTool := tool.NewRouteDocumentStructureTool(documentRouter)
	knowledgeBaseSearchTool := tool.NewKnowledgeBaseSearchTool(svcCtx, retrievalEngine)
	agentRunner := agent.NewEinoAgentRunner(svcCtx,
		agent.WithMiddleware(contextInjectMiddleware),
		agent.WithTools(structureTool),
		agent.WithTools(knowledgeBaseSearchTool),
	)

	stages := []conversation.Stage{
		conversation.NewStartStage(chatRepo, chainRuntime, redisMutexLock),
		conversation.NewMemoryLoadStage(svcCtx, memoryManageImpl, chatRepo),
		conversation.NewQueryRewriteStage(svcCtx, rewriteImpl),
		conversation.NewRouteStage(chatRepo, knowledgeAdapter, documentForChat),
		conversation.NewSemanticCacheStage(svcCtx, semanticCacheStore, ollamaModel, renderer),
		conversation.NewAgentStage(agentRunner),
		conversation.NewCacheWriteStage(svcCtx, semanticCacheStore, rocketMQMessageProducer),
		conversation.NewAnswerEvaluateStage(NewAnswerEvaluators(ollamaModel, renderer, embedder)),
		conversation.NewRecommendStage(svcCtx, chatRepo, memoryManageImpl, recommender),
		conversation.NewEndStage(chatRepo),
	}
	chain := conversation.NewChain(chatRepo, redisMutexLock, memoryManageImpl, chainRuntime, stages)
	conversationLogicImpl := chatlogic.NewConversationLogicImpl(chatRepo, knowledgeAdapter, memoryManageImpl, redisMutexLock, checkPointStore, chain)
	strategyRegistry := NewChunkStrategyRegistry(svcCtx, chatModel, renderer)

	generateImpl := process.NewProfileGenerateImpl(documentRepo)
	analysisChain := analysis.NewAnalysisChain(svcCtx, documentRepo, tableRepo, minioStorage, generateImpl, knowledgeAdapter)
	indexChain := index.NewBuildIndexChain(documentRepo, minioStorage, milvusVector, milvusKeyword, strategyRegistry, knowledgeAdapter, gseTokenizer)
	asyncProcessImpl := process.NewAsyncProcessImpl(documentRepo, analysisChain, indexChain)
	lifecycleLogicImpl := doclogic.NewLifecycleLogicImpl(svcCtx, rocketMQMessageProducer, minioStorage, documentRepo, knowledgeAdapter)
	profileLogicImpl := doclogic.NewProfileLogicImpl(documentRepo, minioStorage, generateImpl)
	parseConsumer := consumer.NewParseDocumentConsumer(svcCtx, asyncProcessImpl)
	buildIndexConsumer := consumer.NewBuildIndexConsumer(svcCtx, asyncProcessImpl)

	chatService := handler.NewChatService(conversationLogicImpl)
	documentService := handler.NewDocumentService(lifecycleLogicImpl, profileLogicImpl)
	knowledgeService := handler.NewKnowledgeService(knowledgeLogicImpl, knowledgeBaseLogicImpl)
	httpServer := server.NewHTTPServer(svcCtx, documentService, chatService, knowledgeService)

	return server.NewServer(httpServer, parseConsumer, buildIndexConsumer, rocketMQMessageProducer, semanticCacheWriteConsumer)
}

func NewChunkStrategyRegistry(svcCtx *svc.ServiceContext, chatModel model.ChatModel, template adapter.PromptRenderer) *chunk.Registry {
	chunkers := []chunk.Chunker{
		// 递归分块
		recursive.NewChunker(
			recursive.WithMaxChars(svcCtx.Config.Chunk.RecursiveMaxChars),
			recursive.WithOverlapChars(svcCtx.Config.Chunk.RecursiveOverlapChars),
		),
		// 语义分块
		semantic.NewChunker(
			semantic.WithMinChars(svcCtx.Config.Chunk.SemanticMinChars),
			semantic.WithMaxChars(svcCtx.Config.Chunk.SemanticMaxChars),
			semantic.WithSimilarityThreshold(svcCtx.Config.Chunk.SemanticSimilarityThreshold),
		),
		// 大模型切块
		chunkllm.NewChunker(chatModel, template,
			chunkllm.WithLlmSplitPrompt(enum.DocumentLlmSplit),
		),
	}
	return chunk.NewChunkStrategyRegistry(chunkers)
}

func NewAnswerEvaluators(llm model.ChatModel, renderer adapter.PromptRenderer, emb *emb.OllamaEmbedder) []evaluate.Evaluator {
	return []evaluate.Evaluator{
		evaluate.NewAnswerFaithfulnessEvaluator(llm, renderer),
		evaluate.NewAnswerRelevancyEvaluator(llm, renderer, emb),
		evaluate.NewContextPrecisionEvaluator(llm, renderer),
	}
}
