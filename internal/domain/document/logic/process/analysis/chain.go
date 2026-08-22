package analysis

import (
	"context"
	"fmt"
	"time"

	"github.com/swiftbit/know-agent/common/logx"
	"github.com/swiftbit/know-agent/internal/domain/document/adapter"
	"github.com/swiftbit/know-agent/internal/svc"
)

// Chain 解析路由阶段责任链
type Chain struct {
	phases []Stage
}

// NewAnalysisChain 创建并注册所有阶段
func NewAnalysisChain(svcCtx *svc.ServiceContext, repo adapter.DocumentRepository, tableRepo adapter.TableRepository,
	port *adapter.DocumentPort, resolver IndexingConfigResolver) *Chain {
	phases := []Stage{
		NewInitializationStage(repo),             // 1. 初始化任务状态，标记解析开始
		NewDownloadStage(port),                   // 2. 从对象存储下载原始文件
		NewParseStage(repo, port),                // 3. 解析文件内容，生成分析结果
		NewSaveStage(repo, tableRepo, port),      // 4. 保存解析产物（文本、结构节点等）
		NewStrategyStage(svcCtx, repo, resolver), // 5. 生成推荐切分策略
		NewFinalizationStage(repo, port),         // 6. 持久化策略，更新文档/任务最终状态
	}

	return &Chain{phases: phases}
}

// Run 执行责任链
func (c *Chain) Run(ctx context.Context, parseCtx *Context) (err error) {
	for _, phase := range c.phases {
		phaseName := phase.Name()
		startTime := time.Now()
		logx.Infof("[ParseChain] 开始执行阶段: %s, documentId=%d, taskId=%d", phaseName, parseCtx.DocumentId, parseCtx.TaskId)

		if err = phase.Execute(ctx, parseCtx); err != nil {
			logx.Errorf("[ParseChain] 阶段 %s 执行失败: %v", phaseName, err)
			return fmt.Errorf("阶段 %s 执行失败: %w", phaseName, err)
		}

		logx.Infof("[ParseChain] 阶段 %s 执行成功, costMillis=%d", phaseName, time.Since(startTime).Milliseconds())
	}
	return nil
}
