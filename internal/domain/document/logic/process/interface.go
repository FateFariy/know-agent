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
	// Recommend 推荐策略方案
	Recommend(ctx context.Context, document *entity.Document, analysisResult *vo.AnalysisResult) (*vo.DocumentStrategyPlanDraft, error)

	// NormalizeSteps 标准化策略步骤
	NormalizeSteps(ctx context.Context, baseSteps []*entity.DocumentStrategyStep,
		parentStrategyTypes []int, childStrategyTypes []int, documentId int64) ([]*entity.DocumentStrategyStep, error)

	// BuildParentBlocks 构建父子块结构
	BuildParentBlocks(ctx context.Context, document *entity.Document,
		steps []*entity.DocumentStrategyStep, parsedText string) ([]*vo.ParentChunkCandidate, error)
}

// TextPreprocessor 文本预处理器
type TextPreprocessor interface {
	// Process 文本预处理
	Process(ctx context.Context, documentTitle, rawText, fileType string, opts ...transform.TransformerOption) (*vo.AnalysisResult, error)
}

// StructureNodeManager 结构节点管理器
type StructureNodeManager interface {
	// ListDocumentNodes 查询文档结构节点列表
	ListDocumentNodes(ctx context.Context, documentId, parseTaskId int64) ([]*entity.StructureNode, error)

	// DeleteByDocumentId 按文档ID删除所有结构节点
	DeleteByDocumentId(ctx context.Context, documentId int64) error
}

// ProfileGenerator 文档画像生成器
type ProfileGenerator interface {
	Generate(ctx context.Context, documentId int64, analysisResult *vo.AnalysisResult, structureNodes []*entity.StructureNode) (*entity.DocumentProfile, error)
}

// GraphRagBuilder GraphRAG 图谱构建器
type GraphRagBuilder interface {
	// RebuildDocumentGraph 重建文档图谱
	RebuildDocumentGraph(ctx context.Context, documentId, taskId int64, chunks []*entity.DocumentChunk) (*vo.GraphRagBuildResult, error)

	// ReplaceTypedIndex 替换类型化索引
	ReplaceTypedIndex(ctx context.Context, documentId, taskId, planId int64, chunks []*entity.DocumentChunk, nextChunkNo int) ([]*entity.DocumentChunk, error)
}

// GraphRagOutcomePolicy GraphRAG 结果策略
type GraphRagOutcomePolicy interface {
	// FinalizeOuterDisposition 最终化外部处置
	FinalizeOuterDisposition(buildResult *vo.GraphRagBuildResult, typedOutcome vo.ComponentOutcome, observationOutcome vo.ObservationProjectionOutcome) *vo.GraphRagBuildResult

	// WithCrossDocumentOutcome 设置跨文档结果
	WithCrossDocumentOutcome(buildResult *vo.GraphRagBuildResult, outcome vo.ComponentOutcome) *vo.GraphRagBuildResult
}

// GraphRagBuildCheckpoint GraphRAG 构建检查点
type GraphRagBuildCheckpoint interface {
	// MarkOutcome 标记结果
	MarkOutcome(ctx context.Context, documentId, taskId int64, result *vo.GraphRagBuildResult, attempt, maxAttempts int) error
}

// CrossDocumentIndexer 跨文档索引器
type CrossDocumentIndexer interface {
	// RebuildAll 重建所有跨文档索引
	RebuildAll(ctx context.Context, documentId, taskId int64) error
}

// RaptorBuilder RAPTOR 层级摘要树构建器
type RaptorBuilder interface {
	// RebuildDocumentTree 重建文档树
	RebuildDocumentTree(ctx context.Context, documentId, taskId int64, chunks []*entity.DocumentChunk) (*vo.RaptorBuildResult, error)
}
