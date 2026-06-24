package handler

import (
	"context"
	"mime/multipart"

	"github.com/duke-git/lancet/v2/strutil"

	"github.com/swiftbit/know-agent/api/document"
	"github.com/swiftbit/know-agent/internal/convert"
	"github.com/swiftbit/know-agent/internal/domain/document/logic"
)

type DocumentService struct {
	l logic.LifecycleLogic
}

var _ document.HTTPServer = (*DocumentService)(nil)

func NewDocumentService(l logic.LifecycleLogic) *DocumentService {
	return &DocumentService{
		l: l,
	}
}

// UploadDocument 上传文档
func (d *DocumentService) UploadDocument(ctx context.Context, file multipart.File, header *multipart.FileHeader, req *document.UploadDocumentReq) (*document.UploadDocumentResp, error) {
	doc := convert.FromUploadDocumentReq(req)
	documentUpload, err := d.l.Upload(ctx, file, header, doc)
	return convert.ToUploadDocumentResp(documentUpload), err
}

// QueryDocumentPage 查询文档分页列表
func (d *DocumentService) QueryDocumentPage(ctx context.Context, req *document.QueryDocumentPageReq) (*document.QueryDocumentPageResp, error) {
	documents, total, err := d.l.QueryDocumentPage(ctx, req.PageNo, req.PageSize, strutil.Trim(req.Keyword))
	return &document.QueryDocumentPageResp{
		PageNo:   req.PageNo,
		PageSize: req.PageSize,
		Total:    total,
		Records:  convert.ToDocumentListItemList(documents),
	}, err
}

// QueryDocumentDetail 查询文档详情
func (d *DocumentService) QueryDocumentDetail(ctx context.Context, req *document.QueryDocumentDetailReq) (*document.DocumentListItem, error) {
	detail, err := d.l.QueryDocumentDetail(ctx, req.DocumentId)
	return convert.ToDocumentListItem(detail), err
}

// DeleteDocument 删除文档
func (d *DocumentService) DeleteDocument(ctx context.Context, req *document.DeleteDocumentReq) (*document.DeleteDocumentResp, error) {
	// TODO implement me
	panic("implement me")
}

func (d *DocumentService) QueryStrategyPlan(ctx context.Context, req *document.QueryStrategyPlanReq) (*document.QueryStrategyPlanResp, error) {
	doc, plan, err := d.l.QueryStrategyPlan(ctx, req.DocumentId)
	if err != nil {
		return nil, err
	}
	resp := convert.ToQueryStrategyPlanResp(doc)
	resp.Plan = convert.ToDocumentStrategyPlan(plan)
	return resp, nil
}

func (d *DocumentService) ConfirmStrategy(ctx context.Context, req *document.ConfirmStrategyReq) (*document.ConfirmStrategyResp, error) {
	// TODO implement me
	panic("implement me")
}

// BuildIndex 构建索引
func (d *DocumentService) BuildIndex(ctx context.Context, req *document.BuildIndexReq) (*document.BuildIndexResp, error) {
	resp, err := d.l.BuildIndex(ctx, req.DocumentId, req.PlanId, req.OperatorId)
	return convert.ToBuildIndexResp(resp), err
}

func (d *DocumentService) QueryDocumentChunks(ctx context.Context, req *document.QueryDocumentChunksReq) (*document.QueryDocumentChunksResp, error) {
	// TODO implement me
	panic("implement me")
}

func (d *DocumentService) QueryDocumentChunkDetail(ctx context.Context, req *document.QueryDocumentChunkDetailReq) (*document.DocumentChunk, error) {
	// TODO implement me
	panic("implement me")
}

func (d *DocumentService) QueryTaskLogs(ctx context.Context, req *document.QueryTaskLogsReq) (*document.QueryTaskLogsResp, error) {
	// TODO implement me
	panic("implement me")
}
