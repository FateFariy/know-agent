package analysis

import (
	"context"
	"time"

	"github.com/swiftbit/know-agent/internal/domain/document/adapter"
	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/document/model/vo"
)

// Context 贯穿整个解析路由流程的上下文对象
type Context struct {
	// 基础参数
	DocumentID int64
	TaskID     int64

	// 实体对象
	Document *entity.Document
	Task     *entity.DocumentTask

	// 业务数据
	StartTime         time.Time
	RawFileBytes      []byte
	AnalysisResult    *vo.DocumentAnalysisResult
	ParsedTextPath    string
	StructureNodes    []*entity.DocumentStructureNode
	StrategyPlanDraft *vo.DocumentStrategyPlanDraft
	PlanID            int64

	// 统计耗时
	ParserCostMillis           int64
	ArtifactPersistCostMillis  int64
	StructurePersistCostMillis int64
	NavigationCostMillis       int64
	ProfileCostMillis          int64
	StrategyPersistCostMillis  int64
}

// Phase 解析路由阶段接口
type Phase interface {
	// Name 阶段名称
	Name() string
	// Execute 执行阶段
	Execute(ctx context.Context, parseCtx *Context) error
}

// PhaseDeps 解析路由阶段依赖项
type PhaseDeps struct {
	Repo adapter.DocumentRepository
	Port *adapter.DocumentPort
	//NodeManager process.StructureNodeManager
	//Gen         process.ProfileGenerator
	//Coordinator process.ChunkCoordinator
}

// StrategyRecommender 策略推荐器接口
type StrategyRecommender interface {
	Recommend(ctx context.Context, document *entity.Document, analysisResult *vo.DocumentAnalysisResult) (*vo.DocumentStrategyPlanDraft, error)
}
