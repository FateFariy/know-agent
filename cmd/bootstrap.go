package main

import (
	"github.com/swiftbit/know-agent/internal/config"
	"github.com/swiftbit/know-agent/internal/domain/chat/adapter/model"
	chatlogic "github.com/swiftbit/know-agent/internal/domain/chat/logic"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/conversation"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/graph"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/intent"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/memory"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/memory/strategy"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/rag"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/rag/channel"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/rag/fuse"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/recommend"
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
	"github.com/swiftbit/know-agent/internal/infrastructure/persistence"
	"github.com/swiftbit/know-agent/internal/infrastructure/port/check"
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
	serviceContext := svc.NewServiceContext(c)

	checkPointStore := check.NewMemoryCheckPointStore()
	localConfig := portconfig.NewLocalConfig(serviceContext)
	milvusKeyword := keyword.NewMilvusKeyword(serviceContext)
	milvusVector := vector.NewMilvusVector(serviceContext)
	rocketMQMessageProducer := mq.NewRocketMQMessageProducer(serviceContext)
	redisMutexLock := lock.NewRedisMutexLock(serviceContext)
	chatModel := llm.NewChatModelImpl(serviceContext)
	renderer := prompt.NewRendererImpl()
	dashScope := rerank.NewDashScope(serviceContext)
	minioStorage := storage.NewMinioStorage(serviceContext)
	gseTokenizer := tokenize.NewGseTokenizer(serviceContext)
	embedder := emb.NewEmbedder(serviceContext)

	documentPort := adapter.NewDocumentPort(minioStorage, rocketMQMessageProducer, milvusVector, milvusKeyword)
	documentRepo := persistence.NewDocumentRepository(serviceContext, minioStorage, milvusVector)
	tableRepo := persistence.NewTableRepository(serviceContext)
	chatRepo := persistence.NewChatRepository(serviceContext)
	knowledgeRepo := persistence.NewKnowledgeRepository(serviceContext)
	adapterForKnowledge := gateway.NewDocumentAdapterForKnowledge(documentRepo)
	adapterForChat := gateway.NewDocumentAdapterForChat(documentRepo)

	retrievalScopeLogicImpl := knowlogic.NewKnowledgeBaseRetrievalScopeLogicImpl(knowledgeRepo, adapterForKnowledge, localConfig)
	knowledgeRouter := knowroute.NewKnowledgeRouteImpl(knowledgeRepo, adapterForKnowledge, knowroute.WithEmbedding(embedder))
	knowledgeLogicImpl := knowlogic.NewKnowledgeLogicImpl(knowledgeRepo, adapterForKnowledge)
	knowledgeBaseLogicImpl := knowlogic.NewKnowledgeBaseLogicImpl(knowledgeRepo, adapterForKnowledge)
	knowledgeAdapter := gateway.NewKnowledgeAdapter(retrievalScopeLogicImpl, knowledgeRepo, knowledgeRouter, localConfig)

	intentRecognizer := intent.NewCompositeIntentRecognizer(chatModel, renderer)
	recommender := recommend.NewQuestionRecommendImpl(serviceContext, renderer, chatModel)
	channels := []rag.Retrieval{
		channel.NewVectorRetrievalChannel(serviceContext, milvusVector),
		channel.NewKeywordRetrievalChannel(serviceContext, milvusKeyword),
	}

	rrfFusion := fuse.NewRRFFusion()
	retrievalEngine := rag.NewRetrievalEngine(serviceContext, chatRepo, dashScope, channels, adapterForChat, rrfFusion)

	compressionStrategy := strategy.NewSummaryCompressionStrategy(serviceContext, chatRepo, chatModel, renderer)
	memoryManageImpl := memory.NewSessionMemoryManageImpl(compressionStrategy)
	graphQuerier := graph.NewDefaultStructureGraphQuerier(serviceContext)
	rewriteImpl := rewrite.NewQueryRewriteImpl(serviceContext, chatModel, renderer)
	documentRouter := route.NewDocumentRouter(graphQuerier, nil)
	chain := conversation.NewChain(serviceContext, chatRepo, redisMutexLock, memoryManageImpl, intentRecognizer,
		rewriteImpl, knowledgeAdapter, documentRouter, adapterForChat, retrievalEngine, renderer, chatModel, recommender)
	conversationLogicImpl := chatlogic.NewConversationLogicImpl(chatRepo, knowledgeAdapter, memoryManageImpl, redisMutexLock, checkPointStore, chain)
	strategyRegistry := NewChunkStrategyRegistry(serviceContext, chatModel, renderer)

	generateImpl := process.NewProfileGenerateImpl(documentRepo)
	analysisChain := analysis.NewAnalysisChain(serviceContext, documentRepo, tableRepo, minioStorage, generateImpl, knowledgeAdapter)
	indexChain := index.NewBuildIndexChain(documentRepo, documentPort, strategyRegistry, knowledgeAdapter, gseTokenizer)
	asyncProcessImpl := process.NewAsyncProcessImpl(documentRepo, analysisChain, indexChain)
	lifecycleLogicImpl := doclogic.NewLifecycleLogicImpl(serviceContext, documentPort, minioStorage, documentRepo, knowledgeAdapter)
	profileLogicImpl := doclogic.NewProfileLogicImpl(documentRepo, documentPort, generateImpl)
	parseConsumer := consumer.NewParseDocumentConsumer(serviceContext, asyncProcessImpl)
	buildIndexConsumer := consumer.NewBuildIndexConsumer(serviceContext, asyncProcessImpl)

	chatService := handler.NewChatService(conversationLogicImpl)
	documentService := handler.NewDocumentService(lifecycleLogicImpl, profileLogicImpl)
	knowledgeService := handler.NewKnowledgeService(knowledgeLogicImpl)
	httpServer := server.NewHTTPServer(serviceContext, documentService, chatService, knowledgeService)

	return server.NewServer(httpServer, parseConsumer, buildIndexConsumer, rocketMQMessageProducer)
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
