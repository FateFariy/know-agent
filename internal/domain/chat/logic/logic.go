package logic

import (
	"context"

	"github.com/swiftbit/know-agent/internal/domain/chat/adapter"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/aggregate"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
)

// ConversationLogic 聊天业务逻辑接口
type ConversationLogic interface {
	// OpenConversationStream 打开会话流
	OpenConversationStream(ctx context.Context, sink adapter.Sink, cmd *vo.ChatCommand) error

	// StopConversation 停止会话
	StopConversation(ctx context.Context, conversationId string) (bool, string, error)

	// GetSessionDetail 获取会话详情
	GetSessionDetail(ctx context.Context, conversationId string) (*aggregate.ConversationArchiveRecord, error)

	// GetExchangeDetail 获取对话详情
	GetExchangeDetail(ctx context.Context, conversationId string, exchangeId int64) (*entity.ChatExchange, []*entity.ChatExchangeTraceStage, error)

	// ListSessions 获取会话列表
	ListSessions(ctx context.Context, pageNo, pageSize, chatMode, latestTurnStatus int, keyword string) ([]*aggregate.ConversationArchiveRecord, int64, error)

	// ResetConversation 重置会话
	ResetConversation(ctx context.Context, conversationId string) (*vo.ConversationReset, error)

	// RebuildConversationSummary 重建会话摘要
	RebuildConversationSummary(ctx context.Context, conversationId string) (*entity.ChatMemorySummary, error)

	// GetRetrievalResults 获取检索结果
	GetRetrievalResults(ctx context.Context, conversationId string, exchangeId int64) ([]*vo.ChatRetrievalResult, error)

	// GetChannelExecutions 获取渠道执行结果
	GetChannelExecutions(ctx context.Context, conversationId string, exchangeId int64) ([]*vo.ChatChannelExecution, error)
}
