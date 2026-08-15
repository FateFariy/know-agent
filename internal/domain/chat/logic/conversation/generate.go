package conversation

import (
	"context"

	"github.com/swiftbit/know-agent/internal/domain/chat/model/enum"
)

type GenerateStage struct {
	//executorRegistry *executor.Registry
}

func NewGenerateStage() *GenerateStage {
	return &GenerateStage{}
}

func (g *GenerateStage) Name() string {
	return enum.ConversationTraceStageAnswerGenerate.Name
}

func (g *GenerateStage) Execute(ctx context.Context, convCtx *Context) error {
	// 发送"上下文分析完成"的思考事件（前端调试/感知）
	if err := convCtx.PublishThinking("上下文分析完成，已准备执行计划。"); err != nil {
		return err
	}
	plan := convCtx.ExecutionPlan.Load()
	// 根据执行计划 Mode 从执行器注册表解析执行器
	exec, err := g.executorRegistry.Get(plan.Mode)
	if err != nil {
		return err
	}

	resultCh, err := exec.Execute(ctx, convCtx)
	if err != nil {
		return err
	}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case chunk, ok := <-resultCh:
			if !ok {
				return nil
			}
			if err = convCtx.PublishText(chunk); err != nil {
				return err
			}
		}
	}
}
