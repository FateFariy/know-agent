package handler

import (
	"context"
	"strconv"

	"github.com/swiftbit/know-agent/api/chat"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic"
)

// ChatService 聊天服务实现
type ChatService struct {
	l logic.ChatLogic
}

var _ chat.HTTPServer = (*ChatService)(nil)

// NewChatService 创建聊天服务实例
func NewChatService(l logic.ChatLogic) *ChatService {
	return &ChatService{
		l: l,
	}
}

// StreamChat 流式聊天
func (c *ChatService) StreamChat(ctx context.Context, req *chat.ChatReq) (<-chan string, error) {
	return c.l.OpenConversationStream(ctx, req)
}

// GetDocumentOptions 获取知识文档选项
func (c *ChatService) GetDocumentOptions(ctx context.Context) ([]*chat.KnowledgeDocumentOptionResp, error) {
	return c.l.ListKnowledgeDocumentOptions(ctx)
}

// StopConversation 停止会话
func (c *ChatService) StopConversation(ctx context.Context, req *chat.ConversationIdentityReq) (*chat.ConversationStopResp, error) {
	return c.l.StopConversation(ctx, req.ConversationId)
}

// GetSessionDetail 获取会话详情
func (c *ChatService) GetSessionDetail(ctx context.Context, req *chat.ConversationIdentityReq) (*chat.ConversationSessionResp, error) {
	return c.l.GetSession(ctx, req.ConversationId)
}

// GetExchangeDetail 获取对话详情
func (c *ChatService) GetExchangeDetail(ctx context.Context, req *chat.ConversationExchangeDetailQueryReq) (*chat.ConversationExchangeDetailResp, error) {
	return c.l.GetExchangeDetail(ctx, req.ConversationId, req.ExchangeId)
}

// ListSessions 获取会话列表
func (c *ChatService) ListSessions(ctx context.Context, req *chat.ConversationSessionListReq) (*chat.ConversationSessionListResp, error) {
	return c.l.ListSessions(ctx, req)
}

// ResetConversation 重置会话
func (c *ChatService) ResetConversation(ctx context.Context, req *chat.ConversationIdentityReq) (*chat.ConversationResetResp, error) {
	return c.l.ResetConversation(ctx, req.ConversationId)
}

// RebuildSummary 重建会话摘要
func (c *ChatService) RebuildSummary(ctx context.Context, req *chat.ConversationIdentityReq) (*chat.ConversationMemorySummaryResp, error) {
	return c.l.RebuildConversationSummary(ctx, req.ConversationId)
}

// GetRetrievalResults 获取检索结果
func (c *ChatService) GetRetrievalResults(ctx context.Context, req *chat.RetrievalObserveReq) ([]*chat.RetrievalResultResp, error) {
	exchangeId, err := strconv.ParseInt(req.ExchangeId, 10, 64)
	if err != nil {
		return nil, err
	}
	return c.l.GetRetrievalResults(ctx, req.ConversationId, exchangeId)
}

// GetChannelExecutions 获取渠道执行结果
func (c *ChatService) GetChannelExecutions(ctx context.Context, req *chat.RetrievalObserveReq) ([]*chat.ChannelExecutionResp, error) {
	exchangeId, err := strconv.ParseInt(req.ExchangeId, 10, 64)
	if err != nil {
		return nil, err
	}
	return c.l.GetChannelExecutions(ctx, req.ConversationId, exchangeId)
}

// GetStageBenchmarks 获取阶段基准
func (c *ChatService) GetStageBenchmarks(ctx context.Context) ([]*chat.StageBenchmarkResp, error) {
	return c.l.GetStageBenchmarks(ctx)
}
