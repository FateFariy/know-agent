package logic

import (
	"context"
	"mime/multipart"

	"github.com/swiftbit/know-agent/internal/domain/document/logic/transform"
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
	ConfirmStrategy(ctx context.Context, cmd *vo.DocumentStrategyConfirmCmd) (*entity.DocumentStrategyPlan, *entity.Document, error)

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

// StructureNodeLogic 结构节点业务逻辑接口
type StructureNodeLogic interface {
	// ReplaceDocumentNodes 替换文档结构节点：先按文档ID删除，再按候选节点批量插入
	ReplaceDocumentNodes(ctx context.Context, documentId, parseTaskId int64, candidates []*vo.DocumentStructureNodeCandidate) ([]*entity.DocumentStructureNode, error)

	// ListDocumentNodes 查询文档结构节点列表
	ListDocumentNodes(ctx context.Context, documentId, parseTaskId int64) ([]*entity.DocumentStructureNode, error)

	// DeleteByDocumentId 按文档ID删除所有结构节点
	DeleteByDocumentId(ctx context.Context, documentId int64) error
}

// StrategyLogic 策略业务逻辑接口
type StrategyLogic interface {
	// RecommendStrategy 推荐策略方案
	RecommendStrategy(ctx context.Context, document *entity.Document, analysisResult *vo.DocumentAnalysisResult) (*vo.DocumentStrategyPlanDraft, error)

	// NormalizeSteps 标准化策略步骤
	NormalizeSteps(ctx context.Context, baseSteps []*entity.DocumentStrategyStep,
		parentStrategyTypes []int, childStrategyTypes []int, documentId int64) ([]*entity.DocumentStrategyStep, error)

	// BuildParentBlocks 构建父子块结构
	BuildParentBlocks(ctx context.Context, document *entity.Document,
		steps []*entity.DocumentStrategyStep, parsedText string) ([]*vo.ParentBlockCandidate, error)
}

// TextPreProcessLogic 文本预处理业务逻辑接口
type TextPreProcessLogic interface {
	// PreProcess 文本预处理
	PreProcess(ctx context.Context, documentTitle, parsedText, fileType string, opts ...transform.TransformerOption) error
}
