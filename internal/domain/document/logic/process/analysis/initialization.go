package analysis

import (
	"context"
	"encoding/json"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/document/adapter"
	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/document/model/vo"
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

func (p *InitializationPhase) Execute(ctx context.Context, parseCtx *Context) error {
	// 事务性标记任务运行中 + 文档解析中，并写入"开始解析"日志
	markParseStartTx := func(txCtx context.Context) error {
		// 标记任务运行中
		runningTask := &entity.DocumentTask{
			ID:           parseCtx.TaskID,
			TaskStatus:   vo.TaskStatusRunning,
			CurrentStage: vo.TaskStageContentParse,
			StartTime:    utils.Pointer(parseCtx.StartTime),
		}
		if err := p.repo.UpdateTaskById(txCtx, runningTask); err != nil {
			return err
		}

		// 标记文档解析中
		document := &entity.Document{
			ID:          parseCtx.DocumentID,
			ParseStatus: vo.ParseStatusParsing,
		}
		if err := p.repo.UpdateDocumentById(txCtx, document); err != nil {
			return err
		}

		// 写入"开始解析文档内容"日志
		detail, _ := json.Marshal(map[string]any{
			"objectName": parseCtx.Document.ObjectName,
			"fileType":   vo.FileTypeName(parseCtx.Document.FileType),
			"fileName":   parseCtx.Document.OriginalFileName,
		})
		startLog := &entity.DocumentTaskLog{
			TaskId:       parseCtx.TaskID,
			DocumentId:   parseCtx.DocumentID,
			StageType:    vo.TaskStageContentParse,
			EventType:    vo.TaskEventStart,
			LogLevel:     vo.LogLevelInfo,
			OperatorType: vo.OperatorTypeSystem,
			Content:      "开始解析文档内容，解析方式已确定：" + vo.FileTypeName(parseCtx.Document.FileType),
			DetailJson:   string(detail),
		}
		return p.repo.InsertTaskLog(txCtx, startLog)
	}
	return p.repo.Do(ctx, markParseStartTx)
}
