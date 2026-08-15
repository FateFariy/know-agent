package conversation

import (
	"context"
	"errors"

	"github.com/swiftbit/know-agent/common"
	"github.com/swiftbit/know-agent/common/logx"
	"github.com/swiftbit/know-agent/internal/domain/chat/adapter"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/memory"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/recommend"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/enum"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
	errorx "github.com/swiftbit/know-agent/internal/error"
)

type Chain struct {
	repo            adapter.ChatRepository
	stages          []Stage
	runtime         *ChatRuntimeRegistry
	docGateway      adapter.DocumentGateway
	memoryManager   memory.SessionMemoryManager
	recommender     recommend.QuestionRecommender
	distributedLock adapter.DistributedLock
	checkPointStore adapter.CheckPointStore
}

func NewChain(repo adapter.ChatRepository, runtime *ChatRuntimeRegistry, stages ...Stage) *Chain {
	return &Chain{
		repo:    repo,
		stages:  stages,
		runtime: runtime,
	}
}

func (c *Chain) Run(ctx context.Context, convCtx *Context) error {
	defer c.final(convCtx)

	for _, stage := range c.stages {
		if err := stage.Execute(ctx, convCtx); err != nil {
			if errors.Is(err, errorx.ErrSessionRunning) {
				_ = c.startFailed(ctx, convCtx)
			} else if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				c.stop(ctx, convCtx, "客户端已取消请求")
			} else {
				c.finishFailed(ctx, convCtx, err)
			}
			return err
		}
	}
	c.finishSuccessfully(ctx, convCtx)
	return nil
}

func (c *Chain) Stop(ctx context.Context, convCtx *Context, reason string) (string, bool) {
	convCtx, ok := c.runtime.Get(conversationId)
	if !ok {
		return "没有找到正在执行的会话", false
	}
	return c.stop(ctx, convCtx, reason)
}

// stop 停止：原子切换状态 -> 发送停止事件 -> 落库 -> 清理
func (c *Chain) stop(ctx context.Context, convCtx *Context, reason string) (string, bool) {
	if !convCtx.Finalized.CompareAndSwap(false, true) {
		return "会话已停止", false
	}
	if curr, exists := c.runtime.Get(convCtx.ConversationId); exists && curr != convCtx {
		return "会话已由新的执行接管", false
	}

	// todo 中断 ReactAgent
	//        try {
	//         businessChatReactAgent.interrupt(taskInfo.runnableConfig());
	//     } catch (RuntimeException exception) {
	//         log.debug("中断 ReactAgent 时出现异常，继续释放资源", exception);
	//     }
	responseMessage := "已停止会话生成"
	ctx = vo.OnStart(ctx, enum.ConversationTraceStageFinalize,
		convCtx.ExecutionModeName(), &vo.StageInput{SummaryText: "正在收尾停止中的会话。"})

	// 发送 status 事件
	err := convCtx.Sink.Status("⏹ "+reason, convCtx.ConversationId, convCtx.ExchangeId)
	if err != nil {
		logx.Warnf("发送停止事件失败, conversationId=%s, exchangeId=%d, err=%v", convCtx.ConversationId, convCtx.ExchangeId, err)
		responseMessage = "会话已停止，停止事件发送失败"
	}

	// 构造停止态 exchange 并落库
	stopExchange := convCtx.BuildChatExchange(enum.ChatTurnStatusStopped, reason)
	if err = c.completeExchange(ctx, stopExchange); err == nil {
		metadata := map[string]any{
			"finalStatus":  enum.ChatTurnStatusName(enum.ChatTurnStatusStopped),
			"reason":       reason,
			"answerLength": convCtx.AnswerLength(),
		}
		_ = vo.OnEnd(ctx, &vo.StageOutput{SummaryText: "会话已按停止状态收尾", Snapshot: metadata})
	} else {
		responseMessage = "会话已停止，收尾落库失败"
		_ = vo.OnError(ctx, "会话已按停止状态收尾", err)
	}

	return responseMessage, true
}

// startFailed 处理会话启动失败的情况，回写失败状态并拒绝，让客户端稍后重试
func (c *Chain) startFailed(ctx context.Context, convCtx *Context) error {
	failExchange := &entity.ChatExchange{
		ID:             convCtx.ExchangeId,
		ConversationId: convCtx.ConversationId,
		TurnStatus:     enum.ChatTurnStatusFailed,
		ErrorMessage:   "该会话当前正在执行中，请稍后再试",
	}
	return c.completeExchange(ctx, failExchange)
}

// finishSuccessfully 以成功状态完成当前会话交互
func (c *Chain) finishSuccessfully(ctx context.Context, convCtx *Context) {
	// CAS确保收尾仅执行一次
	if !convCtx.Finalized.CompareAndSwap(false, true) {
		return
	}

	// 开启finalize追踪阶段
	finalizeCtx := vo.OnStart(ctx, enum.ConversationTraceStageFinalize,
		convCtx.ExecutionModeName(), &vo.StageInput{SummaryText: "正在收尾已完成会话。"})

	answer := convCtx.Answer()
	uniqueReferences := convCtx.UniqueReferences()
	recommendations := convCtx.Recommendations

	// 构建成功态交换记录并落库
	successExchange := convCtx.BuildChatExchange(enum.ChatTurnStatusCompleted, "")
	successExchange.Recommendations = common.ToJSONArray(recommendations)
	if err := c.completeExchange(ctx, successExchange); err == nil {
		// 落库成功，记录完成快照
		snapshot := map[string]any{
			"finalStatus":         enum.ChatTurnStatusName(enum.ChatTurnStatusCompleted),
			"recommendationCount": len(recommendations),
			"recommendations":     recommendations,
			"referenceCount":      len(uniqueReferences),
			"answerLength":        len(answer),
		}
		_ = vo.OnEnd(finalizeCtx, &vo.StageOutput{SummaryText: "会话已按完成状态收尾。", Snapshot: snapshot})
	} else {
		_ = vo.OnError(finalizeCtx, "会话收尾落库失败", err)
	}
}

// finishFailed 以失败状态完成当前会话交互
func (c *Chain) finishFailed(ctx context.Context, convCtx *Context, err error) {
	// CAS确保收尾仅执行一次
	if !convCtx.Finalized.CompareAndSwap(false, true) {
		return
	}
	logx.Errorf("会话执行失败, conversationId=%s, exchangeId=%d, error=%v", convCtx.ConversationId, convCtx.ExchangeId, err)

	// 开启finalize追踪阶段（失败时忽略错误，不影响主流程）
	errorMessage := err.Error()
	finalizeCtx := vo.OnStart(ctx, enum.ConversationTraceStageFinalize,
		convCtx.ExecutionModeName(), &vo.StageInput{SummaryText: "正在收尾失败会话。"})

	// 向下游发送失败事件
	err = convCtx.Sink.Error(errorMessage, convCtx.ConversationId, convCtx.ExchangeId)
	if err != nil {
		logx.Warnf("发送失败事件失败, conversationId=%s, exchangeId=%d, error=%v", convCtx.ConversationId, convCtx.ExchangeId, err)
	}

	// 构建失败态交换记录并落库
	failExchange := convCtx.BuildChatExchange(enum.ChatTurnStatusFailed, errorMessage)
	if err = c.completeExchange(ctx, failExchange); err == nil {
		// 落库成功，记录失败快照
		snapshot := map[string]any{
			"finalStatus":  enum.ChatTurnStatusName(enum.ChatTurnStatusFailed),
			"errorMessage": errorMessage,
			"answerLength": convCtx.AnswerLength(),
		}
		_ = vo.OnEnd(finalizeCtx, &vo.StageOutput{SummaryText: "会话已按失败状态收尾。", Snapshot: snapshot})
	} else {
		_ = vo.OnError(finalizeCtx, "失败态收尾失败。", err)
	}
}

// final 会话收尾，确保仅首次调用生效，避免重复收尾
func (c *Chain) final(convCtx *Context) {
	if !convCtx.Finalized.CompareAndSwap(false, true) {
		return
	}
	c.memoryManager.RefreshConversationSummaryAsync(convCtx.ConversationId)
	c.refreshDebugTraceRuntimeStats(convCtx)
	c.runtime.Remove(convCtx.ConversationId, convCtx)
	convCtx.ReleaseResources()
}

// refreshDebugTraceRuntimeStats 刷新调试轨迹中的统计信息
func (c *Chain) refreshDebugTraceRuntimeStats(convCtx *Context) {
	debugTrace := convCtx.DebugTrace.Load()
	if debugTrace == nil {
		return
	}
	modelUsageTraces := convCtx.Trace.SnapshotModelUsageTraces()
	debugTrace.ModelUsageTraces = modelUsageTraces
	debugTrace.LimitStats = &vo.ChatLimitStats{
		ModelCallsUsed:        len(modelUsageTraces),
		ToolCallsUsed:         len(convCtx.SnapshotUsedTools()),
		ModelCallsRunLimit:    c.options.maxModelCallsPerRun,
		ToolCallsRunLimit:     c.options.maxToolCallsPerRun,
		ModelCallsThreadLimit: c.options.maxModelCallsPerThread,
		ToolCallsThreadLimit:  c.options.maxToolCallsPerThread,
	}
	convCtx.DebugTrace.Store(debugTrace)
}

// completeExchange 完成会话交互（exchange）
func (c *Chain) completeExchange(ctx context.Context, exchange *entity.ChatExchange) error {
	completeFn := func(txCtx context.Context) error {
		// 更新交互记录（含答案、耗时、最终状态等，由调用方已在 exchange 对象中填充）
		if err := c.repo.UpdateExchangeById(txCtx, exchange); err != nil {
			return err
		}
		// 将对应会话的状态重置为 Idle（释放"运行中"标记）
		dialogue := &entity.ChatDialogue{
			ConversationId: exchange.ConversationId,
			SessionStatus:  enum.ChatSessionStatusIdle,
		}
		return c.repo.UpdateDialogueByConversationId(txCtx, dialogue)
	}
	if err := c.repo.Do(ctx, completeFn); err != nil {
		logx.Errorf("会话落库失败, conversationId=%s, exchangeId=%d, err=%v",
			exchange.ConversationId, exchange.ID, err)
		return err
	}
	return nil
}
