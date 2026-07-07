package handler

import (
	"context"

	"github.com/swiftbit/know-agent/api/chat"
	"github.com/swiftbit/know-agent/internal/convert"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
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
func (c *ChatService) StreamChat(ctx context.Context, req *chat.ChatReq) <-chan string {
	return c.l.OpenConversationStream(ctx, convert.FromChatReq(req))
}

// GetDocumentOptions 获取知识文档选项
func (c *ChatService) GetDocumentOptions(ctx context.Context) ([]*chat.KnowledgeDocumentOptionResp, error) {
	return c.l.ListKnowledgeDocumentOptions(ctx)
}

// StopConversation 停止会话
func (c *ChatService) StopConversation(ctx context.Context, req *chat.ConversationIdentityReq) (*chat.ConversationStopResp, error) {
	stopped, message, err := c.l.StopConversation(ctx, req.ConversationId)
	return &chat.ConversationStopResp{
		ConversationId: req.ConversationId,
		Stopped:        stopped,
		Message:        message,
	}, err
}

// GetSessionDetail 获取会话详情
func (c *ChatService) GetSessionDetail(ctx context.Context, req *chat.ConversationIdentityReq) (*chat.ConversationSessionResp, error) {
	detail, err := c.l.GetSessionDetail(ctx, req.ConversationId)
	if err != nil {
		return nil, err
	}
	return convert.ToConversationSessionResp(detail), err
}

// GetExchangeDetail 获取对话详情
func (c *ChatService) GetExchangeDetail(ctx context.Context, req *chat.ConversationExchangeDetailQueryReq) (*chat.ConversationExchangeDetailResp, error) {
	detail, stages, err := c.l.GetExchangeDetail(ctx, req.ConversationId, req.ExchangeId)
	if err != nil {
		return nil, err
	}
	return &chat.ConversationExchangeDetailResp{
		ConversationId: req.ConversationId,
		Exchange:       convert.ToConversationExchangeResp(detail),
		StageTraces:    convert.ToConversationStageTraceRespList(stages),
	}, nil
}

// ListSessions 获取会话列表
func (c *ChatService) ListSessions(ctx context.Context, req *chat.ConversationSessionListReq) (*chat.ConversationSessionListResp, error) {
	records, total, err := c.l.ListSessions(ctx, req.PageNo, req.PageSize, vo.ToChatQueryMode(req.ChatMode), vo.ToChatTurnStatus(req.TurnStatus), req.Keyword)
	if err != nil {
		return nil, err
	}
	return &chat.ConversationSessionListResp{
		PageNo:     req.PageNo,
		PageSize:   req.PageSize,
		TotalPages: (total + int64(req.PageSize) - 1) / int64(req.PageSize),
		TotalSize:  total,
		Records:    convert.ToConversationSessionRespList(records),
	}, err
}

// ResetConversation 重置会话
func (c *ChatService) ResetConversation(ctx context.Context, req *chat.ConversationIdentityReq) (*chat.ConversationResetResp, error) {
	reset, err := c.l.ResetConversation(ctx, req.ConversationId)
	if err != nil {
		return nil, err
	}
	return convert.ToConversationResetResp(reset), err
}

// RebuildSummary 重建会话摘要
func (c *ChatService) RebuildSummary(ctx context.Context, req *chat.ConversationIdentityReq) (*chat.ConversationMemorySummaryResp, error) {
	return c.l.RebuildConversationSummary(ctx, req.ConversationId)
}

// GetRetrievalResults 获取检索结果
func (c *ChatService) GetRetrievalResults(ctx context.Context, req *chat.RetrievalObserveReq) ([]*chat.RetrievalResultResp, error) {
	results, err := c.l.GetRetrievalResults(ctx, req.ConversationId, req.ExchangeId)
	if err != nil {
		return nil, err
	}
	return convert.ToRetrievalResultRespList(results), err
}

// GetChannelExecutions 获取渠道执行结果
func (c *ChatService) GetChannelExecutions(ctx context.Context, req *chat.RetrievalObserveReq) ([]*chat.ChannelExecutionResp, error) {
	executions, err := c.l.GetChannelExecutions(ctx, req.ConversationId, req.ExchangeId)
	if err != nil {
		return nil, err
	}
	return convert.ToChannelExecutionRespList(executions), err
}

// GetStageBenchmarks 获取阶段基准
func (c *ChatService) GetStageBenchmarks(ctx context.Context) ([]*chat.StageBenchmarkResp, error) {
	// todo 待实现
	panic("implemented")
}
