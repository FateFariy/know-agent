// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package knowledge

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest"
)

func RegisterHandlers(server *rest.Server, srv HTTPServer) {
	server.AddRoutes(
		[]rest.Route{
			{
				Method:  http.MethodPost,
				Path:    "/scope/save",
				Handler: SaveKnowledgeScopeHandler(srv),
			},
			{
				Method:  http.MethodPost,
				Path:    "/scope/delete",
				Handler: DeleteKnowledgeScopeHandler(srv),
			},
			{
				Method:  http.MethodPost,
				Path:    "/scope/list",
				Handler: ListKnowledgeScopeHandler(srv),
			},
			{
				Method:  http.MethodPost,
				Path:    "/topic/save",
				Handler: SaveKnowledgeTopicHandler(srv),
			},
			{
				Method:  http.MethodPost,
				Path:    "/topic/delete",
				Handler: DeleteKnowledgeTopicHandler(srv),
			},
			{
				Method:  http.MethodPost,
				Path:    "/topic/list",
				Handler: ListKnowledgeTopicHandler(srv),
			},
			{
				Method:  http.MethodPost,
				Path:    "/topic/document/list",
				Handler: ListTopicDocumentRelationHandler(srv),
			},
			{
				Method:  http.MethodPost,
				Path:    "/topic/document/save",
				Handler: SaveTopicDocumentRelationHandler(srv),
			},
			{
				Method:  http.MethodPost,
				Path:    "/topic/document/remove",
				Handler: RemoveTopicDocumentRelationHandler(srv),
			},
			{
				Method:  http.MethodPost,
				Path:    "/route/trace/page/query",
				Handler: QueryKnowledgeRouteTracePageHandler(srv),
			},
			{
				Method:  http.MethodPost,
				Path:    "base/list",
				Handler: ListKnowledgeBaseHandler(srv),
			},
		},
		rest.WithPrefix("/manage/knowledge"),
	)
}
