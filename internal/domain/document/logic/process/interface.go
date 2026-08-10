package process

import (
	"context"

	"github.com/swiftbit/know-agent/internal/domain/document/logic/process/transform"
	"github.com/swiftbit/know-agent/internal/domain/document/model/aggregate"
	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/document/model/enum"
	"github.com/swiftbit/know-agent/internal/domain/document/model/vo"
)

// AsyncProcessor 异步处理器
type AsyncProcessor interface {
	// HandleParseRoute 处理解析路由任务
	HandleParseRoute(ctx context.Context, documentId, taskId int64) error

	// HandleIndexBuild 处理索引构建任务
	HandleIndexBuild(ctx context.Context, documentId, taskId, planId int64) error
}

// TextPreprocessor 文本预处理器
type TextPreprocessor interface {
	// Process 文本预处理
	Process(ctx context.Context, documentTitle, rawText, fileType string, opts ...transform.TransformerOption) (*aggregate.AnalysisResult, error)
}

// ProfileGenerator 文档画像生成器
type ProfileGenerator interface {
	Generate(ctx context.Context, documentId int64, analysisResult *aggregate.AnalysisResult, structureNodes []*entity.StructureNode) (*entity.DocumentProfile, error)
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
	FinalizeOuterDisposition(buildResult *vo.GraphRagBuildResult, typedOutcome enum.ComponentOutcome, observationOutcome enum.ObservationProjectionOutcome) *vo.GraphRagBuildResult

	// WithCrossDocumentOutcome 设置跨文档结果
	WithCrossDocumentOutcome(buildResult *vo.GraphRagBuildResult, outcome enum.ComponentOutcome) *vo.GraphRagBuildResult
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
