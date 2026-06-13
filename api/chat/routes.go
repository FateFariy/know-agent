package chat

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest"

	"github.com/swiftbit/know-agent/internal/svc"
)

// RegisterHandlers 注册聊天服务路由
func RegisterHandlers(server *rest.Server, svcCtx *svc.ServiceContext, srv HTTPServer) {
	server.AddRoutes(
		[]rest.Route{
			{
				Method:  http.MethodPost,
				Path:    "/stream",
				Handler: StreamChatHandler(svcCtx, srv),
			},
			{
				Method:  http.MethodPost,
				Path:    "/document/options",
				Handler: GetDocumentOptionsHandler(svcCtx, srv),
			},
			{
				Method:  http.MethodPost,
				Path:    "/session/stop",
				Handler: StopConversationHandler(svcCtx, srv),
			},
			{
				Method:  http.MethodPost,
				Path:    "/session/detail",
				Handler: GetSessionDetailHandler(svcCtx, srv),
			},
			{
				Method:  http.MethodPost,
				Path:    "/exchange/detail",
				Handler: GetExchangeDetailHandler(svcCtx, srv),
			},
			{
				Method:  http.MethodPost,
				Path:    "/session/list",
				Handler: ListSessionsHandler(svcCtx, srv),
			},
			{
				Method:  http.MethodPost,
				Path:    "/session/reset",
				Handler: ResetConversationHandler(svcCtx, srv),
			},
			{
				Method:  http.MethodPost,
				Path:    "/session/summary/rebuild",
				Handler: RebuildSummaryHandler(svcCtx, srv),
			},
			{
				Method:  http.MethodPost,
				Path:    "/exchange/retrieval/results",
				Handler: GetRetrievalResultsHandler(svcCtx, srv),
			},
			{
				Method:  http.MethodPost,
				Path:    "/exchange/channel/executions",
				Handler: GetChannelExecutionsHandler(svcCtx, srv),
			},
			{
				Method:  http.MethodPost,
				Path:    "/stage/benchmarks",
				Handler: GetStageBenchmarksHandler(svcCtx, srv),
			},
		},
		rest.WithPrefix("/api/chat"), rest.WithSSE(),
	)
}
