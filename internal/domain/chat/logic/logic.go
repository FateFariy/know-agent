package logic

import (
	"context"

	"github.com/swiftbit/know-agent/api/chat"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
)

// ChatLogic 聊天业务逻辑接口
type ChatLogic interface {
	// OpenConversationStream 打开会话流
	OpenConversationStream(ctx context.Context, cmd *vo.ChatCommand) <-chan string

	// ListKnowledgeDocumentOptions 获取知识文档选项列表
	ListKnowledgeDocumentOptions(ctx context.Context) ([]*chat.KnowledgeDocumentOptionResp, error)

	// StopConversation 停止会话
	StopConversation(ctx context.Context, conversationId string) (*chat.ConversationStopResp, error)

	// GetSession 获取会话详情
	GetSession(ctx context.Context, conversationId string) (*chat.ConversationSessionResp, error)

	// GetExchangeDetail 获取对话详情
	GetExchangeDetail(ctx context.Context, conversationId, exchangeId string) (*chat.ConversationExchangeDetailResp, error)

	// ListSessions 获取会话列表
	ListSessions(ctx context.Context, req *chat.ConversationSessionListReq) (*chat.ConversationSessionListResp, error)

	// ResetConversation 重置会话
	ResetConversation(ctx context.Context, conversationId string) (*chat.ConversationResetResp, error)

	// RebuildConversationSummary 重建会话摘要
	RebuildConversationSummary(ctx context.Context, conversationId string) (*chat.ConversationMemorySummaryResp, error)

	// GetRetrievalResults 获取检索结果
	GetRetrievalResults(ctx context.Context, conversationId string, exchangeId int64) ([]*chat.RetrievalResultResp, error)

	// GetChannelExecutions 获取渠道执行结果
	GetChannelExecutions(ctx context.Context, conversationId string, exchangeId int64) ([]*chat.ChannelExecutionResp, error)

	// GetStageBenchmarks 获取阶段基准
	GetStageBenchmarks(ctx context.Context) ([]*chat.StageBenchmarkResp, error)
}
