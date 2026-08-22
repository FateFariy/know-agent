package analysis

import (
	"context"
	"time"

	"github.com/swiftbit/know-agent/internal/domain/document/logic/process/analysis/save"
	"github.com/swiftbit/know-agent/internal/domain/document/model/aggregate"
	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/document/model/vo"
)

// Context 贯穿整个解析路由流程的上下文对象
type Context struct {
	DocumentId        int64
	TaskId            int64
	PlanId            int64
	Document          *entity.Document
	Task              *entity.DocumentTask
	StartTime         time.Time
	RawFileBytes      []byte
	AnalysisResult    *aggregate.AnalysisResult
	StrategyPlanDraft *vo.DocumentStrategyPlanDraft
	SaveCtx           *save.Context
}

// Stage 解析路由阶段接口
type Stage interface {
	// Name 阶段名称
	Name() string
	// Execute 执行阶段
	Execute(ctx context.Context, parseCtx *Context) error
}

// StrategyRecommender 策略推荐器接口
type StrategyRecommender interface {
	Recommend(ctx context.Context, document *entity.Document, analysisResult *aggregate.AnalysisResult) (*vo.DocumentStrategyPlanDraft, error)
}

type IndexingConfigResolver interface {
	Resolve(ctx context.Context, document *entity.Document) *vo.IndexingOptions
}
