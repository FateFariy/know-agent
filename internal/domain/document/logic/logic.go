package logic

import (
	"context"
	"mime/multipart"

	"github.com/swiftbit/know-agent/internal/domain/document/model/aggregate"
	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/document/model/vo"
)

// LifecycleLogic 生命周期逻辑接口
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
	ConfirmStrategy(ctx context.Context, cmd *vo.DocumentStrategyConfirmCmd) (*entity.DocumentStrategyPlan, *entity.Document, error)

	// BuildIndex 构建索引
	BuildIndex(ctx context.Context, documentId, planId, operatorId int64) (*vo.DocumentIndexBuild, error)

	// QueryDocumentChunks 查询文档块
	QueryDocumentChunks(ctx context.Context, documentId, taskId int64, pageNo, pageSize int) ([]*entity.DocumentChunk, int64, int64, error)

	// QueryDocumentChunkDetail 查询文档块详情
	QueryDocumentChunkDetail(ctx context.Context, documentId, taskId, chunkId int64) (*aggregate.DocumentChunkDetail, error)

	// QueryTaskLogs 查询任务日志
	QueryTaskLogs(ctx context.Context, taskId int64, pageNo, pageSize int) (*entity.DocumentTask, int64, error)

	// ListRetrievableDocuments 获取可检索的文档列表
	ListRetrievableDocuments(ctx context.Context, documentIds ...int64) ([]*vo.KnowledgeDocument, error)

	// QueryParentBlocks 查询父块列表
	QueryParentBlocks(ctx context.Context, parentIds []int64) ([]*entity.DocumentParentChunk, error)
}

// ProfileLogic 文档画像逻辑接口
type ProfileLogic interface {
	// GetAllProfiles 获取所有画像
	GetAllProfiles(ctx context.Context) ([]*entity.DocumentProfile, error)

	// GetProfileByDocumentId 根据文档ID获取画像
	GetProfileByDocumentId(ctx context.Context, documentId int64) (*entity.DocumentProfile, error)

	// RegenerateProfile 重新生成文档画像（读取已解析文本与结构节点）
	RegenerateProfile(ctx context.Context, documentId int64) (*entity.DocumentProfile, error)

	// BatchRegenerateProfiles 批量重新生成文档画像
	BatchRegenerateProfiles(ctx context.Context, documentIds []int64) ([]*entity.DocumentProfile, error)
}
