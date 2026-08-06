package index

import (
	"context"
	"encoding/json"
	"time"

	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/document/model/enum"
)

// KeywordIndexPhase 关键词索引阶段
type KeywordIndexPhase struct {
	*PhaseDeps
}

func NewKeywordIndexPhase(deps *PhaseDeps) *KeywordIndexPhase {
	return &KeywordIndexPhase{PhaseDeps: deps}
}

func (p *KeywordIndexPhase) Name() string {
	return "关键词索引阶段"
}

func (p *KeywordIndexPhase) Execute(ctx context.Context, buildCtx *BuildContext) error {
	vectorSize := len(buildCtx.ChildChunks)

	// 标记开始关键词索引
	markKeywordIndexTx := func(txCtx context.Context) error {
		if err := p.Repo.UpdateTaskById(txCtx, &entity.DocumentTask{
			ID: buildCtx.TaskID, CurrentStage: enum.TaskStageKeywordIndex,
		}); err != nil {
			return err
		}
		keywordStartDetail, _ := json.Marshal(map[string]any{"chunkCount": vectorSize})
		keywordStartLog := &entity.DocumentTaskLog{
			TaskId: buildCtx.TaskID, DocumentId: buildCtx.DocumentID,
			StageType: enum.TaskStageKeywordIndex, EventType: enum.TaskEventStart,
			LogLevel: enum.LogLevelInfo, OperatorType: enum.OperatorTypeSystem,
			Content: "开始构建关键词索引", DetailJson: string(keywordStartDetail),
		}
		return p.Repo.InsertTaskLog(txCtx, keywordStartLog)
	}
	if err := p.Repo.Do(ctx, markKeywordIndexTx); err != nil {
		return err
	}

	// 执行关键词索引构建
	keywordStartedTime := time.Now()
	if err := p.Port.BuildIndexes(ctx, buildCtx.ChildChunks); err != nil {
		return err
	}
	buildCtx.KeywordCostMillis = time.Since(keywordStartedTime).Milliseconds()

	// 标记完成
	markKeywordCompleteTx := func(txCtx context.Context) error {
		keywordEndDetail, _ := json.Marshal(map[string]any{
			"chunkCount": vectorSize,
			"costMillis": buildCtx.KeywordCostMillis,
		})
		keywordEndLog := &entity.DocumentTaskLog{
			TaskId: buildCtx.TaskID, DocumentId: buildCtx.DocumentID,
			StageType: enum.TaskStageKeywordIndex, EventType: enum.TaskEventComplete,
			LogLevel: enum.LogLevelInfo, OperatorType: enum.OperatorTypeSystem,
			Content: "关键词索引完成", DetailJson: string(keywordEndDetail),
		}
		return p.Repo.InsertTaskLog(txCtx, keywordEndLog)
	}
	return p.Repo.Do(ctx, markKeywordCompleteTx)
}
