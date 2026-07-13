// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package document

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
				Path:    "/upload",
				Handler: UploadDocumentHandler(svcCtx, srv),
			},
			{
				Method:  http.MethodPost,
				Path:    "/page/query",
				Handler: QueryDocumentPageHandler(svcCtx, srv),
			},
			{
				Method:  http.MethodPost,
				Path:    "/detail/query",
				Handler: QueryDocumentDetailHandler(svcCtx, srv),
			},
			{
				Method:  http.MethodPost,
				Path:    "/delete",
				Handler: DeleteDocumentHandler(svcCtx, srv),
			},
			{
				Method:  http.MethodPost,
				Path:    "/strategy/plan/query",
				Handler: QueryStrategyPlanHandler(svcCtx, srv),
			},
			{
				Method:  http.MethodPost,
				Path:    "/strategy/confirm",
				Handler: ConfirmStrategyHandler(svcCtx, srv),
			},
			{
				Method:  http.MethodPost,
				Path:    "/index/build",
				Handler: BuildIndexHandler(svcCtx, srv),
			},
			{
				Method:  http.MethodPost,
				Path:    "/chunk/query",
				Handler: QueryDocumentChunksHandler(svcCtx, srv),
			},
			{
				Method:  http.MethodPost,
				Path:    "/chunk/detail/query",
				Handler: QueryDocumentChunkDetailHandler(svcCtx, srv),
			},
			{
				Method:  http.MethodPost,
				Path:    "/document/options",
				Handler: GetDocumentOptionsHandler(svcCtx, srv),
			},
			{
				Method:  http.MethodPost,
				Path:    "/task/log/query",
				Handler: QueryTaskLogsHandler(svcCtx, srv),
			},
			{
				Method:  http.MethodPost,
				Path:    "/profile/detail",
				Handler: GetDocumentProfileHandler(svcCtx, srv),
			},
			{
				Method:  http.MethodPost,
				Path:    "/profile/regenerate",
				Handler: RegenerateDocumentProfileHandler(svcCtx, srv),
			},
			{
				Method:  http.MethodPost,
				Path:    "/profile/batch/regenerate",
				Handler: BatchRegenerateDocumentProfileHandler(svcCtx, srv),
			},
		},
		rest.WithPrefix("/manage/document"),
	)
}
