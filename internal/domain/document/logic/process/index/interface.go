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
	DocumentId           int64
	TaskId               int64
	PlanId               int64
	Document             *entity.Document
	Task                 *entity.DocumentTask
	Plan                 *entity.DocumentStrategyPlan
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
}

// Phase 索引构建阶段接口
type Phase interface {
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
