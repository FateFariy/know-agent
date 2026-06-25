package logic

import (
	"context"
	"mime/multipart"

	"github.com/swiftbit/know-agent/internal/domain/document/model/aggregate"
	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/document/model/vo"
)

// LifecycleLogic 生命周期业务逻辑接口
type LifecycleLogic interface {
	// Upload 上传文档
	Upload(ctx context.Context, file multipart.File, header *multipart.FileHeader, doc *entity.Document) (*vo.DocumentUpload, error)

	// QueryDocumentPage 分页查询文档列表
	QueryDocumentPage(ctx context.Context, pageNo, pageSize int, keyword string) ([]*entity.Document, int64, error)

	// QueryDocumentDetail 查询文档详情
	QueryDocumentDetail(ctx context.Context, documentId int64) (*entity.Document, error)

	// DeleteDocument 删除文档
	DeleteDocument(ctx context.Context, documentId int64) (string, error)

	// QueryStrategyPlan 查询策略方案
	QueryStrategyPlan(ctx context.Context, documentId int64) (*entity.Document, *entity.DocumentStrategyPlan, error)

	// ConfirmStrategy 确认策略
	ConfirmStrategy(ctx context.Context, cmd *vo.DocumentStrategyConfirmCmd) (*entity.DocumentStrategyPlan, error)

	// BuildIndex 构建索引
	BuildIndex(ctx context.Context, documentId, planId, operatorId int64) (*vo.DocumentIndexBuild, error)

	// QueryDocumentChunks 查询文档块
	QueryDocumentChunks(ctx context.Context, documentId, taskId int64, pageNo, pageSize int) ([]*entity.DocumentChunk, int64, int64, error)

	// QueryDocumentChunkDetail 查询文档块详情
	QueryDocumentChunkDetail(ctx context.Context, documentId, taskId, chunkId int64) (*aggregate.DocumentChunkDetail, error)

	// QueryTaskLogs 查询任务日志
	QueryTaskLogs(ctx context.Context, taskId int64, pageNo, pageSize int) (*entity.DocumentTask, int64, error)
}

// AsyncProcessingLogic 异步处理业务逻辑接口
type AsyncProcessingLogic interface {
	// HandleParseRoute 处理解析路由任务
	HandleParseRoute(ctx context.Context, documentId, taskId int64) error

	// HandleIndexBuild 处理索引构建任务
	HandleIndexBuild(ctx context.Context, documentId, taskId, planId int64) error
}

type ParserLogic interface {
	Parse(ctx context.Context, bytes []byte, originalFileName string, mimeType, fileType int) (*vo.DocumentAnalysisResult, error)
}
