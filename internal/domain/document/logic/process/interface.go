package process

import (
	"context"

	"github.com/swiftbit/know-agent/internal/domain/document/logic/process/transform"
	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/document/model/vo"
)

// AsyncProcessor 异步处理器
type AsyncProcessor interface {
	// HandleParseRoute 处理解析路由任务
	HandleParseRoute(ctx context.Context, documentId, taskId int64) error

	// HandleIndexBuild 处理索引构建任务
	HandleIndexBuild(ctx context.Context, documentId, taskId, planId int64) error
}

// ChunkCoordinator 分块协调器
type ChunkCoordinator interface {
	// RecommendStrategy 推荐策略方案
	RecommendStrategy(ctx context.Context, document *entity.Document, analysisResult *vo.DocumentAnalysisResult) (*vo.DocumentStrategyPlanDraft, error)

	// NormalizeSteps 标准化策略步骤
	NormalizeSteps(ctx context.Context, baseSteps []*entity.DocumentStrategyStep,
		parentStrategyTypes []int, childStrategyTypes []int, documentId int64) ([]*entity.DocumentStrategyStep, error)

	// BuildParentBlocks 构建父子块结构
	BuildParentBlocks(ctx context.Context, document *entity.Document,
		steps []*entity.DocumentStrategyStep, parsedText string) ([]*vo.ParentBlockCandidate, error)
}

// TextPreprocessor 文本预处理器
type TextPreprocessor interface {
	// Process 文本预处理
	Process(ctx context.Context, documentTitle, rawText, fileType string, opts ...transform.TransformerOption) (*vo.DocumentAnalysisResult, error)
}

// StructureNodeManager 结构节点管理器
type StructureNodeManager interface {
	// ReplaceDocumentNodes 替换文档结构节点：先按文档ID删除，再按候选节点批量插入
	ReplaceDocumentNodes(ctx context.Context, documentId, parseTaskId int64, candidates []*vo.DocumentStructureNodeCandidate) ([]*entity.DocumentStructureNode, error)

	// ListDocumentNodes 查询文档结构节点列表
	ListDocumentNodes(ctx context.Context, documentId, parseTaskId int64) ([]*entity.DocumentStructureNode, error)

	// DeleteByDocumentId 按文档ID删除所有结构节点
	DeleteByDocumentId(ctx context.Context, documentId int64) error
}

type ProfileGenerator interface {
	Generate(ctx context.Context, documentId int64, analysisResult *vo.DocumentAnalysisResult, structureNodes []*entity.DocumentStructureNode) (*entity.DocumentProfile, error)
}
