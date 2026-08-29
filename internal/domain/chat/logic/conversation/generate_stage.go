package conversation

import (
	"context"
	"errors"
	"fmt"

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
	plan := convCtx.ExecutionPlan.Load()
	if plan == nil {
		return fmt.Errorf("invalid value")
	}

	// 语义缓存命中：检索/证据阶段已被跳过，这里补发引用（两种策略都需要），
	// 并在「复用答案」策略下直接复用缓存答案、跳过生成。
	if convCtx.IsCacheHit() {
		if refs := plan.RetrievalResult.FlattenReferences(); len(refs) > 0 {
			_ = convCtx.PublishReferences(refs)
		}
		if convCtx.ReuseStrategy() == enum.ReuseAnswerAndRetrieval &&
			utils.IsNotBlank(convCtx.CacheEntry().AnswerDraft) {
			return g.reuseCachedAnswer(ctx, convCtx, plan)
		}
		// 否则：复用检索结果，落到下方正常生成（命中 + ReuseRetrievalOnly，
		// 或配置要复用答案但历史条目无 AnswerDraft 时的兼容降级）
	}

	if plan.RetrievalResult.IsEmpty() {
		text := utils.BlankToDefault(plan.NoEvidenceReply, defaultNoEvidenceReply)
		if err := convCtx.PublishText(text); err != nil {
			return err
		}
		return nil
	}
	switch plan.Mode {
	case enum.ExecutionModeClarification:
		return g.clarificationExecute(ctx, convCtx)
	case enum.ExecutionModeRetrieval:
		return g.ragExecute(ctx, convCtx)
	case enum.ExecutionModeReactAgent:
		return g.agentExecute(ctx, convCtx)
	}

	return nil
}

// reuseCachedAnswer 命中且复用答案策略：直接采用缓存答案，跳过 LLM 生成。
// 引用已在 Execute 入口补发；答案文本与溯源信息从缓存条目获取。
func (g *GenerateStage) reuseCachedAnswer(ctx context.Context, convCtx *Context, plan *vo.ConversationExecutionPlan) error {
	ctx = vo.OnStart(ctx, enum.ConversationTraceStageAnswerGenerate, plan.Mode.Name(), &vo.StageInput{SummaryText: "命中语义缓存，复用答案。"})

	if err := convCtx.PublishThinking("命中语义缓存，复用答案。"); err != nil {
		return nil
	}

	answer := convCtx.CacheEntry().AnswerDraft
	convCtx.WriteAnswerBuffer(answer)
	if err := convCtx.PublishText(answer); err != nil {
		return err
	}

	vo.OnEnd(ctx, &vo.StageOutput{
		SummaryText: "已复用缓存答案。",
		Snapshot: map[string]any{
			"cacheHit":   true,
			"similarity": convCtx.CacheSimilarity(),
		},
	})
	return nil
}

func (g *GenerateStage) clarificationExecute(ctx context.Context, convCtx *Context) error {
	plan := convCtx.ExecutionPlan.Load()
	ctx = vo.OnStart(ctx, enum.ConversationTraceStageAnswerGenerate, plan.Mode.Name(), &vo.StageInput{SummaryText: "当前候选存在歧义，先返回澄清问题。"})

	// 从执行计划中取出澄清文本、原因与候选项；原因写入调试轨迹以便离线分析
	reply := utils.BlankToDefault(plan.ClarificationReply, "当前我无法稳定判断你想问哪份知识文档，请补充更具体的文档名、主题或关键词。")
	reason := plan.ClarificationReason
	options := plan.ClarificationOptions

	if debugTrace := convCtx.DebugTrace.Load(); debugTrace != nil {
		if utils.IsNotBlank(reason) {
			debugTrace.AddRetrievalNotes(reason)
		}
	}

	// 向客户端流发布思考事件；原因非空时再追加一条状态事件
	if err := convCtx.PublishThinking("当前问题涉及多份候选文档，先向你确认知识范围。"); err != nil {
		return nil
	}
	if utils.IsNotBlank(reason) {
		if err := convCtx.PublishStatus(reason); err != nil {
			return nil
		}
	}

	// 提交追踪快照（包含回复、原因、候选项）
	snapshot := map[string]any{
		"clarificationReply":   reply,
		"clarificationReason":  reason,
		"clarificationOptions": options,
	}
	vo.OnEnd(ctx, &vo.StageOutput{SummaryText: "已返回澄清问题。", Snapshot: snapshot})

	return convCtx.PublishText(reply)
}

func (g *GenerateStage) ragExecute(ctx context.Context, convCtx *Context) error {
	plan := convCtx.ExecutionPlan.Load()
	ctx = vo.OnStart(ctx, enum.ConversationTraceStageAnswerGenerate, plan.Mode.Name(), &vo.StageInput{SummaryText: "正在基于证据生成回答。"})

	if err := convCtx.PublishThinking("证据整理完成，正在基于证据生成回答。"); err != nil {
		return err
	}
	prompt := plan.PromptAssemblyResult
	if prompt == nil {
		return errors.New("执行计划中缺少Prompt")
	}
	streamCh, err := g.chatModel.Stream(ctx, enum.ChatStageRagAnswer, prompt.SystemPrompt, prompt.UserPrompt)
	if err != nil {
		logx.Errorf("模型流式调用失败: conversationId=%s, error=%v", convCtx.ConversationId, err)
		vo.OnError(ctx, "答案生成失败。", err)
		return err
	}
	return g.channel(ctx, convCtx, streamCh)
}

func (g *GenerateStage) agentExecute(ctx context.Context, convCtx *Context) error {
	//plan := convCtx.ExecutionPlan.Load()
	//
	// 	publishThinking(convCtx, "问题涉及多方面信息，交由 ReAct Agent 综合回答。")
	//
	// 	agentStage, err := e.tracer.OnStart(ctx, convCtx.Trace, vo.ConversationTraceStageReActAgent,
	// 		e.Mode().Name(), "ReAct Agent 正在思考与执行。", nil)
	//
	// 	streamCh, err := e.reactAgent.Stream(ctx, plan.Question)
	// 	if err != nil {
	// 		logx.Errorf("ReAct Agent 调用失败: conversationId=%s err=%v", convCtx.ConversationId, err)
	// 		e.tracer.OnErr(ctx, agentStage, "ReAct Agent 执行失败。", err, nil)
	// 		publishText(convCtx, utils.BlankToDefault(plan.NoEvidenceReply, defaultNoEvidenceReply))
	// 		return nil, err
	// 	}
	//
	// 	snapshot := map[string]any{
	// 		"firstResponseTimeMs": convCtx.FirstResponseTimeMs.Load(),
	// 		"answerLength":        convCtx.AnswerLength(),
	// 	}
	// 	_ = e.tracer.OnEnd(ctx, agentStage, "ReAct Agent 回答完成。", snapshot)
	return nil
}

func (g *GenerateStage) channel(ctx context.Context, convCtx *Context, ch <-chan string) error {
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
		case chunk, ok := <-ch:
			if !ok {
				return nil
			}
			if err := convCtx.PublishText(chunk); err != nil {
				return err
			}
		}
	}
}
