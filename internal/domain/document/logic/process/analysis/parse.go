package analysis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/swiftbit/know-agent/internal/domain/document/adapter"
	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/document/model/enum"
)

// ParsePhase 解析阶段：调用文本预处理逻辑解析文件内容
type ParsePhase struct {
	processor TextProcessor
	repo      adapter.DocumentRepository
	port      *adapter.DocumentPort
}

func NewParsePhase(processor TextProcessor, repo adapter.DocumentRepository, port *adapter.DocumentPort) *ParsePhase {
	return &ParsePhase{
		processor: processor,
		repo:      repo,
		port:      port}
}

func (p *ParsePhase) Name() string {
	return "解析阶段"
}

func (p *ParsePhase) Execute(ctx context.Context, parseCtx *Context) error {
	startTime := time.Now()

	analysisResult, err := p.processor.Process(ctx, parseCtx.Document.OriginalFileName,
		string(parseCtx.RawFileBytes), enum.FileTypeName(parseCtx.Document.FileType))
	if err != nil {
		return err
	}
	parseCtx.AnalysisResult = analysisResult
	structureCandidateCount := len(analysisResult.StructureNodes)
	// 记录"解析器返回结果"日志
	parserResultDetail, _ := json.Marshal(map[string]any{
		"parserProviderName":      analysisResult.ParserProviderName,
		"parserProviderVersion":   analysisResult.ParserProviderVersion,
		"parserTraceMetadata":     analysisResult.ParserTraceMetadata,
		"artifactCount":           len(analysisResult.ParseArtifacts),
		"blockCount":              len(analysisResult.Blocks),
		"tableCandidateCount":     len(analysisResult.TableCandidates),
		"structureCandidateCount": structureCandidateCount,
		"parserCostMillis":        time.Since(startTime).Milliseconds(),
	})
	parserResultLog := &entity.DocumentTaskLog{
		TaskId:       parseCtx.TaskId,
		DocumentId:   parseCtx.DocumentId,
		StageType:    enum.TaskStageContentParse,
		EventType:    enum.TaskEventComplete,
		LogLevel:     enum.LogLevelInfo,
		OperatorType: enum.OperatorTypeSystem,
		Content:      fmt.Sprintf("解析器返回结果，结构候选 %d 个。", structureCandidateCount),
		DetailJson:   string(parserResultDetail),
	}
	return p.repo.InsertTaskLog(ctx, parserResultLog)
}
