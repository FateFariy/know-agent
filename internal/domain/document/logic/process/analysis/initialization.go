package analysis

import (
	"context"
	"encoding/json"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/document/adapter"
	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/document/model/enum"
)

// InitializationPhase 初始化阶段：标记任务运行中、文档解析中、写入开始日志
type InitializationPhase struct {
	repo adapter.DocumentRepository
}

func NewInitializationPhase(repo adapter.DocumentRepository) *InitializationPhase {
	return &InitializationPhase{repo: repo}
}

func (p *InitializationPhase) Name() string {
	return "初始化阶段"
}

func (p *InitializationPhase) Order() int {
	return 0
}

func (p *InitializationPhase) Execute(ctx context.Context, parseCtx *Context) error {
	// 事务性标记任务运行中 + 文档解析中，并写入"开始解析"日志
	markParseStartTx := func(txCtx context.Context) error {
		// 标记任务运行中
		runningTask := &entity.DocumentTask{
			ID:           parseCtx.TaskId,
			TaskStatus:   enum.TaskStatusRunning,
			CurrentStage: enum.TaskStageContentParse,
			StartTime:    utils.Pointer(parseCtx.StartTime),
		}
		if err := p.repo.UpdateTaskById(txCtx, runningTask); err != nil {
			return err
		}

		// 标记文档解析中
		document := &entity.Document{
			ID:          parseCtx.DocumentId,
			ParseStatus: enum.ParseStatusParsing,
		}
		if err := p.repo.UpdateDocumentById(txCtx, document); err != nil {
			return err
		}

		// 写入"开始解析文档内容"日志
		detail, _ := json.Marshal(map[string]any{
			"objectName": parseCtx.Document.ObjectName,
			"fileType":   enum.FileTypeName(parseCtx.Document.FileType),
			"fileName":   parseCtx.Document.OriginalFileName,
		})
		startLog := &entity.DocumentTaskLog{
			TaskId:       parseCtx.TaskId,
			DocumentId:   parseCtx.DocumentId,
			StageType:    enum.TaskStageContentParse,
			EventType:    enum.TaskEventStart,
			LogLevel:     enum.LogLevelInfo,
			OperatorType: enum.OperatorTypeSystem,
			Content:      "开始解析文档内容，文档类型：" + enum.FileTypeName(parseCtx.Document.FileType),
			DetailJson:   string(detail),
		}
		_ = p.repo.InsertTaskLog(txCtx, startLog)
		return nil
	}
	return p.repo.Do(ctx, markParseStartTx)
}
