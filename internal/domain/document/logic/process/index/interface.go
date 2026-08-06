package index

import (
	"context"
	"time"

	"github.com/swiftbit/know-agent/internal/domain/document/adapter"
	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/document/model/vo"
)

// BuildContext 贯穿整个构建流程的上下文对象
type BuildContext struct {
	// 基础参数
	DocumentID int64
	TaskID     int64
	PlanID     int64

	// 实体对象
	Document *entity.Document
	Task     *entity.DocumentTask
	Plan     *entity.DocumentStrategyPlan

	// 业务数据
	StartTime            time.Time
	BuildStartedTime     time.Time
	PipelineSteps        []*entity.DocumentStrategyStep
	ParentCandidates     []*vo.ParentBlockCandidate
	ChildChunks          []*entity.DocumentChunk
	ParentBlocks         []*entity.DocumentParentBlock
	GraphRagBuildResult  *vo.GraphRagBuildResult
	GraphFinalization    *vo.GraphRagFinalization
	RaptorBuildResult    *vo.RaptorBuildResult
	GraphTypedChunkList  []vo.TypedChunk
	ResumeCommittedGraph bool

	// 统计耗时
	ChunkCostMillis    int64
	ProcessCostMillis  int64
	VectorCostMillis   int64
	KeywordCostMillis  int64
	GraphRagCostMillis int64
	RaptorCostMillis   int64
	TotalCostMillis    int64
}

// BuildPhase 索引构建阶段接口
type BuildPhase interface {
	// Name 阶段名称
	Name() string
	// Execute 执行阶段
	Execute(ctx context.Context, buildCtx *BuildContext) error
}

// PhaseDeps 阶段依赖项
type PhaseDeps struct {
	Repo adapter.DocumentRepository
	Port *adapter.DocumentPort
	//Coordinator             process.ChunkCoordinator
	//GraphRagBuilder         process.GraphRagBuilder
	//GraphRagOutcomePolicy   process.GraphRagOutcomePolicy
	//GraphRagBuildCheckpoint process.GraphRagBuildCheckpoint
	//CrossDocumentIndexer    process.CrossDocumentIndexer
	//RaptorBuilder           process.RaptorBuilder
}
