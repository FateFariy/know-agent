package document

import (
	"context"
	"mime/multipart"
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"

	"github.com/swiftbit/know-agent/common"
	"github.com/swiftbit/know-agent/internal/svc"
)

type HTTPServer interface {
	// UploadDocument 上传文档
	UploadDocument(ctx context.Context, file multipart.File, header *multipart.FileHeader, req *UploadDocumentReq) (*UploadDocumentResp, error)

	// QueryDocumentPage 分页查询文档列表
	QueryDocumentPage(ctx context.Context, req *QueryDocumentPageReq) (*QueryDocumentPageResp, error)

	// QueryDocumentDetail 查询文档详情
	QueryDocumentDetail(ctx context.Context, req *QueryDocumentDetailReq) (*DocumentDetailResp, error)

	// GetDocumentOptions 获取知识文档选项
	GetDocumentOptions(ctx context.Context) ([]*KnowledgeDocumentOptionResp, error)

	// DeleteDocument 删除文档
	DeleteDocument(ctx context.Context, req *DeleteDocumentReq) (*DeleteDocumentResp, error)

	// QueryStrategyPlan 查询文档策略方案
	QueryStrategyPlan(ctx context.Context, req *QueryStrategyPlanReq) (*QueryStrategyPlanResp, error)

	// ConfirmStrategy 确认文档策略方案
	ConfirmStrategy(ctx context.Context, req *ConfirmStrategyReq) (*ConfirmStrategyResp, error)

	// BuildIndex 构建文档索引
	BuildIndex(ctx context.Context, req *BuildIndexReq) (*BuildIndexResp, error)

	// QueryDocumentChunks 查询文档chunk列表
	QueryDocumentChunks(ctx context.Context, req *QueryDocumentChunksReq) (*QueryDocumentChunksResp, error)

	// QueryDocumentChunkDetail 查询文档chunk详情
	QueryDocumentChunkDetail(ctx context.Context, req *QueryDocumentChunkDetailReq) (*QueryDocumentChunkDetailResp, error)

	// QueryTaskLogs 查询任务日志
	QueryTaskLogs(ctx context.Context, req *QueryTaskLogsReq) (*QueryTaskLogsResp, error)

	// GetDocumentProfile 查询文档画像详情
	GetDocumentProfile(ctx context.Context, req *DocumentProfileDetailReq) (*DocumentProfileResp, error)

	// RegenerateDocumentProfile 重新生成文档画像
	RegenerateDocumentProfile(ctx context.Context, req *DocumentProfileRegenerateReq) (*DocumentProfileResp, error)

	// BatchRegenerateDocumentProfile 批量重新生成文档画像
	BatchRegenerateDocumentProfile(ctx context.Context, req *DocumentProfileBatchRegenerateReq) ([]*DocumentProfileResp, error)
}

// UploadDocumentHandler 上传文档
func UploadDocumentHandler(svcCtx *svc.ServiceContext, srv HTTPServer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req UploadDocumentReq
		if err := httpx.Parse(r, &req); err != nil {
			common.Response(w, nil, "", common.ErrInvalidParam.Format(err.Error()))
			return
		}

		file, header, err := r.FormFile("file")
		if err != nil {
			common.Response(w, nil, "", common.ErrInvalidParam.Format(err.Error()))
			return
		}
		defer file.Close()

		resp, err := srv.UploadDocument(r.Context(), file, header, &req)
		common.Response(w, resp, "", err)
	}
}

// QueryDocumentPageHandler 分页查询文档列表
func QueryDocumentPageHandler(svcCtx *svc.ServiceContext, srv HTTPServer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req QueryDocumentPageReq
		if err := httpx.Parse(r, &req); err != nil {
			common.Response(w, nil, "", common.ErrInvalidParam.Format(err.Error()))
			return
		}

		resp, err := srv.QueryDocumentPage(r.Context(), &req)
		common.Response(w, resp, "", err)
	}
}

// QueryDocumentDetailHandler 查询文档详情
func QueryDocumentDetailHandler(svcCtx *svc.ServiceContext, srv HTTPServer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req QueryDocumentDetailReq
		if err := httpx.Parse(r, &req); err != nil {
			common.Response(w, nil, "", common.ErrInvalidParam.Format(err.Error()))
			return
		}

		resp, err := srv.QueryDocumentDetail(r.Context(), &req)
		common.Response(w, resp, "", err)
	}
}

// GetDocumentOptionsHandler 获取知识文档选项
func GetDocumentOptionsHandler(svcCtx *svc.ServiceContext, srv HTTPServer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resp, err := srv.GetDocumentOptions(r.Context())
		common.Response(w, resp, "", err)
	}
}

// DeleteDocumentHandler 删除文档
func DeleteDocumentHandler(svcCtx *svc.ServiceContext, srv HTTPServer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req DeleteDocumentReq
		if err := httpx.Parse(r, &req); err != nil {
			common.Response(w, nil, "", common.ErrInvalidParam.Format(err.Error()))
			return
		}

		resp, err := srv.DeleteDocument(r.Context(), &req)
		common.Response(w, resp, "", err)
	}
}

// QueryStrategyPlanHandler 查询策略方案
func QueryStrategyPlanHandler(svcCtx *svc.ServiceContext, srv HTTPServer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req QueryStrategyPlanReq
		if err := httpx.Parse(r, &req); err != nil {
			common.Response(w, nil, "", common.ErrInvalidParam.Format(err.Error()))
			return
		}

		resp, err := srv.QueryStrategyPlan(r.Context(), &req)
		common.Response(w, resp, "", err)
	}
}

// ConfirmStrategyHandler 确认策略
func ConfirmStrategyHandler(svcCtx *svc.ServiceContext, srv HTTPServer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req ConfirmStrategyReq
		if err := httpx.Parse(r, &req); err != nil {
			common.Response(w, nil, "", common.ErrInvalidParam.Format(err.Error()))
			return
		}

		resp, err := srv.ConfirmStrategy(r.Context(), &req)
		common.Response(w, resp, "", err)
	}
}

// BuildIndexHandler 构建索引
func BuildIndexHandler(svcCtx *svc.ServiceContext, srv HTTPServer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req BuildIndexReq
		if err := httpx.Parse(r, &req); err != nil {
			common.Response(w, nil, "", common.ErrInvalidParam.Format(err.Error()))
			return
		}

		resp, err := srv.BuildIndex(r.Context(), &req)
		common.Response(w, resp, "", err)
	}
}

// QueryDocumentChunksHandler 查询文档chunk列表
func QueryDocumentChunksHandler(svcCtx *svc.ServiceContext, srv HTTPServer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req QueryDocumentChunksReq
		if err := httpx.Parse(r, &req); err != nil {
			common.Response(w, nil, "", common.ErrInvalidParam.Format(err.Error()))
			return
		}

		resp, err := srv.QueryDocumentChunks(r.Context(), &req)
		common.Response(w, resp, "", err)
	}
}

// QueryDocumentChunkDetailHandler 查询文档chunk详情
func QueryDocumentChunkDetailHandler(svcCtx *svc.ServiceContext, srv HTTPServer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req QueryDocumentChunkDetailReq
		if err := httpx.Parse(r, &req); err != nil {
			common.Response(w, nil, "", common.ErrInvalidParam.Format(err.Error()))
			return
		}

		resp, err := srv.QueryDocumentChunkDetail(r.Context(), &req)
		common.Response(w, resp, "", err)
	}
}

// QueryTaskLogsHandler 查询任务日志
func QueryTaskLogsHandler(svcCtx *svc.ServiceContext, srv HTTPServer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req QueryTaskLogsReq
		if err := httpx.Parse(r, &req); err != nil {
			common.Response(w, nil, "", common.ErrInvalidParam.Format(err.Error()))
			return
		}

		resp, err := srv.QueryTaskLogs(r.Context(), &req)
		common.Response(w, resp, "", err)
	}
}

// GetDocumentProfileHandler 查询文档画像详情
func GetDocumentProfileHandler(svcCtx *svc.ServiceContext, srv HTTPServer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req DocumentProfileDetailReq
		if err := httpx.Parse(r, &req); err != nil {
			common.Response(w, nil, "", common.ErrInvalidParam.Format(err.Error()))
			return
		}

		resp, err := srv.GetDocumentProfile(r.Context(), &req)
		common.Response(w, resp, "", err)
	}
}

// RegenerateDocumentProfileHandler 重新生成文档画像
func RegenerateDocumentProfileHandler(svcCtx *svc.ServiceContext, srv HTTPServer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req DocumentProfileRegenerateReq
		if err := httpx.Parse(r, &req); err != nil {
			common.Response(w, nil, "", common.ErrInvalidParam.Format(err.Error()))
			return
		}

		resp, err := srv.RegenerateDocumentProfile(r.Context(), &req)
		common.Response(w, resp, "", err)
	}
}

// BatchRegenerateDocumentProfileHandler 批量重新生成文档画像
func BatchRegenerateDocumentProfileHandler(svcCtx *svc.ServiceContext, srv HTTPServer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req DocumentProfileBatchRegenerateReq
		if err := httpx.Parse(r, &req); err != nil {
			common.Response(w, nil, "", common.ErrInvalidParam.Format(err.Error()))
			return
		}

		resp, err := srv.BatchRegenerateDocumentProfile(r.Context(), &req)
		common.Response(w, resp, "", err)
	}
}
