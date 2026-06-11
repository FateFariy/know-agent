package logic

import (
	"context"
	"mime/multipart"

	"github.com/swiftbit/know-agent/internal/domain/document/model/dto"
	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/document/model/vo"
)

// DocumentLifecycleLogic 文档生命周期业务逻辑接口
type DocumentLifecycleLogic interface {
	// Upload 上传文档
	Upload(ctx context.Context, file multipart.File, header *multipart.FileHeader, doc *entity.Document) (*vo.DocumentUploadVo, error)

	// QueryDocumentPage 分页查询文档列表
	QueryDocumentPage(ctx context.Context, req *dto.DocumentPageQueryDto) (*vo.DocumentPageQueryVo, error)

	// QueryDocumentDetail 查询文档详情
	QueryDocumentDetail(ctx context.Context, req *dto.DocumentDetailQueryDto) (*vo.DocumentListItemVo, error)

	// DeleteDocument 删除文档
	DeleteDocument(ctx context.Context, req *dto.DocumentDeleteDto) (*vo.DocumentDeleteVo, error)

	// QueryStrategyPlan 查询策略方案
	QueryStrategyPlan(ctx context.Context, req *dto.DocumentStrategyPlanQueryDto) (*vo.DocumentStrategyPlanQueryVo, error)

	// ConfirmStrategy 确认策略
	ConfirmStrategy(ctx context.Context, req *dto.DocumentStrategyConfirmDto) (*vo.DocumentStrategyConfirmVo, error)

	// BuildIndex 构建索引
	BuildIndex(ctx context.Context, req *dto.DocumentIndexBuildDto) (*vo.DocumentIndexBuildVo, error)

	// QueryTaskLogs 查询任务日志
	QueryTaskLogs(ctx context.Context, req *dto.DocumentTaskLogQueryDto) (*vo.DocumentTaskLogQueryVo, error)

	// QueryDocumentChunks 查询文档块
	QueryDocumentChunks(ctx context.Context, req *dto.DocumentChunkQueryDto) (*vo.DocumentChunkQueryVo, error)

	// QueryDocumentChunkDetail 查询文档块详情
	QueryDocumentChunkDetail(ctx context.Context, req *dto.DocumentChunkDetailQueryDto) (*vo.DocumentChunkDetailVo, error)
}
