package conversation

import (
	"context"
	"errors"
	"time"

	"github.com/swiftbit/know-agent/common"
	"github.com/swiftbit/know-agent/common/logx"
	"github.com/swiftbit/know-agent/internal/domain/chat/adapter"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/memory"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/enum"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
	errorx "github.com/swiftbit/know-agent/internal/error"
)

type Chain struct {
	repo            adapter.ChatRepository
	stages          []Stage
	runtimeRegistry *ChatRuntimeRegistry
	documentFetcher adapter.DocumentFetcher
	memoryManager   memory.SessionMemoryManager
	distributedLock adapter.DistributedLock
	checkPointStore adapter.CheckPointStore
}

func NewChain(repo adapter.ChatRepository, runtimeRegistry *ChatRuntimeRegistry, stages ...Stage) *Chain {
	return &Chain{
		repo:            repo,
		stages:          stages,
		runtimeRegistry: runtimeRegistry,
	}
}

func (c *Chain) Run(ctx context.Context, convCtx *Context) error {
	for _, stage := range c.stages {
		if err := stage.Execute(ctx, convCtx); err != nil {
			if errors.Is(err, errorx.ErrSessionRunning) {
				return c.startFailed(ctx, convCtx)
			}
			return err
		}
	}
	return nil
}

func (c *Chain) Stop(ctx context.Context, convCtx *Context, reason string) (string, bool) {
	convCtx, ok := c.runtimeRegistry.Get(conversationId)
	if !ok {
		return "没有找到正在执行的会话", false
	}
	return c.stopTask(ctx, convCtx, reason)
}

// stopTask 停止任务：原子切换状态 -> 发送停止事件 -> 落库 -> 清理
func (c *Chain) stopTask(ctx context.Context, convCtx *Context, reason string) (string, bool) {
	if !convCtx.Finalized.CompareAndSwap(false, true) {
		return "会话已停止", false
	}
	if curr, exists := c.runtimeRegistry.Get(convCtx.ConversationId); exists && curr != convCtx {
		return "会话已由新的执行接管", false
	}
	// defer 中刷新会话摘要 + 执行清理
	// 使用 defer 确保即便后续步骤出错，这两个清理动作也会执行
	defer func() {
		_ = recover()
		c.memoryManager.RefreshConversationSummaryAsync(convCtx.ConversationId)
		c.cleanup(convCtx)
	}()

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

	// 刷新调试轨迹统计
	c.refreshDebugTraceRuntimeStats(convCtx)

	// 构造停止态 exchange 并落库
	stopExchange := c.buildCurrentChatExchange(convCtx, enum.ChatTurnStatusStopped, reason)
	if err := c.completeExchange(ctx, stopExchange); err == nil {
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

// finishSuccessfully 以成功状态完成当前会话交互（exchange）
//
// 执行流程：
//  1. 原子检查 Finalized 标志（CAS：false → true），避免重复收尾
//  2. defer 中异步刷新会话摘要并执行清理（任何返回路径都会执行）
//  3. 启动 finalize 与 recommendation 两个追踪阶段（忽略其错误）
//  4. 生成推荐追问：需要澄清时返回澄清选项，否则由 recommender 生成
//  5. 向客户端流补发引用事件、推荐事件，并发送流 Complete
//  6. 刷新 DebugTrace 运行时统计
//  7. 组装成功态 ChatExchange，调用 completeExchange 落库；根据落库结果完成或标记追踪阶段
func (c *Chain) finishSuccessfully(ctx context.Context, convCtx *Context) {
	// 原子检查 Finalized 标志（CAS），确保仅首次调用生效，避免重复收尾
	if !convCtx.Finalized.CompareAndSwap(false, true) {
		return
	}

	// 发送 finish 事件
	_ = convCtx.PublishFinish()

	// defer 中刷新会话摘要 + 执行清理
	// 使用 defer 确保即便后续步骤出错，这两个清理动作也会执行
	defer func() {
		_ = recover()
		c.memoryManager.RefreshConversationSummaryAsync(convCtx.ConversationId)
		c.cleanup(convCtx)
	}()

	// 从 convCtx 中取出当前答案与去重后的引用列表（供后续发送事件与落库使用）
	answer := convCtx.Answer()
	uniqueReferences := convCtx.UniqueReferences()

	// 启动 recommendation 阶段
	recommendCtx := vo.OnStart(ctx, enum.ConversationTraceStageRecommendation,
		convCtx.ExecutionModeName(), &vo.StageInput{SummaryText: "正在生成推荐追问。"})

	// 生成推荐追问
	// - 若本次交互是澄清（NeedClarification 为真），则直接使用澄清选项作为推荐
	// - 否则，拉取最近交互记录，由 recommender 基于当前问答与历史生成推荐
	var recommendations []string
	if convCtx.NeedClarification() {
		recommendations = convCtx.ClarificationOptions()
	} else {
		recentExchanges := c.fetchRecentExchanges(ctx, convCtx.ConversationId, convCtx.ExchangeId)
		recommendations = c.recommender.Generate(ctx, convCtx.Question, answer, recentExchanges)
	}

	// 完成 recommendation 追踪阶段，并写入推荐数量快照
	snapshot := map[string]any{"recommendationCount": len(recommendations), "recommendations": recommendations}
	_ = vo.OnEnd(recommendCtx, &vo.StageOutput{SummaryText: "推荐追问生成完成。", Snapshot: snapshot})

	// 向客户端流补发引用事件，最后发送流 Complete 信号
	if len(recommendations) > 0 {
		err := convCtx.Sink.Recommendations(recommendations, convCtx.ConversationId, convCtx.ExchangeId)
		if err != nil {
			logx.Warnf("发送推荐事件失败, conversationId=%s, exchangeId=%d, err=%v", convCtx.ConversationId, convCtx.ExchangeId, err)
		}
	}

	// 启动 finalize 与 recommendation 两个追踪阶段
	finalizeCtx := vo.OnStart(ctx, enum.ConversationTraceStageFinalize,
		convCtx.ExecutionModeName(), &vo.StageInput{SummaryText: "正在收尾已完成会话。"})

	// 刷新 DebugTrace 的运行时统计
	c.refreshDebugTraceRuntimeStats(convCtx)

	// 组装成功态 ChatExchange，调用 completeExchange 落库；并根据落库结果完成或标记 finalize 追踪阶段
	successExchange := c.buildCurrentChatExchange(convCtx, enum.ChatTurnStatusCompleted, "")
	successExchange.Recommendations = common.ToJSONArray(recommendations)
	if err := c.completeExchange(ctx, successExchange); err == nil {
		// 落库成功：完成 finalize 追踪阶段，写入完成快照（含推荐、引用、答案长度）
		snapshot = map[string]any{
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

// finishWithFailure 以失败状态收尾当前会话交互（exchange）。
//
// 执行流程：
//  1. 原子检查 Finalized 标志（CAS：false → true），确保仅首次调用生效，避免重复收尾
//  2. 打印错误日志
//  3. defer 中异步刷新会话摘要并执行清理（保证在任何 return 路径都会执行）
//  4. 启动 finalize 追踪阶段
//  5. 发送失败事件与流 Complete 到客户端（失败不影响主流程）
//  6. 刷新 DebugTrace 的运行时统计
//  7. 组装失败态 ChatExchange，调用 completeExchange 落库；并根据落库结果完成或标记追踪阶段
func (c *Chain) finishWithFailure(ctx context.Context, convCtx *Context, err error) {
	// 原子检查 Finalized 标志（CAS），确保仅首次调用生效，避免重复收尾
	if !convCtx.Finalized.CompareAndSwap(false, true) {
		return
	}
	// 打印错误日志
	logx.Errorf("会话执行失败, conversationId=%s, exchangeId=%d, error=%v", convCtx.ConversationId, convCtx.ExchangeId, err)

	// defer 中刷新会话摘要 + 执行清理
	// 使用 defer 确保即便后续步骤出错，这两个清理动作也会执行
	defer func() {
		_ = recover()
		c.memoryManager.RefreshConversationSummaryAsync(convCtx.ConversationId)
		c.cleanup(convCtx)
	}()

	// 启动 finalize 追踪阶段（失败时忽略错误，不影响主流程）
	errorMessage := err.Error()
	finalizeCtx := vo.OnStart(ctx, enum.ConversationTraceStageFinalize,
		convCtx.ExecutionModeName(), &vo.StageInput{SummaryText: "正在收尾失败会话。"})

	// 向失败事件 + 流 Complete 信号；发送失败仅告警
	err = convCtx.Sink.Error(errorMessage, convCtx.ConversationId, convCtx.ExchangeId)
	if err != nil {
		logx.Warnf("发送失败事件失败, conversationId=%s, exchangeId=%d, error=%v", convCtx.ConversationId, convCtx.ExchangeId, err)
	}

	// 刷新 DebugTrace 的运行时统计
	c.refreshDebugTraceRuntimeStats(convCtx)

	// 组装失败 ChatExchange（保留已生成的答案/引用/思考链），调用 completeExchange 落库
	failExchange := c.buildCurrentChatExchange(convCtx, enum.ChatTurnStatusFailed, errorMessage)
	if err = c.completeExchange(ctx, failExchange); err == nil {
		// 落库成功：完成 finalize 追踪阶段，写入失败快照
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

// cleanup 清理会话运行时资源（管道、子协程、分布式锁、注册表）
func (c *Chain) cleanup(convCtx *Context) {
	c.releaseConversationLock(convCtx.LeaseKey)
	c.runtimeRegistry.Remove(convCtx.ConversationId, convCtx)
	convCtx.ReleaseResources()
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

// buildCurrentChatExchange 构建当前会话交互（exchange）
func (c *Chain) buildCurrentChatExchange(convCtx *Context, turnStatus int, errorMsg string) *entity.ChatExchange {
	return &entity.ChatExchange{
		ID:                  convCtx.ExchangeId,
		ConversationId:      convCtx.ConversationId,
		Question:            convCtx.Question,
		Answer:              convCtx.Answer(),
		ThinkingSteps:       common.ToJSONArray(convCtx.SnapshotThinkingSteps()),
		References:          common.ToJSONArray(convCtx.UniqueReferences()),
		UsedTools:           common.ToJSONArray(convCtx.SnapshotUsedTools()),
		DebugTrace:          convCtx.DebugTraceJSON(),
		TurnStatus:          turnStatus,
		ErrorMessage:        errorMsg,
		FirstResponseTimeMs: convCtx.FirstResponseTimeMs.Load(),
		TotalResponseTimeMs: time.Since(convCtx.StartTime).Milliseconds(),
	}
}
