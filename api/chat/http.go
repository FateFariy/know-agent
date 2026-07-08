package chat

import (
	"context"
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"

	"github.com/swiftbit/know-agent/common"
	"github.com/swiftbit/know-agent/internal/svc"
)

// HTTPServer 聊天服务HTTP接口
type HTTPServer interface {
	// StreamChat 流式聊天
	StreamChat(ctx context.Context, req *ChatReq) <-chan string

	// GetDocumentOptions 获取知识文档选项
	GetDocumentOptions(ctx context.Context) ([]*KnowledgeDocumentOptionResp, error)

	// StopConversation 停止会话
	StopConversation(ctx context.Context, req *ConversationIdentityReq) (*ConversationStopResp, error)

	// GetSessionDetail 获取会话详情
	GetSessionDetail(ctx context.Context, req *ConversationIdentityReq) (*ConversationSessionResp, error)

	// GetExchangeDetail 获取对话详情
	GetExchangeDetail(ctx context.Context, req *ConversationExchangeDetailQueryReq) (*ConversationExchangeDetailResp, error)

	// ListSessions 获取会话列表
	ListSessions(ctx context.Context, req *ConversationSessionListReq) (*ConversationSessionListResp, error)

	// ResetConversation 重置会话
	ResetConversation(ctx context.Context, req *ConversationIdentityReq) (*ConversationResetResp, error)

	// RebuildSummary 重建会话摘要
	RebuildSummary(ctx context.Context, req *ConversationIdentityReq) (*ConversationMemorySummaryResp, error)

	// GetRetrievalResults 获取检索结果
	GetRetrievalResults(ctx context.Context, req *RetrievalObserveReq) ([]*RetrievalResultResp, error)

	// GetChannelExecutions 获取渠道执行结果
	GetChannelExecutions(ctx context.Context, req *RetrievalObserveReq) ([]*ChannelExecutionResp, error)

	// GetStageBenchmarks 获取阶段基准
	GetStageBenchmarks(ctx context.Context) ([]*StageBenchmarkResp, error)
}

// StreamChatHandler 流式聊天
func StreamChatHandler(svcCtx *svc.ServiceContext, srv HTTPServer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req ChatReq
		if err := httpx.Parse(r, &req); err != nil {
			common.Response(w, nil, "", common.ErrInvalidParam.Format(err.Error()))
			return
		}

		w.Header().Set("Content-Type", "text/event-stream;charset=UTF-8")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		var stream <-chan string
		for {
			if stream == nil {
				stream = srv.StreamChat(r.Context(), &req)
			}
			select {
			case <-r.Context().Done():
				return
			case msg, ok := <-stream:
				if !ok {
					return
				}
				w.Write([]byte(msg))
				w.(http.Flusher).Flush()
			}
		}
	}
}

// GetDocumentOptionsHandler 获取知识文档选项
func GetDocumentOptionsHandler(svcCtx *svc.ServiceContext, srv HTTPServer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resp, err := srv.GetDocumentOptions(r.Context())
		common.Response(w, resp, "", err)
	}
}

// StopConversationHandler 停止会话
func StopConversationHandler(svcCtx *svc.ServiceContext, srv HTTPServer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req ConversationIdentityReq
		if err := httpx.Parse(r, &req); err != nil {
			common.Response(w, nil, "", common.ErrInvalidParam.Format(err.Error()))
			return
		}

		resp, err := srv.StopConversation(r.Context(), &req)
		common.Response(w, resp, "", err)
	}
}

// GetSessionDetailHandler 获取会话详情
func GetSessionDetailHandler(svcCtx *svc.ServiceContext, srv HTTPServer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req ConversationIdentityReq
		if err := httpx.Parse(r, &req); err != nil {
			common.Response(w, nil, "", common.ErrInvalidParam.Format(err.Error()))
			return
		}

		resp, err := srv.GetSessionDetail(r.Context(), &req)
		common.Response(w, resp, "", err)
	}
}

// GetExchangeDetailHandler 获取对话详情
func GetExchangeDetailHandler(svcCtx *svc.ServiceContext, srv HTTPServer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req ConversationExchangeDetailQueryReq
		if err := httpx.Parse(r, &req); err != nil {
			common.Response(w, nil, "", common.ErrInvalidParam.Format(err.Error()))
			return
		}

		resp, err := srv.GetExchangeDetail(r.Context(), &req)
		common.Response(w, resp, "", err)
	}
}

// ListSessionsHandler 获取会话列表
func ListSessionsHandler(svcCtx *svc.ServiceContext, srv HTTPServer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req ConversationSessionListReq
		if err := httpx.Parse(r, &req); err != nil {
			common.Response(w, nil, "", common.ErrInvalidParam.Format(err.Error()))
			return
		}

		resp, err := srv.ListSessions(r.Context(), &req)
		common.Response(w, resp, "", err)
	}
}

// ResetConversationHandler 重置会话
func ResetConversationHandler(svcCtx *svc.ServiceContext, srv HTTPServer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req ConversationIdentityReq
		if err := httpx.Parse(r, &req); err != nil {
			common.Response(w, nil, "", common.ErrInvalidParam.Format(err.Error()))
			return
		}

		resp, err := srv.ResetConversation(r.Context(), &req)
		common.Response(w, resp, "", err)
	}
}

// RebuildSummaryHandler 重建会话摘要
func RebuildSummaryHandler(svcCtx *svc.ServiceContext, srv HTTPServer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req ConversationIdentityReq
		if err := httpx.Parse(r, &req); err != nil {
			common.Response(w, nil, "", common.ErrInvalidParam.Format(err.Error()))
			return
		}

		resp, err := srv.RebuildSummary(r.Context(), &req)
		common.Response(w, resp, "", err)
	}
}

// GetRetrievalResultsHandler 获取检索结果
func GetRetrievalResultsHandler(svcCtx *svc.ServiceContext, srv HTTPServer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req RetrievalObserveReq
		if err := httpx.Parse(r, &req); err != nil {
			common.Response(w, nil, "", common.ErrInvalidParam.Format(err.Error()))
			return
		}

		resp, err := srv.GetRetrievalResults(r.Context(), &req)
		common.Response(w, resp, "", err)
	}
}

// GetChannelExecutionsHandler 获取渠道执行结果
func GetChannelExecutionsHandler(svcCtx *svc.ServiceContext, srv HTTPServer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req RetrievalObserveReq
		if err := httpx.Parse(r, &req); err != nil {
			common.Response(w, nil, "", common.ErrInvalidParam.Format(err.Error()))
			return
		}

		resp, err := srv.GetChannelExecutions(r.Context(), &req)
		common.Response(w, resp, "", err)
	}
}

// GetStageBenchmarksHandler 获取阶段基准
func GetStageBenchmarksHandler(svcCtx *svc.ServiceContext, srv HTTPServer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resp, err := srv.GetStageBenchmarks(r.Context())
		common.Response(w, resp, "", err)
	}
}
