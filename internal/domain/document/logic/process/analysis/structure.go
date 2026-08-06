package analysis

import (
	"context"
	"encoding/json"
	"time"

	"github.com/swiftbit/know-agent/common/logx"
	"github.com/swiftbit/know-agent/internal/domain/document/logic/process/parse"
	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/document/model/enum"
)

// StructurePhase 结构节点阶段：替换结构节点、同步导航产物、生成文档画像
type StructurePhase struct {
	deps *parse.PhaseDeps
}

func NewStructurePhase(deps *parse.PhaseDeps) *StructurePhase {
	return &StructurePhase{deps: deps}
}

func (p *StructurePhase) Name() string {
	return "结构节点阶段"
}

func (p *StructurePhase) Execute(ctx context.Context, parseCtx *parse.Context) error {
	// 替换文档结构节点
	structurePersistStartedNanos := time.Now()
	structureNodes, err := p.deps.NodeManager.ReplaceDocumentNodes(ctx, parseCtx.DocumentID, parseCtx.TaskID, parseCtx.AnalysisResult.StructureNodes)
	if err != nil {
		return err
	}
	parseCtx.StructureNodes = structureNodes
	parseCtx.StructurePersistCostMillis = time.Since(structurePersistStartedNanos).Milliseconds()
	logx.Infof("结构节点入库完成，documentId=%d, taskId=%d, structureNodeCount=%d, costMillis=%d",
		parseCtx.DocumentID, parseCtx.TaskID, len(structureNodes), parseCtx.StructurePersistCostMillis)

	// 同步导航产物
	navigationStartedNanos := time.Now()
	if err := p.syncNavigationArtifacts(ctx, parseCtx.DocumentID, parseCtx.TaskID, structureNodes); err != nil {
		return err
	}
	parseCtx.NavigationCostMillis = time.Since(navigationStartedNanos).Milliseconds()
	logx.Infof("导航产物同步完成，documentId=%d, taskId=%d, structureNodeCount=%d, costMillis=%d",
		parseCtx.DocumentID, parseCtx.TaskID, len(structureNodes), parseCtx.NavigationCostMillis)

	// 生成文档画像
	profileStartedNanos := time.Now()
	if _, err := p.deps.Gen.Generate(ctx, parseCtx.DocumentID, parseCtx.AnalysisResult, structureNodes); err != nil {
		return err
	}
	parseCtx.ProfileCostMillis = time.Since(profileStartedNanos).Milliseconds()
	logx.Infof("文档画像生成流程完成，documentId=%d, taskId=%d, costMillis=%d",
		parseCtx.DocumentID, parseCtx.TaskID, parseCtx.ProfileCostMillis)

	// 写入"解析产物入库完成"日志
	persistDetail, _ := json.Marshal(map[string]any{
		"parseTextPath":              parseCtx.ParsedTextPath,
		"structureNodeCount":         len(structureNodes),
		"structurePersistCostMillis": parseCtx.StructurePersistCostMillis,
		"navigationCostMillis":       parseCtx.NavigationCostMillis,
		"profileCostMillis":          parseCtx.ProfileCostMillis,
	})
	persistLog := &entity.DocumentTaskLog{
		TaskId: parseCtx.TaskID, DocumentId: parseCtx.DocumentID,
		StageType: enum.TaskStageContentParse, EventType: enum.TaskEventComplete,
		LogLevel: enum.LogLevelInfo, OperatorType: enum.OperatorTypeSystem,
		Content:    "解析产物入库完成，已保存 parsed text、structure 和文档画像。",
		DetailJson: string(persistDetail),
	}
	if err := p.deps.Repo.InsertTaskLog(ctx, persistLog); err != nil {
		return err
	}

	// 写入"文档解析完成"日志
	parseFinishDetail, _ := json.Marshal(map[string]any{
		"charCount":           parseCtx.AnalysisResult.CharCount,
		"tokenCount":          parseCtx.AnalysisResult.TokenCount,
		"structureLevel":      parseCtx.AnalysisResult.StructureLevel,
		"contentQualityLevel": parseCtx.AnalysisResult.ContentQualityLevel,
		"structureNodeCount":  len(structureNodes),
		"paragraphCount":      parseCtx.AnalysisResult.ParagraphCount,
	})
	parseFinishLog := &entity.DocumentTaskLog{
		TaskId: parseCtx.TaskID, DocumentId: parseCtx.DocumentID,
		StageType: enum.TaskStageContentParse, EventType: enum.TaskEventComplete,
		LogLevel: enum.LogLevelInfo, OperatorType: enum.OperatorTypeSystem,
		Content: "文档解析完成", DetailJson: string(parseFinishDetail),
	}
	return p.deps.Repo.InsertTaskLog(ctx, parseFinishLog)
}

// syncNavigationArtifacts 同步导航产物（占位实现）
func (p *StructurePhase) syncNavigationArtifacts(ctx context.Context, documentId, parseTaskId int64, structureNodes []*entity.DocumentStructureNode) error {
	return nil
}
