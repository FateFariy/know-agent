// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package knowledge

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest"

	"github.com/swiftbit/know-agent/internal/svc"
)

func RegisterHandlers(server *rest.Server, svcCtx *svc.ServiceContext, srv HTTPServer) {
	server.AddRoutes(
		[]rest.Route{
			{
				Method:  http.MethodPost,
				Path:    "/scope/save",
				Handler: SaveKnowledgeScopeHandler(svcCtx, srv),
			},
			{
				Method:  http.MethodPost,
				Path:    "/scope/delete",
				Handler: DeleteKnowledgeScopeHandler(svcCtx, srv),
			},
			{
				Method:  http.MethodPost,
				Path:    "/scope/list",
				Handler: ListKnowledgeScopeHandler(svcCtx, srv),
			},
			{
				Method:  http.MethodPost,
				Path:    "/topic/save",
				Handler: SaveKnowledgeTopicHandler(svcCtx, srv),
			},
			{
				Method:  http.MethodPost,
				Path:    "/topic/delete",
				Handler: DeleteKnowledgeTopicHandler(svcCtx, srv),
			},
			{
				Method:  http.MethodPost,
				Path:    "/topic/list",
				Handler: ListKnowledgeTopicHandler(svcCtx, srv),
			},
			{
				Method:  http.MethodPost,
				Path:    "/document/profile/detail",
				Handler: GetDocumentProfileHandler(svcCtx, srv),
			},
			{
				Method:  http.MethodPost,
				Path:    "/document/profile/regenerate",
				Handler: RegenerateDocumentProfileHandler(svcCtx, srv),
			},
			{
				Method:  http.MethodPost,
				Path:    "/document/profile/batch/regenerate",
				Handler: BatchRegenerateDocumentProfileHandler(svcCtx, srv),
			},
			{
				Method:  http.MethodPost,
				Path:    "/topic/document/list",
				Handler: ListTopicDocumentRelationHandler(svcCtx, srv),
			},
			{
				Method:  http.MethodPost,
				Path:    "/topic/document/save",
				Handler: SaveTopicDocumentRelationHandler(svcCtx, srv),
			},
			{
				Method:  http.MethodPost,
				Path:    "/topic/document/remove",
				Handler: RemoveTopicDocumentRelationHandler(svcCtx, srv),
			},
			{
				Method:  http.MethodPost,
				Path:    "/route/trace/page/query",
				Handler: QueryKnowledgeRouteTracePageHandler(svcCtx, srv),
			},
		},
		rest.WithPrefix("/manage/knowledge"),
	)
}
