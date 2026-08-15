package logic

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/duke-git/lancet/v2/strutil"

	"github.com/swiftbit/know-agent/common"
	"github.com/swiftbit/know-agent/common/logx"
	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/chat/adapter"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/conversation"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/executor"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/memory"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/preparation"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/recommend"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/aggregate"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/enum"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
	errorx "github.com/swiftbit/know-agent/internal/error"
	"github.com/swiftbit/know-agent/internal/svc"
)

const (
	chatRunningLeasePrefix        = "conversation:running:"
	chatRunningLeaseRenewInterval = 10 * time.Second
)

// ConversationLogicImpl 聊天业务逻辑实现
type ConversationLogicImpl struct {
	repo             adapter.ChatRepository
	preOrchestrator  preparation.ConversationPreOrchestrator
	renderer         adapter.PromptRenderer
	baseGateway      adapter.KnowledgeBaseGateway
	runtimeRegistry  *conversation.ChatRuntimeRegistry
	executorRegistry *executor.Registry
	recommender      recommend.QuestionRecommender
	memoryManager    memory.SessionMemoryManager
	distributedLock  adapter.DistributedLock
	checkPointStore  adapter.CheckPointStore
	chain            *conversation.Chain
	*options
}

var _ ConversationLogic = (*ConversationLogicImpl)(nil)

// NewConversationLogicImpl 创建聊天逻辑实例
func NewConversationLogicImpl(svcCtx *svc.ServiceContext,
	repo adapter.ChatRepository,
	executorRegistry *executor.Registry,
	baseGateway adapter.KnowledgeBaseGateway,
	preOrchestrator preparation.ConversationPreOrchestrator,
	renderer adapter.PromptRenderer,
	recommender recommend.QuestionRecommender,
	memoryManager memory.SessionMemoryManager,
	distributedLock adapter.DistributedLock,
	checkPointStore adapter.CheckPointStore,
) *ConversationLogicImpl {
	return &ConversationLogicImpl{
		repo:             repo,
		executorRegistry: executorRegistry,
		baseGateway:      baseGateway,
		preOrchestrator:  preOrchestrator,
		renderer:         renderer,
		runtimeRegistry:  &conversation.ChatRuntimeRegistry{},
		recommender:      recommender,
		memoryManager:    memoryManager,
		distributedLock:  distributedLock,
		checkPointStore:  checkPointStore,
		options: &options{
			historyPreviewTurns:    svcCtx.Config.Chat.Agent.HistoryPreviewTurns,
			maxModelCallsPerRun:    svcCtx.Config.Chat.Agent.MaxModelCallsPerRun,
			maxModelCallsPerThread: svcCtx.Config.Chat.Agent.MaxModelCallsPerThread,
			maxToolCallsPerRun:     svcCtx.Config.Chat.Agent.MaxToolCallsPerRun,
			maxToolCallsPerThread:  svcCtx.Config.Chat.Agent.MaxToolCallsPerThread,
		},
	}
}

// OpenConversationStream 打开会话流
func (c *ConversationLogicImpl) OpenConversationStream(ctx context.Context, sink adapter.Sink, cmd *vo.ChatCommand) (err error) {
	cmdJSON, _ := json.Marshal(cmd)
	logx.Infof("====== request 内容：%s", string(cmdJSON))

	leaseKey := chatRunningLeasePrefix + cmd.ConversationId
	defer func() {
		if err != nil {
			c.releaseConversationLock(leaseKey)
			logx.Errorf("会话启动失败, conversationId=%s, question=%s, err=%v",
				cmd.ConversationId, cmd.Question, err)
			if err = sink.Error(err.Error(), cmd.ConversationId, 0); err != nil {
				return
			}
		}
	}()

	// 获取分布式租约
	if err = c.distributedLock.TryLock(ctx, leaseKey); err != nil {
		return fmt.Errorf("该会话当前正在执行中，请稍后再试")
	}

	// 构建对话上下文
	convCtx, err := c.buildConversationContext(ctx, cmd, sink)
	if err != nil {
		return err
	}
	// 运行会话链
	return c.chain.Run(ctx, convCtx)
}

// StopConversation 停止会话
func (c *ConversationLogicImpl) StopConversation(ctx context.Context, conversationId string) (bool, string, error) {
	convCtx, ok := c.runtimeRegistry.Get(conversationId)
	if !ok {
		return false, "没有找到正在执行的会话", nil
	}
	responseMessage, stopped := c.chain.Stop(ctx, convCtx, "用户已停止生成")
	return stopped, responseMessage, nil
}

// GetSessionDetail 获取会话详情
func (c *ConversationLogicImpl) GetSessionDetail(ctx context.Context, conversationId string) (*aggregate.ConversationArchiveRecord, error) {
	record, err := c.repo.SelectSessionRecord(ctx, conversationId)
	if err != nil {
		return nil, err
	}
	record.MemorySummary, err = c.memoryManager.GetConversationSummary(ctx, conversationId)
	if err != nil {
		return nil, err
	}
	record.FillSummaryFields()

	return record, nil
}

// GetExchangeDetail 获取对话详情（含阶段追踪）
func (c *ConversationLogicImpl) GetExchangeDetail(ctx context.Context, conversationId string, exchangeId int64) (*entity.ChatExchange, []*entity.ChatExchangeTraceStage, error) {
	exchange, err := c.repo.SelectExchangeById(ctx, exchangeId)
	if err != nil {
		return nil, nil, err
	}

	stages, err := c.repo.SelectStages(ctx, conversationId, exchangeId)
	if err != nil {
		return nil, nil, err
	}
	return exchange, stages, nil
}

// ListSessions 获取会话列表（分页）
func (c *ConversationLogicImpl) ListSessions(ctx context.Context, pageNo, pageSize, chatMode, latestTurnStatus int, keyword string) ([]*aggregate.ConversationArchiveRecord, int64, error) {
	records, total, err := c.repo.ListSessionRecordPage(ctx, pageNo, pageSize, chatMode, latestTurnStatus, keyword)
	if err != nil {
		return nil, 0, err
	}
	for _, record := range records {
		record.FillSummaryFields()
		record.Exchanges = nil
	}

	return records, total, nil
}

// ResetConversation 重置会话：停止并清除所有相关落库数据
func (c *ConversationLogicImpl) ResetConversation(ctx context.Context, conversationId string) (*vo.ConversationReset, error) {
	stopResult := &vo.ConversationStop{}
	// 停止正在运行的会话
	if convCtx, ok := c.runtimeRegistry.Get(conversationId); ok {
		stopResult = c.stopTask(ctx, convCtx, "会话被重置")
	}

	var dialogueCount, exchangeCount int64
	var err error

	// 删除记忆摘要
	if err = c.memoryManager.DeleteConversationSummary(ctx, conversationId); err != nil {
		return nil, err
	}
	fn := func(txCtx context.Context) error {
		// 删除会话及关联 exchange
		dialogueCount, exchangeCount, err = c.repo.DeleteSession(ctx, conversationId)
		if err != nil {
			return err
		}
		// 删除阶段追踪
		if err = c.repo.DeleteStage(txCtx, conversationId); err != nil {
			return err
		}
		// 删除检索结果
		if err = c.repo.DeleteRetrievalResultsByConversationId(txCtx, conversationId); err != nil {
			return err
		}
		// 删除渠道执行记录
		return c.repo.DeleteChannelExecutionsByConversationId(txCtx, conversationId)
	}
	if err = c.repo.Do(ctx, fn); err != nil {
		return nil, err
	}
	// 删除检查点
	count, err := c.checkPointStore.Delete(ctx, conversationId)
	if err != nil {
		return nil, err
	}
	return &vo.ConversationReset{
		ConversationId:         conversationId,
		StoppedRunningTask:     stopResult.Stopped,
		RemovedDialogueCount:   int(dialogueCount),
		RemovedExchangeCount:   int(exchangeCount),
		RemovedCheckpointCount: count,
		Message:                "会话已重置",
	}, nil
}

// RebuildConversationSummary 重建会话摘要
func (c *ConversationLogicImpl) RebuildConversationSummary(ctx context.Context, conversationId string) (*entity.ChatMemorySummary, error) {
	summary, err := c.memoryManager.RebuildConversationSummary(ctx, conversationId)
	if err != nil {
		return nil, err
	}

	return summary, nil
}

// GetRetrievalResults 获取检索结果
func (c *ConversationLogicImpl) GetRetrievalResults(ctx context.Context, conversationId string, exchangeId int64) ([]*vo.ChatRetrievalResult, error) {
	return c.repo.SelectRetrievalResults(ctx, conversationId, exchangeId)
}

// GetChannelExecutions 获取渠道执行结果
func (c *ConversationLogicImpl) GetChannelExecutions(ctx context.Context, conversationId string, exchangeId int64) ([]*vo.ChatChannelExecution, error) {
	return c.repo.SelectChannelExecutions(ctx, conversationId, exchangeId)
}

// buildConversationContext 构建对话上下文：从 ChatCommand 提取问题与对话上下文，生成规范化的对话上下文
//
// 执行流程：
//  1. 规范化会话 ID：优先使用传入的 conversationId；为空时生成无连字符的 UUID
//  2. 构建初始计划（问题、会话 ID、聊天模式），并填充当前时间
//  3. 若命令中指定了文档 ID，则从可检索文档列表中查找并写入文档名与索引任务 ID；缺失则返回错误
func (c *ConversationLogicImpl) buildConversationContext(ctx context.Context, cmd *vo.ChatCommand, sink adapter.Sink) (*conversation.Context, error) {
	// 规范化会话 ID —— 空值时生成 UUID 作为新会话标识
	conversationId := utils.Trim(cmd.ConversationId)
	if conversationId == "" {
		conversationId = utils.GenerateUUIDWithoutHyphen()
	}

	selectionSnapshot, err := c.baseGateway.DetermineKnowledgeScope(ctx, cmd.ChatMode, cmd.KnowledgeBaseSelectionMode, cmd.SelectedKnowledgeBaseIds)
	if err != nil {
		return nil, err
	}
	// 构建启动计划，填充问题、会话 ID、聊天模式
	convCtx := &conversation.Context{
		ConversationId:                 conversationId,
		Question:                       cmd.Question,
		ChatMode:                       enum.ToChatQueryMode(cmd.ChatMode),
		KnowledgeBaseSelectionSnapshot: selectionSnapshot,
		Sink:                           sink,
	}
	if selectionSnapshot.SelectionModeName() == enum.KbSelectionModeNone {
		convCtx.ChatMode = enum.ChatQueryModeOpenChat
	}

	// 当指定文档 ID 时，验证该文档存在，并写入文档名与索引任务 ID
	if cmd.SelectedDocumentId != 0 {
		documents := utils.Ternary(cmd.KnowledgeBaseSelectionMode == enum.KbSelectionModeNone, nil, selectionSnapshot.AllowedDocuments)
		index := slices.IndexFunc(documents, func(doc *vo.DocumentMetadata) bool {
			return doc.DocumentId == cmd.SelectedDocumentId
		})
		// 指定的文档不存在或索引不可用，直接返回错误
		if index == -1 {
			return nil, errorx.ErrDocumentIndexUnavailable.Format(cmd.SelectedDocumentId)
		}
		convCtx.SelectedDocumentId = documents[index].DocumentId
		convCtx.SelectedDocumentName = documents[index].DocumentName
		convCtx.SelectedTaskId = documents[index].LastIndexTaskId
	}
	return convCtx, nil
}

// prepareExecutionPlan 准备执行计划
//
//	1.调用编排器准备基础计划（改写、路由、历史记忆等）
//	2.使用 prompt 模板构造 agent 问题（包含当前日期/上下文提示/历史摘要）
//	3. 根据所选文档刷新会话范围（在文档模式下）
//	4. 初始化调试轨迹
func (c *ConversationLogicImpl) prepareExecutionPlan(ctx context.Context, convCtx *conversation.Context) (*vo.ConversationExecutionPlan, error) {
	execPlan, err := c.preOrchestrator.Prepare(ctx, convCtx)
	if err != nil {
		logx.Warnf("执行计划准备失败, conversationId=%s, err=%v", convCtx.ConversationId, err)
		return nil, err
	}

	variables := map[string]any{
		"currentDateText":              execPlan.CurrentDateText,
		"requiresCurrentDateAnchoring": execPlan.RequiresCurrentDateAnchoring,
		"requiresRealTimeSearch":       execPlan.RequiresRealTimeSearch,
		"hasHistorySummary":            strutil.IsNotBlank(execPlan.HistorySummary),
		"historySummary":               execPlan.HistorySummary,
		"question":                     execPlan.OriginalQuestion,
	}
	agentQuestion, err := c.renderer.Render(enum.AgentQuestion, variables)
	if err != nil {
		return nil, err
	}
	execPlan.AgentQuestion = agentQuestion

	// 文档模式下若 selectedDocumentId 发生变化，则刷新会话范围
	if execPlan.SelectedDocumentId > 0 && execPlan.SelectedDocumentId != convCtx.SelectedDocumentId {
		dialogue := &entity.ChatDialogue{
			ConversationId:       convCtx.ConversationId,
			ChatMode:             execPlan.ChatMode,
			SelectedDocumentId:   execPlan.SelectedDocumentId,
			SelectedDocumentName: execPlan.SelectedDocumentName,
		}
		if err = c.repo.RefreshSessionScope(ctx, dialogue); err != nil {
			logx.Warnf("刷新会话范围失败, conversationId=%s, err=%v", convCtx.ConversationId, err)
			return nil, err
		}
	}

	debugTrace := vo.NewChatDebugTrace(execPlan)
	convCtx.DebugTrace.Store(debugTrace)
	convCtx.ExecutionPlan.Store(execPlan)

	return execPlan, nil
}

// stopTask 停止任务：原子切换状态 -> 发送停止事件 -> 落库 -> 清理
func (c *ConversationLogicImpl) stopTask(ctx context.Context, convCtx *conversation.Context, reason string) *vo.ConversationStop {
	if !convCtx.Finalized.CompareAndSwap(false, true) {
		return &vo.ConversationStop{
			ConversationId: convCtx.ConversationId,
			Stopped:        false,
			Message:        "会话已经结束",
		}
	}
	if curr, exists := c.runtimeRegistry.Get(convCtx.ConversationId); exists && curr != convCtx {
		return &vo.ConversationStop{
			ConversationId: convCtx.ConversationId,
			Stopped:        false,
			Message:        "会话已由新的执行接管",
		}
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
	err := convCtx.PublishStatus("⏹ " + reason)
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

	return &vo.ConversationStop{
		ConversationId: convCtx.ConversationId,
		Stopped:        true,
		Message:        responseMessage,
	}
}

// refreshDebugTraceRuntimeStats 刷新调试轨迹中的统计信息
func (c *ConversationLogicImpl) refreshDebugTraceRuntimeStats(convCtx *conversation.Context) {
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
func (c *ConversationLogicImpl) completeExchange(ctx context.Context, exchange *entity.ChatExchange) error {
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

// cleanup 清理会话运行时资源（管道、子协程、分布式锁、注册表）
func (c *ConversationLogicImpl) cleanup(convCtx *conversation.Context) {
	c.releaseConversationLock(convCtx.LeaseKey)
	c.runtimeRegistry.Remove(convCtx.ConversationId, convCtx)
	convCtx.ReleaseResources()
}

// releaseConversationLock 释放会话运行锁
func (c *ConversationLogicImpl) releaseConversationLock(leaseKey string) {
	err := c.distributedLock.Unlock(leaseKey)
	if err != nil && !errors.Is(err, errorx.ErrDistributedLockNotFound) {
		logx.Warnf("会话分布式锁释放失败, leaseKey=%s, err=%v", leaseKey, err)
	}
}

// buildCurrentChatExchange 构建当前会话交互（exchange）
func (c *ConversationLogicImpl) buildCurrentChatExchange(convCtx *conversation.Context, turnStatus int, errorMsg string) *entity.ChatExchange {
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
