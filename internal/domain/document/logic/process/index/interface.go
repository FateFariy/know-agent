package index

import (
	"context"
	"time"

	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/document/model/vo"
)

// Context 贯穿整个构建流程的上下文对象
type Context struct {
	DocumentId           int64
	TaskId               int64
	PlanId               int64
	Document             *entity.Document
	Task                 *entity.DocumentTask
	Plan                 *entity.DocumentStrategyPlan
	StartTime            time.Time
	BuildStartedTime     time.Time
	PipelineSteps        vo.DocumentStrategyStepDrafts
	ParentCandidates     vo.ParentChunkCandidates
	ChildChunks          []*entity.DocumentChunk
	ParentBlocks         []*entity.DocumentParentChunk
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
	Execute(ctx context.Context, buildCtx *Context) error
}

type IndexingConfigResolver interface {
	Resolve(ctx context.Context, document *entity.Document) *vo.IndexingOptions
}
