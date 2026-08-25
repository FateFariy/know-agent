package chat

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest"
)

// RegisterHandlers 注册聊天服务路由
func RegisterHandlers(server *rest.Server, srv HTTPServer) {
	server.AddRoutes(
		[]rest.Route{
			{
				Method:  http.MethodPost,
				Path:    "/session/stop",
				Handler: StopConversationHandler(srv),
			},
			{
				Method:  http.MethodPost,
				Path:    "/session/detail",
				Handler: GetSessionDetailHandler(srv),
			},
			{
				Method:  http.MethodPost,
				Path:    "/exchange/detail",
				Handler: GetExchangeDetailHandler(srv),
			},
			{
				Method:  http.MethodPost,
				Path:    "/session/list",
				Handler: ListSessionsHandler(srv),
			},
			{
				Method:  http.MethodPost,
				Path:    "/session/reset",
				Handler: ResetConversationHandler(srv),
			},
			{
				Method:  http.MethodPost,
				Path:    "/session/summary/rebuild",
				Handler: RebuildSummaryHandler(srv),
			},
			{
				Method:  http.MethodPost,
				Path:    "/exchange/retrieval/results",
				Handler: GetRetrievalResultsHandler(srv),
			},
			{
				Method:  http.MethodPost,
				Path:    "/exchange/channel/executions",
				Handler: GetChannelExecutionsHandler(srv),
			},
		},
		rest.WithPrefix("/chat"),
	)
	server.AddRoutes([]rest.Route{
		{
			Method:  http.MethodPost,
			Path:    "/stream",
			Handler: StreamChatHandler(srv),
		},
	}, rest.WithPrefix("/chat"), rest.WithSSE())
}
