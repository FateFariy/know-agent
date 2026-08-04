package logic

import (
	"context"

	"github.com/swiftbit/know-agent/internal/domain/chat/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
)

// ConversationLogic 聊天业务逻辑接口
type ConversationLogic interface {
	// OpenConversationStream 打开会话流
	OpenConversationStream(ctx context.Context, cmd *vo.ChatCommand) <-chan string

	// StopConversation 停止会话
	StopConversation(ctx context.Context, conversationId string) (bool, string, error)

	// GetSessionDetail 获取会话详情
	GetSessionDetail(ctx context.Context, conversationId string) (*vo.ConversationArchiveRecord, error)

	// GetExchangeDetail 获取对话详情
	GetExchangeDetail(ctx context.Context, conversationId string, exchangeId int64) (*entity.ChatExchange, []*entity.ChatExchangeTraceStage, error)

	// ListSessions 获取会话列表
	ListSessions(ctx context.Context, pageNo, pageSize, chatMode, latestTurnStatus int, keyword string) ([]*vo.ConversationArchiveRecord, int64, error)

	// ResetConversation 重置会话
	ResetConversation(ctx context.Context, conversationId string) (*vo.ConversationReset, error)

	// RebuildConversationSummary 重建会话摘要
	RebuildConversationSummary(ctx context.Context, conversationId string) (*entity.ChatMemorySummary, error)

	// GetRetrievalResults 获取检索结果
	GetRetrievalResults(ctx context.Context, conversationId string, exchangeId int64) ([]*vo.ChatRetrievalResult, error)

	// GetChannelExecutions 获取渠道执行结果
	GetChannelExecutions(ctx context.Context, conversationId string, exchangeId int64) ([]*vo.ChatChannelExecution, error)
}

// MemoryManager 会话记忆逻辑接口
type MemoryManager interface {
	// LoadMemoryContext 加载会话记忆上下文
	LoadMemoryContext(ctx context.Context, conversationId string, trace *vo.ConversationTrace) (*vo.MemoryContext, error)

	// RefreshConversationSummaryAsync 异步刷新会话摘要
	RefreshConversationSummaryAsync(conversationId string)

	// GetConversationSummary 获取会话摘要
	GetConversationSummary(ctx context.Context, conversationId string) (*entity.ChatMemorySummary, error)

	// RebuildConversationSummary 重建会话摘要
	RebuildConversationSummary(ctx context.Context, conversationId string) (*entity.ChatMemorySummary, error)

	// DeleteConversationSummary 删除会话摘要
	DeleteConversationSummary(ctx context.Context, conversationId string) error
}

type PromptRenderer interface {
	Render(templateName string, variables map[string]any) (string, error)
}

type QueryRewriter interface {
	Rewrite(ctx context.Context, question, historySummary string, trace *vo.ConversationTrace) (*vo.QuestionRewriteResult, error)
}

// QuestionRecommender 追问推荐器
type QuestionRecommender interface {
	// GenerateRecommendations 生成推荐追问
	GenerateRecommendations(ctx context.Context, question, answer string, recentExchanges []*entity.ChatExchange, trace *vo.ConversationTrace) []string
}

// DocumentRouter 文档路由器
type DocumentRouter interface {
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

// GraphQuerier 结构图查询接口
type GraphQuerier interface {
	// ListSections 列出指定文档下的所有结构图节点（用于本地短语匹配）
	ListSections(ctx context.Context, documentId int64) ([]*entity.GraphSection, error)

	FindSectionById(ctx context.Context, documentId int64, nodeId int64) (*entity.GraphSection, error)

	// FindSectionByCode 根据编号（如 1.2.3 / 第 3 节）匹配章节节点
	FindSectionByCode(ctx context.Context, documentId int64, sectionCode string) (*entity.GraphSection, error)

	// FindBestSection 根据问题文本查找最佳节点；可接受一个可选的 anchor 短语增强
	FindBestSection(ctx context.Context, documentId int64, question, anchorHint string) (*entity.GraphSection, error)

	// FindSectionWithChildren 根据节点编号查找子节点
	FindSectionWithChildren(ctx context.Context, documentId int64, sectionNodeId int64) (*entity.GraphSectionWithChildren, error)

	// FindSectionWithSiblings 根据节点编号查找同级节点
	FindSectionWithSiblings(ctx context.Context, documentId int64, sectionNodeId int64) (*entity.GraphSectionWithSiblings, error)

	// BuildGraphResult 根据节点编号构建结构图结果
	BuildGraphResult(ctx context.Context, documentId int64, sectionNodeId int64, itemIndex int, itemKeyword string) (*entity.GraphQueryResult, error)
}
