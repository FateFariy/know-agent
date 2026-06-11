package logic

import (
	"context"
	"mime/multipart"

	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/document/model/vo"
)

// DocumentLifecycleLogic 文档生命周期业务逻辑接口
type DocumentLifecycleLogic interface {
	// Upload 上传文档
	Upload(ctx context.Context, file multipart.File, header *multipart.FileHeader, doc *entity.Document) (*vo.DocumentUpload, error)

	// QueryDocumentPage 分页查询文档列表
	QueryDocumentPage(ctx context.Context, req *vo.DocumentPageQuery) (*vo.DocumentPageVo, error)

	// QueryDocumentDetail 查询文档详情
	QueryDocumentDetail(ctx context.Context, documentId int64) (*entity.Document, *entity.DocumentTask, error)

	// DeleteDocument 删除文档
	DeleteDocument(ctx context.Context, documentId int64) (string, error)

	// QueryStrategyPlan 查询策略方案
	QueryStrategyPlan(ctx context.Context, documentId int64) (*vo.DocumentStrategyPlanQueryVo, error)

	// ConfirmStrategy 确认策略
	ConfirmStrategy(ctx context.Context, req *vo.DocumentStrategyConfirm) (*vo.DocumentStrategyConfirmVo, error)

	// BuildIndex 构建索引
	BuildIndex(ctx context.Context, req *vo.DocumentIndexBuild) (*vo.DocumentIndexBuildVo, error)

	// QueryTaskLogs 查询任务日志
	QueryTaskLogs(ctx context.Context, req *vo.DocumentTaskLogQuery) (*vo.DocumentTaskLogQueryVo, error)

	// QueryDocumentChunks 查询文档块
	QueryDocumentChunks(ctx context.Context, req *vo.DocumentChunkQuery) (*vo.DocumentChunkQueryVo, error)

	// QueryDocumentChunkDetail 查询文档块详情
	QueryDocumentChunkDetail(ctx context.Context, req *vo.DocumentChunkDetailQuery) (*vo.DocumentChunkDetailVo, error)
}
