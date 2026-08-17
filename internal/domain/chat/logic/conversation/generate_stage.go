package conversation

import (
	"context"
	"errors"

	"github.com/swiftbit/know-agent/common/logx"
	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/chat/adapter/model"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/enum"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
)

const (
	defaultNoEvidenceReply = "当前没有足够证据支持明确回答。"
)

type GenerateStage struct {
	//executorRegistry *executor.Registry
	chatModel model.ChatModel
}

func NewGenerateStage(chatModel model.ChatModel) *GenerateStage {
	return &GenerateStage{
		chatModel: chatModel,
	}
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

	if plan.RetrievalResult.IsEmpty() {
		text := utils.BlankToDefault(plan.NoEvidenceReply, defaultNoEvidenceReply)
		if err = convCtx.PublishText(text); err != nil {
			return err
		}
		return nil
	}

	defer func() {
		output := &vo.StageOutput{
			SummaryText: "答案生成完成。",
			Snapshot: map[string]any{
				"firstResponseTimeMs": convCtx.FirstResponseTimeMs.Load(),
				"answerLength":        convCtx.AnswerLength(),
			}}
		vo.OnEnd(ctx, output)
	}()
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

func (g *GenerateStage) ClarificationExecute(ctx context.Context, convCtx *Context) error {
	return nil
}

func (g *GenerateStage) ragExecute(ctx context.Context, convCtx *Context) (<-chan string, error) {
	ctx = vo.OnStart(ctx, enum.ConversationTraceStageAnswerGenerate, g.Name(), &vo.StageInput{SummaryText: "正在基于证据生成回答。"})

	if err := convCtx.PublishThinking("证据整理完成，正在基于证据生成回答。"); err != nil {
		return nil, err
	}
	prompt := convCtx.ExecutionPlan.Load().PromptAssemblyResult
	if prompt == nil {
		return nil, errors.New("执行计划中缺少Prompt")
	}
	streamCh, err := g.chatModel.Stream(ctx, enum.ChatStageRagAnswer, prompt.SystemPrompt, prompt.UserPrompt)
	if err != nil {
		logx.Errorf("模型流式调用失败: conversationId=%s, error=%v", convCtx.ConversationId, err)
		vo.OnError(ctx, "答案生成失败。", err)
		return nil, err
	}
	return streamCh, nil
}
