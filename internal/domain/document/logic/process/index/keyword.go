package index

import (
	"context"
	"encoding/json"
	"time"

	"github.com/swiftbit/know-agent/common/logx"
	"github.com/swiftbit/know-agent/internal/domain/document/adapter"
	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/document/model/enum"
)

// KeywordIndexPhase 关键词索引阶段
type KeywordIndexPhase struct {
	repo adapter.DocumentRepository
	port *adapter.DocumentPort
}

func NewKeywordIndexPhase(repo adapter.DocumentRepository, port *adapter.DocumentPort) *KeywordIndexPhase {
	return &KeywordIndexPhase{
		repo: repo,
		port: port,
	}
}

func (p *KeywordIndexPhase) Name() string {
	return "关键词索引阶段"
}

func (p *KeywordIndexPhase) Execute(ctx context.Context, buildCtx *Context) error {
	if buildCtx.ResumeCommittedGraph {
		logx.Infof("GraphRAG 已提交，跳过构建关键词，documentId=%d, taskId=%d", buildCtx.DocumentId, buildCtx.TaskId)
		return nil
	}
	// 标记开始关键词索引
	task := &entity.DocumentTask{
		ID:           buildCtx.TaskId,
		CurrentStage: enum.TaskStageKeywordIndex,
	}
	if err := p.repo.UpdateTaskById(ctx, task); err != nil {
		return err
	}
	vectorSize := len(buildCtx.ChildChunks)
	keywordStartDetail, _ := json.Marshal(map[string]any{
		"chunkCount": vectorSize,
	})
	keywordStartLog := &entity.DocumentTaskLog{
		TaskId:       buildCtx.TaskId,
		DocumentId:   buildCtx.DocumentId,
		StageType:    enum.TaskStageKeywordIndex,
		EventType:    enum.TaskEventStart,
		LogLevel:     enum.LogLevelInfo,
		OperatorType: enum.OperatorTypeSystem,
		Content:      "开始构建关键词索引",
		DetailJson:   string(keywordStartDetail),
	}
	_ = p.repo.InsertTaskLog(ctx, keywordStartLog)

	// 执行关键词索引构建
	keywordStartedTime := time.Now()
	if err := p.port.BuildIndexes(ctx, buildCtx.ChildChunks); err != nil {
		return err
	}

	// 标记完成
	keywordEndDetail, _ := json.Marshal(map[string]any{
		"chunkCount": vectorSize,
		"costMillis": time.Since(keywordStartedTime).Milliseconds(),
	})
	keywordEndLog := &entity.DocumentTaskLog{
		TaskId: buildCtx.TaskId, DocumentId: buildCtx.DocumentId,
		StageType:    enum.TaskStageKeywordIndex,
		EventType:    enum.TaskEventComplete,
		LogLevel:     enum.LogLevelInfo,
		OperatorType: enum.OperatorTypeSystem,
		Content:      "关键词索引完成",
		DetailJson:   string(keywordEndDetail),
	}
	_ = p.repo.InsertTaskLog(ctx, keywordEndLog)

	return nil
}
