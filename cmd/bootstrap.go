package main

import (
	"github.com/swiftbit/know-agent/internal/config"
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
	"github.com/swiftbit/know-agent/internal/domain/document/logic/process"
	"github.com/swiftbit/know-agent/internal/domain/document/logic/process/parse"
	"github.com/swiftbit/know-agent/internal/infrastructure/persistence"
	"github.com/swiftbit/know-agent/internal/infrastructure/port/check"
	portconfig "github.com/swiftbit/know-agent/internal/infrastructure/port/config"
	"github.com/swiftbit/know-agent/internal/infrastructure/port/gateway"
	"github.com/swiftbit/know-agent/internal/infrastructure/port/keyword"
	"github.com/swiftbit/know-agent/internal/infrastructure/port/llm"
	"github.com/swiftbit/know-agent/internal/infrastructure/port/lock"
	"github.com/swiftbit/know-agent/internal/infrastructure/port/mq"
	"github.com/swiftbit/know-agent/internal/infrastructure/port/parser"
	"github.com/swiftbit/know-agent/internal/infrastructure/port/parser/markdown"
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
	conversationLogicImpl := chatlogic.NewConversationLogicImpl()

	chatModelImpl := llm.NewChatModelImpl(serviceContext)
	parserRegistry := NewParserRegistry()
	rendererImpl := prompt.NewRendererImpl()
	dashScope := rerank.NewDashScope(serviceContext)
	minioStorage := storage.NewMinioStorage(serviceContext)
	gseTokenizer := tokenize.NewGseTokenizer(serviceContext)

	documentPort := adapter.NewDocumentPort(minioStorage, rocketMQMessageProducer, milvusVector, milvusKeyword)
	documentRepositoryImpl := persistence.NewDocumentRepository(serviceContext, minioStorage, milvusVector)
	chatRepositoryImpl := persistence.NewChatRepository(serviceContext)
	knowledgeRepositoryImpl := persistence.NewKnowledgeRepository(serviceContext)

	knowledgeAdapter := gateway.NewKnowledgeAdapter()
	adapterForKnowledge := gateway.NewDocumentAdapterForKnowledge(documentRepositoryImpl)
	adapterForChat := gateway.NewDocumentAdapterForChat(documentRepositoryImpl)
	intentRecognizer := intent.NewCompositeIntentRecognizer(chatModelImpl, rendererImpl)

	questionRecommendImpl := recommend.NewQuestionRecommendImpl(serviceContext, rendererImpl, chatModelImpl)
	channels := []rag.Retrieval{
		channel.NewVectorRetrievalChannel(serviceContext, milvusVector),
		channel.NewKeywordRetrievalChannel(serviceContext, milvusKeyword),
	}

	rrfFusion := fuse.NewRRFFusion()
	retrievalImpl := rag.NewRetrievalEngine(serviceContext, chatRepositoryImpl, dashScope, channels, adapterForChat, rrfFusion)

	compressionStrategy := strategy.NewSummaryCompressionStrategy(serviceContext, chatRepositoryImpl, chatModelImpl, rendererImpl)
	memoryManageImpl := memory.NewSessionMemoryManageImpl(compressionStrategy)
	chain := conversation.NewChain(serviceContext, chatRepositoryImpl, redisMutexLock, memoryManageImpl)
	graphQuerier := graph.NewDefaultStructureGraphQuerier(serviceContext)
	rewriteImpl := rewrite.NewQueryRewriteImpl(serviceContext, chatModelImpl, rendererImpl)
	questionRouter := route.NewDocumentQuestionRouter(graphQuerier, nil)

	asyncProcessImpl := process.NewAsyncProcessImpl(documentRepositoryImpl, documentPort)
	parseConsumer := consumer.NewParseDocumentConsumer(serviceContext, asyncProcessImpl)
	buildIndexConsumer := consumer.NewBuildIndexConsumer(serviceContext, asyncProcessImpl)

	chatService := handler.NewChatService(conversationLogicImpl)
	documentService := handler.NewDocumentService()
	knowledgeService := handler.NewKnowledgeService()
	httpServer := server.NewHTTPServer(serviceContext, documentService, chatService, knowledgeService)

	return server.NewServer(httpServer, parseConsumer, buildIndexConsumer, rocketMQMessageProducer)
}

func NewParserRegistry() *parse.Registry {
	fallbackParser := &parser.TextParser{}
	parsers := []parse.Parser{
		&parser.HTMLParser{},
		&parser.TextParser{},
		&parser.PDFParser{},
		&markdown.GoldmarkParser{},
	}
	return parse.NewRegistry(fallbackParser, parsers...)
}
