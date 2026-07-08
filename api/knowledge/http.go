package knowledge

import (
	"context"
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"

	"github.com/swiftbit/know-agent/common"
	"github.com/swiftbit/know-agent/internal/svc"
)

type HTTPServer interface {
	// SaveKnowledgeScope 保存知识范围节点
	SaveKnowledgeScope(ctx context.Context, req *KnowledgeScopeSaveReq) (*KnowledgeScopeItem, error)

	// DeleteKnowledgeScope 删除知识范围节点
	DeleteKnowledgeScope(ctx context.Context, req *KnowledgeScopeDeleteReq) (bool, error)

	// ListKnowledgeScope 查询知识范围列表
	ListKnowledgeScope(ctx context.Context) ([]*KnowledgeScopeItem, error)

	// SaveKnowledgeTopic 保存知识主题节点
	SaveKnowledgeTopic(ctx context.Context, req *KnowledgeTopicSaveReq) (*KnowledgeTopicItem, error)

	// DeleteKnowledgeTopic 删除知识主题节点
	DeleteKnowledgeTopic(ctx context.Context, req *KnowledgeTopicDeleteReq) (bool, error)

	// ListKnowledgeTopic 查询知识主题列表
	ListKnowledgeTopic(ctx context.Context, req *KnowledgeTopicListReq) ([]*KnowledgeTopicItem, error)

	// GetDocumentProfile 查询文档画像详情
	GetDocumentProfile(ctx context.Context, req *DocumentProfileDetailReq) (*DocumentProfileResp, error)

	// RegenerateDocumentProfile 重新生成文档画像
	RegenerateDocumentProfile(ctx context.Context, req *DocumentProfileRegenerateReq) (*DocumentProfileResp, error)

	// BatchRegenerateDocumentProfile 批量重新生成文档画像
	BatchRegenerateDocumentProfile(ctx context.Context, req *DocumentProfileBatchRegenerateReq) ([]*DocumentProfileResp, error)

	// ListTopicDocumentRelation 查询主题文档关联
	ListTopicDocumentRelation(ctx context.Context, req *TopicDocumentRelationListReq) ([]*TopicDocumentRelationItem, error)

	// SaveTopicDocumentRelation 保存主题文档关联
	SaveTopicDocumentRelation(ctx context.Context, req *TopicDocumentRelationSaveReq) (*TopicDocumentRelationItem, error)

	// RemoveTopicDocumentRelation 移除主题文档关联
	RemoveTopicDocumentRelation(ctx context.Context, req *TopicDocumentRelationRemoveReq) (bool, error)

	// QueryKnowledgeRouteTracePage 分页查询知识路由追踪
	QueryKnowledgeRouteTracePage(ctx context.Context, req *KnowledgeRouteTracePageReq) (*KnowledgeRouteTracePageResp, error)
}

// SaveKnowledgeScopeHandler 保存知识范围节点
func SaveKnowledgeScopeHandler(svcCtx *svc.ServiceContext, srv HTTPServer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req KnowledgeScopeSaveReq
		if err := httpx.Parse(r, &req); err != nil {
			common.Response(w, nil, "", common.ErrInvalidParam.Format(err.Error()))
			return
		}

		resp, err := srv.SaveKnowledgeScope(r.Context(), &req)
		common.Response(w, resp, "", err)
	}
}

// DeleteKnowledgeScopeHandler 删除知识范围节点
func DeleteKnowledgeScopeHandler(svcCtx *svc.ServiceContext, srv HTTPServer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req KnowledgeScopeDeleteReq
		if err := httpx.Parse(r, &req); err != nil {
			common.Response(w, nil, "", common.ErrInvalidParam.Format(err.Error()))
			return
		}

		resp, err := srv.DeleteKnowledgeScope(r.Context(), &req)
		common.Response(w, resp, "", err)
	}
}

// ListKnowledgeScopeHandler 查询知识范围列表
func ListKnowledgeScopeHandler(svcCtx *svc.ServiceContext, srv HTTPServer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resp, err := srv.ListKnowledgeScope(r.Context())
		common.Response(w, resp, "", err)
	}
}

// SaveKnowledgeTopicHandler 保存知识主题节点
func SaveKnowledgeTopicHandler(svcCtx *svc.ServiceContext, srv HTTPServer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req KnowledgeTopicSaveReq
		if err := httpx.Parse(r, &req); err != nil {
			common.Response(w, nil, "", common.ErrInvalidParam.Format(err.Error()))
			return
		}

		resp, err := srv.SaveKnowledgeTopic(r.Context(), &req)
		common.Response(w, resp, "", err)
	}
}

// DeleteKnowledgeTopicHandler 删除知识主题节点
func DeleteKnowledgeTopicHandler(svcCtx *svc.ServiceContext, srv HTTPServer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req KnowledgeTopicDeleteReq
		if err := httpx.Parse(r, &req); err != nil {
			common.Response(w, nil, "", common.ErrInvalidParam.Format(err.Error()))
			return
		}

		resp, err := srv.DeleteKnowledgeTopic(r.Context(), &req)
		common.Response(w, resp, "", err)
	}
}

// ListKnowledgeTopicHandler 查询知识主题列表
func ListKnowledgeTopicHandler(svcCtx *svc.ServiceContext, srv HTTPServer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req KnowledgeTopicListReq
		if err := httpx.Parse(r, &req); err != nil {
			common.Response(w, nil, "", common.ErrInvalidParam.Format(err.Error()))
			return
		}

		resp, err := srv.ListKnowledgeTopic(r.Context(), &req)
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

// ListTopicDocumentRelationHandler 查询主题文档关联
func ListTopicDocumentRelationHandler(svcCtx *svc.ServiceContext, srv HTTPServer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req TopicDocumentRelationListReq
		if err := httpx.Parse(r, &req); err != nil {
			common.Response(w, nil, "", common.ErrInvalidParam.Format(err.Error()))
			return
		}

		resp, err := srv.ListTopicDocumentRelation(r.Context(), &req)
		common.Response(w, resp, "", err)
	}
}

// SaveTopicDocumentRelationHandler 保存主题文档关联
func SaveTopicDocumentRelationHandler(svcCtx *svc.ServiceContext, srv HTTPServer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req TopicDocumentRelationSaveReq
		if err := httpx.Parse(r, &req); err != nil {
			common.Response(w, nil, "", common.ErrInvalidParam.Format(err.Error()))
			return
		}

		resp, err := srv.SaveTopicDocumentRelation(r.Context(), &req)
		common.Response(w, resp, "", err)
	}
}

// RemoveTopicDocumentRelationHandler 移除主题文档关联
func RemoveTopicDocumentRelationHandler(svcCtx *svc.ServiceContext, srv HTTPServer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req TopicDocumentRelationRemoveReq
		if err := httpx.Parse(r, &req); err != nil {
			common.Response(w, nil, "", common.ErrInvalidParam.Format(err.Error()))
			return
		}

		resp, err := srv.RemoveTopicDocumentRelation(r.Context(), &req)
		common.Response(w, resp, "", err)
	}
}

// QueryKnowledgeRouteTracePageHandler 分页查询知识路由追踪
func QueryKnowledgeRouteTracePageHandler(svcCtx *svc.ServiceContext, srv HTTPServer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req KnowledgeRouteTracePageReq
		if err := httpx.Parse(r, &req); err != nil {
			common.Response(w, nil, "", common.ErrInvalidParam.Format(err.Error()))
			return
		}

		resp, err := srv.QueryKnowledgeRouteTracePage(r.Context(), &req)
		common.Response(w, resp, "", err)
	}
}
