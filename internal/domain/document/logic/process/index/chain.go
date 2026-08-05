package index

import (
	"context"
	"fmt"

	"github.com/swiftbit/know-agent/common/logx"
)

// PhaseChain 阶段责任链
type PhaseChain struct {
	phases []BuildPhase
	deps   *PhaseDeps
}

// NewPhaseChain 创建并注册所有阶段
func NewPhaseChain(deps *PhaseDeps) *PhaseChain {
	chain := &PhaseChain{deps: deps}
	chain.phases = []BuildPhase{
		NewValidationPhase(deps),
		NewPreparationPhase(deps),
		NewChunkingPhase(deps),
		NewVectorizePhase(deps),
		NewKeywordIndexPhase(deps),
		NewGraphRagPhase(deps),
		NewRaptorPhase(deps),
		NewCompletionPhase(deps),
	}
	return chain
}

// Run 执行责任链
func (c *PhaseChain) Run(ctx context.Context, buildCtx *BuildContext) error {
	for _, phase := range c.phases {
		phaseName := phase.Name()
		logx.Infof("[PhaseChain] 开始执行阶段: %s, documentId=%d, taskId=%d", phaseName, buildCtx.DocumentID, buildCtx.TaskID)

		if err := phase.Execute(ctx, buildCtx); err != nil {
			logx.Errorf("[PhaseChain] 阶段 %s 执行失败: %v", phaseName, err)
			return fmt.Errorf("阶段 %s 执行失败: %w", phaseName, err)
		}

		logx.Infof("[PhaseChain] 阶段 %s 执行成功", phaseName)
	}
	return nil
}
