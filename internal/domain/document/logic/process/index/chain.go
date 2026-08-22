package index

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/swiftbit/know-agent/common/logx"
	"github.com/swiftbit/know-agent/internal/domain/document/adapter"
	"github.com/swiftbit/know-agent/internal/domain/document/logic/process/chunk"
	errorx "github.com/swiftbit/know-agent/internal/error"
)

// BuildIndexChain 阶段责任链
type BuildIndexChain struct {
	stages []Stage
}

func NewBuildIndexChain(
	repo adapter.DocumentRepository,
	vecIndexer adapter.VectorIndexer,
	keyIndexer adapter.KeywordIndexer,
	registry *chunk.Registry,
	resolver IndexingConfigResolver,
	tokenizer Tokenizer,
	// builder GraphRagBuilder,
) *BuildIndexChain {
	stages := []Stage{
		NewPreparationStage(repo),                             // 1. 准备阶段：加载任务、验证状态、推进任务状态
		NewChunkingStage(repo, registry, resolver, tokenizer), // 2. 切块阶段：执行切块流水线
		NewChunkPostPhase(repo),                               // 3. 切块后处理阶段：构建父子块实体并持久化
		NewVectorizePhase(repo, vecIndexer),                   // 4. 向量化阶段：批量向量化并回写状态
		NewKeywordIndexStage(repo, keyIndexer),                // 5. 关键词索引阶段：构建关键词索引
		//NewGraphRagPhase(repo, builder),                             // 6. GraphRAG构建阶段：构建实体关系图谱
		NewCompletionStage(repo), // 7. 完成阶段：事务性更新任务/方案/文档状态
	}
	return &BuildIndexChain{stages: stages}
}

// Run 执行责任链
func (c *BuildIndexChain) Run(ctx context.Context, buildCtx *Context) error {
	for _, phase := range c.stages {
		phaseName := phase.Name()
		startTime := time.Now()
		logx.Infof("[BuildIndexChain] 开始执行阶段: %s, documentId=%d, taskId=%d", phaseName, buildCtx.DocumentId, buildCtx.TaskId)

		if err := phase.Execute(ctx, buildCtx); err != nil {
			if errors.Is(err, errorx.ErrGraphRagBuildFailed) {
				logx.Warnf("[BuildIndexChain] 阶段 %s 执行失败: %v", phaseName, err)
				return nil
			}
			logx.Errorf("[BuildIndexChain] 阶段 %s 执行失败: %v", phaseName, err)
			return fmt.Errorf("阶段 %s 执行失败: %w", phaseName, err)
		}

		logx.Infof("[BuildIndexChain] 阶段 %s 执行成功, costMillis=%d", phaseName, time.Since(startTime).Milliseconds())
	}
	return nil
}
