package analysis

import (
	"context"
	"fmt"
	"time"

	"github.com/swiftbit/know-agent/common/logx"
)

// PhaseChain 解析路由阶段责任链
type PhaseChain struct {
	phases []Phase
}

// NewPhaseChain 创建并注册所有阶段
func NewPhaseChain(phases []Phase) *PhaseChain {
	return &PhaseChain{phases: phases}
}

// Run 执行责任链
func (c *PhaseChain) Run(ctx context.Context, parseCtx *Context) (err error) {
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
