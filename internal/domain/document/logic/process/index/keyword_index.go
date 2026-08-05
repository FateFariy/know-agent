package index

import (
	"context"
	"encoding/json"
	"time"

	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/document/model/vo"
)

// KeywordIndexPhase 关键词索引阶段
type KeywordIndexPhase struct {
	deps *PhaseDeps
}

func NewKeywordIndexPhase(deps *PhaseDeps) *KeywordIndexPhase {
	return &KeywordIndexPhase{deps: deps}
}

func (p *KeywordIndexPhase) Name() string {
	return "关键词索引阶段"
}

func (p *KeywordIndexPhase) Execute(ctx context.Context, buildCtx *BuildContext) error {
	vectorSize := len(buildCtx.ChildChunks)

	// 标记开始关键词索引
	markKeywordIndexTx := func(txCtx context.Context) error {
		if err := p.deps.Repo.UpdateTaskById(txCtx, &entity.DocumentTask{
			ID: buildCtx.TaskID, CurrentStage: vo.TaskStageKeywordIndex,
		}); err != nil {
			return err
		}
		keywordStartDetail, _ := json.Marshal(map[string]any{"chunkCount": vectorSize})
		keywordStartLog := &entity.DocumentTaskLog{
			TaskId: buildCtx.TaskID, DocumentId: buildCtx.DocumentID,
			StageType: vo.TaskStageKeywordIndex, EventType: vo.TaskEventStart,
			LogLevel: vo.LogLevelInfo, OperatorType: vo.OperatorTypeSystem,
			Content: "开始构建关键词索引", DetailJson: string(keywordStartDetail),
		}
		return p.deps.Repo.InsertTaskLog(txCtx, keywordStartLog)
	}
	if err := p.deps.Repo.Do(ctx, markKeywordIndexTx); err != nil {
		return err
	}

	// 执行关键词索引构建
	keywordStartedNanos := time.Now()
	if err := p.deps.Port.BuildIndexes(ctx, buildCtx.ChildChunks); err != nil {
		return err
	}
	buildCtx.KeywordCostMillis = time.Since(keywordStartedNanos).Milliseconds()

	// 标记完成
	markKeywordCompleteTx := func(txCtx context.Context) error {
		keywordEndDetail, _ := json.Marshal(map[string]any{
			"chunkCount": vectorSize,
			"costMillis": buildCtx.KeywordCostMillis,
		})
		keywordEndLog := &entity.DocumentTaskLog{
			TaskId: buildCtx.TaskID, DocumentId: buildCtx.DocumentID,
			StageType: vo.TaskStageKeywordIndex, EventType: vo.TaskEventComplete,
			LogLevel: vo.LogLevelInfo, OperatorType: vo.OperatorTypeSystem,
			Content: "关键词索引完成", DetailJson: string(keywordEndDetail),
		}
		return p.deps.Repo.InsertTaskLog(txCtx, keywordEndLog)
	}
	return p.deps.Repo.Do(ctx, markKeywordCompleteTx)
}
