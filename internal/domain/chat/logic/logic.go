package logic

import (
	"context"

	"github.com/swiftbit/know-agent/api/chat"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
)

// ChatLogic 聊天业务逻辑接口
type ChatLogic interface {
	// OpenConversationStream 打开会话流
	OpenConversationStream(ctx context.Context, cmd *vo.ChatCommand) <-chan string

	// ListKnowledgeDocumentOptions 获取知识文档选项列表
	ListKnowledgeDocumentOptions(ctx context.Context) ([]*chat.KnowledgeDocumentOptionResp, error)

	// StopConversation 停止会话
	StopConversation(ctx context.Context, conversationId string) (bool, string, error)

	// GetSessionDetail 获取会话详情
	GetSessionDetail(ctx context.Context, conversationId string) (*chat.ConversationSessionResp, error)

	// GetExchangeDetail 获取对话详情
	GetExchangeDetail(ctx context.Context, conversationId string, exchangeId int64) (*chat.ConversationExchangeDetailResp, error)

	// ListSessions 获取会话列表
	ListSessions(ctx context.Context, req *chat.ConversationSessionListReq) (*chat.ConversationSessionListResp, error)

	// ResetConversation 重置会话
	ResetConversation(ctx context.Context, conversationId string) (*vo.ConversationReset, error)

	// RebuildConversationSummary 重建会话摘要
	RebuildConversationSummary(ctx context.Context, conversationId string) (*chat.ConversationMemorySummaryResp, error)

	// GetRetrievalResults 获取检索结果
	GetRetrievalResults(ctx context.Context, conversationId string, exchangeId int64) ([]*vo.ChatRetrievalResult, error)

	// GetChannelExecutions 获取渠道执行结果
	GetChannelExecutions(ctx context.Context, conversationId string, exchangeId int64) ([]*vo.ChatChannelExecution, error)

	// GetStageBenchmarks 获取阶段基准
	// GetStageBenchmarks(ctx context.Context) ([]*vo.ChatStageBenchmark, error)
}

// SessionMemoryLogic 会话记忆逻辑接口
type SessionMemoryLogic interface {
	// LoadMemoryContext 加载会话记忆上下文
	LoadMemoryContext(ctx context.Context, conversationId string, trace *vo.ConversationTrace) (*vo.MemoryContext, error)

	// RefreshConversationSummaryAsync 异步刷新会话摘要
	RefreshConversationSummaryAsync(ctx context.Context, conversationId string)

	// GetConversationSummary 获取会话摘要
	GetConversationSummary(ctx context.Context, conversationId string) (*entity.ChatMemorySummary, error)

	// RebuildConversationSummary 重建会话摘要
	RebuildConversationSummary(ctx context.Context, conversationId string) (*entity.ChatMemorySummary, error)

	// DeleteConversationSummary 删除会话摘要
	DeleteConversationSummary(ctx context.Context, conversationId string) error
}

type PromptTemplateLogic interface {
	Render(templateName string, variables map[string]any) (string, error)
}

type QueryRewriteLogic interface {
	Rewrite(ctx context.Context, question, historySummary string, trace *vo.ConversationTrace) (*vo.QuestionRewriteResult, error)
}

// RecommendationLogic 推荐追问业务逻辑接口
type RecommendationLogic interface {
	// GenerateRecommendations 生成推荐追问
	GenerateRecommendations(ctx context.Context, question, answer string, recentExchanges []*entity.ChatExchange, trace *vo.ConversationTrace) []string
}

// DocumentQuestionRouteLogic 文档问题路由接口
type DocumentQuestionRouteLogic interface {
	// Route 根据文档ID和问题进行文档内路由
	Route(ctx context.Context, documentId int64, question string, rewriteResult *vo.QuestionRewriteResult) (*vo.DocumentNavigationDecision, error)
}

// ChatPreparationOrchestratorLogic 聊天准备编排器接口
type ChatPreparationOrchestratorLogic interface {
	// Prepare 准备对话执行计划
	Prepare(ctx context.Context, convCtx *vo.ConversationContext) (*vo.ConversationExecutionPlan, error)
}

// RagRetriever RAG 检索引擎接口
type RagRetriever interface {
	Retrieve(ctx context.Context, plan *vo.ConversationExecutionPlan, trace *vo.ConversationTrace) (*vo.RagRetrievalContext, error)
}
