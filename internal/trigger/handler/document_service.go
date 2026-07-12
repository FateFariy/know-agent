package handler

import (
	"context"
	"mime/multipart"

	"github.com/duke-git/lancet/v2/strutil"

	"github.com/swiftbit/know-agent/api/document"
	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/convert"
	"github.com/swiftbit/know-agent/internal/domain/document/logic"
)

type DocumentService struct {
	lifeCycleLogic logic.LifecycleLogic
	profileLogic   logic.ProfileLogic
}

var _ document.HTTPServer = (*DocumentService)(nil)

func NewDocumentService(lifeCycleLogic logic.LifecycleLogic, profileLogic logic.ProfileLogic) *DocumentService {
	return &DocumentService{
		lifeCycleLogic: lifeCycleLogic,
		profileLogic:   profileLogic,
	}
}

// UploadDocument 上传文档
func (d *DocumentService) UploadDocument(ctx context.Context, file multipart.File, header *multipart.FileHeader, req *document.UploadDocumentReq) (*document.UploadDocumentResp, error) {
	doc := convert.FromUploadDocumentReq(req)
	documentUpload, err := d.lifeCycleLogic.Upload(ctx, file, header, doc)
	return convert.ToUploadDocumentResp(documentUpload), err
}

// QueryDocumentPage 查询文档分页列表
func (d *DocumentService) QueryDocumentPage(ctx context.Context, req *document.QueryDocumentPageReq) (*document.QueryDocumentPageResp, error) {
	documents, total, err := d.lifeCycleLogic.QueryDocumentPage(ctx, req.PageNo, req.PageSize, strutil.Trim(req.Keyword))
	return &document.QueryDocumentPageResp{
		PageNo:   req.PageNo,
		PageSize: req.PageSize,
		Total:    total,
		Records:  convert.ToDocumentListItemList(documents),
	}, err
}

// QueryDocumentDetail 查询文档详情
func (d *DocumentService) QueryDocumentDetail(ctx context.Context, req *document.QueryDocumentDetailReq) (*document.DocumentListItem, error) {
	detail, err := d.lifeCycleLogic.QueryDocumentDetail(ctx, req.DocumentId)
	return convert.ToDocumentListItem(detail), err
}

// DeleteDocument 删除文档
func (d *DocumentService) DeleteDocument(ctx context.Context, req *document.DeleteDocumentReq) (*document.DeleteDocumentResp, error) {
	documentName, err := d.lifeCycleLogic.DeleteDocument(ctx, req.DocumentId)
	return &document.DeleteDocumentResp{DocumentId: req.DocumentId, DocumentName: documentName}, err
}

// QueryStrategyPlan 查询策略计划
func (d *DocumentService) QueryStrategyPlan(ctx context.Context, req *document.QueryStrategyPlanReq) (*document.QueryStrategyPlanResp, error) {
	doc, plan, err := d.lifeCycleLogic.QueryStrategyPlan(ctx, req.DocumentId)
	if err != nil {
		return nil, err
	}
	resp := convert.ToQueryStrategyPlanResp(doc)
	resp.Plan = convert.ToDocumentStrategyPlan(plan)
	return resp, nil
}

func (d *DocumentService) ConfirmStrategy(ctx context.Context, req *document.ConfirmStrategyReq) (*document.ConfirmStrategyResp, error) {
	plan, doc, err := d.lifeCycleLogic.ConfirmStrategy(ctx, convert.FromConfirmStrategyReq(req))
	if err != nil {
		return nil, err
	}
	resp := convert.ToConfirmStrategyResp(plan)
	resp.StrategyStatus = doc.StrategyStatus
	resp.StrategyStatusName = doc.StrategyStatusName
	return resp, nil
}

// BuildIndex 构建索引
func (d *DocumentService) BuildIndex(ctx context.Context, req *document.BuildIndexReq) (*document.BuildIndexResp, error) {
	resp, err := d.lifeCycleLogic.BuildIndex(ctx, req.DocumentId, req.PlanId, req.OperatorId)
	return convert.ToBuildIndexResp(resp), err
}

// QueryDocumentChunks 查询文档块列表
func (d *DocumentService) QueryDocumentChunks(ctx context.Context, req *document.QueryDocumentChunksReq) (*document.QueryDocumentChunksResp, error) {
	chunks, total, planId, err := d.lifeCycleLogic.QueryDocumentChunks(ctx, req.DocumentId, req.TaskId, req.PageNo, req.PageSize)
	return &document.QueryDocumentChunksResp{
		DocumentId: req.DocumentId,
		PageNo:     req.PageNo,
		PageSize:   req.PageSize,
		PlanId:     planId,
		Records:    convert.ToDocumentChunkItemList(chunks),
		TaskId:     utils.Ternary(total > 0, chunks[0].TaskId, 0),
		Total:      total,
	}, err
}

// QueryDocumentChunkDetail 查询文档块详情
func (d *DocumentService) QueryDocumentChunkDetail(ctx context.Context, req *document.QueryDocumentChunkDetailReq) (*document.QueryDocumentChunkDetailResp, error) {
	detail, err := d.lifeCycleLogic.QueryDocumentChunkDetail(ctx, req.DocumentId, req.TaskId, req.ChunkId)
	return convert.ToQueryDocumentChunkDetailResp(detail), err
}

// QueryTaskLogs 查询任务日志
func (d *DocumentService) QueryTaskLogs(ctx context.Context, req *document.QueryTaskLogsReq) (*document.QueryTaskLogsResp, error) {
	task, total, err := d.lifeCycleLogic.QueryTaskLogs(ctx, req.TaskId, req.PageNo, req.PageSize)
	if err != nil {
		return nil, err
	}
	queryTaskLogsResp := convert.ToQueryTaskLogsResp(task)
	queryTaskLogsResp.Total = total
	return queryTaskLogsResp, nil
}

// GetDocumentProfile 查询文档画像详情
func (d *DocumentService) GetDocumentProfile(ctx context.Context, req *document.DocumentProfileDetailReq) (*document.DocumentProfileResp, error) {
	profile, err := d.profileLogic.GetProfileByDocumentId(ctx, req.DocumentId)
	if err != nil {
		return nil, err
	}
	return convert.ToDocumentProfileResp(profile), nil
}

// RegenerateDocumentProfile 重新生成文档画像
func (d *DocumentService) RegenerateDocumentProfile(ctx context.Context, req *document.DocumentProfileRegenerateReq) (*document.DocumentProfileResp, error) {
	profile, err := d.profileLogic.RegenerateProfile(ctx, req.DocumentId)
	if err != nil {
		return nil, err
	}
	return convert.ToDocumentProfileResp(profile), nil
}

// BatchRegenerateDocumentProfile 批量重新生成文档画像
func (d *DocumentService) BatchRegenerateDocumentProfile(ctx context.Context, req *document.DocumentProfileBatchRegenerateReq) ([]*document.DocumentProfileResp, error) {
	profiles, err := d.profileLogic.BatchRegenerateProfiles(ctx, req.DocumentIds)
	if err != nil {
		return nil, err
	}
	return convert.ToDocumentProfileRespList(profiles), nil
}
