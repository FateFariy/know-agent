package analysis

import (
	"context"
	"time"

	"github.com/swiftbit/know-agent/internal/domain/document/logic/process/analysis/save"
	"github.com/swiftbit/know-agent/internal/domain/document/logic/process/transform"
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
	AnalysisResult    *vo.AnalysisResult
	StrategyPlanDraft *vo.DocumentStrategyPlanDraft
	SaveCtx           *save.Context
}

// Phase 解析路由阶段接口
type Phase interface {
	// Name 阶段名称
	Name() string
	// Execute 执行阶段
	Execute(ctx context.Context, parseCtx *Context) error
}

// StrategyRecommender 策略推荐器接口
type StrategyRecommender interface {
	Recommend(ctx context.Context, document *entity.Document, analysisResult *vo.AnalysisResult) (*vo.DocumentStrategyPlanDraft, error)
}

// TextProcessor 文本处理器
type TextProcessor interface {
	// Process 文本预处理
	Process(ctx context.Context, documentTitle, rawText, fileType string, opts ...transform.TransformerOption) (*vo.AnalysisResult, error)
}
