package handler

import (
	"context"
	"mime/multipart"
	"strings"

	"github.com/swiftbit/know-agent/api/document"
	"github.com/swiftbit/know-agent/internal/convert"
	"github.com/swiftbit/know-agent/internal/domain/document/logic"
)

type DocumentService struct {
	l logic.DocumentLifecycleLogic
}

var _ document.HTTPServer = (*DocumentService)(nil)

func NewDocumentService(l logic.DocumentLifecycleLogic) *DocumentService {
	return &DocumentService{
		l: l,
	}
}

func (d *DocumentService) UploadDocument(ctx context.Context, file multipart.File, header *multipart.FileHeader, req *document.UploadDocumentReq) (*document.UploadDocumentResp, error) {
	// TODO implement me
	panic("implement me")
}

func (d *DocumentService) QueryDocumentPage(ctx context.Context, req *document.QueryDocumentPageReq) (*document.QueryDocumentPageResp, error) {
	keyword := strings.TrimSpace(req.Keyword)
	documents, total, err := d.l.QueryDocumentPage(ctx, req.PageNo, req.PageSize, keyword)
	return &document.QueryDocumentPageResp{
		PageNo:   req.PageNo,
		PageSize: req.PageSize,
		Total:    total,
		Records:  convert.ToDocumentListItem(documents),
	}, err
}

func (d *DocumentService) QueryDocumentDetail(ctx context.Context, req *document.QueryDocumentDetailReq) (*document.DocumentListItem, error) {
	document, _, err := d.l.QueryDocumentDetail(ctx, req.DocumentId)
}

func (d *DocumentService) DeleteDocument(ctx context.Context, req *document.DeleteDocumentReq) (*document.DeleteDocumentResp, error) {
	// TODO implement me
	panic("implement me")
}

func (d *DocumentService) QueryStrategyPlan(ctx context.Context, req *document.QueryStrategyPlanReq) (*document.QueryStrategyPlanResp, error) {
	// TODO implement me
	panic("implement me")
}

func (d *DocumentService) ConfirmStrategy(ctx context.Context, req *document.ConfirmStrategyReq) (*document.ConfirmStrategyResp, error) {
	// TODO implement me
	panic("implement me")
}

func (d *DocumentService) BuildIndex(ctx context.Context, req *document.BuildIndexReq) (*document.BuildIndexResp, error) {
	// TODO implement me
	panic("implement me")
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
