package conversation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/duke-git/lancet/v2/slice"
	"github.com/duke-git/lancet/v2/strutil"
	"github.com/zeromicro/go-zero/core/logx"

	"github.com/swiftbit/know-agent/api/chat"
	"github.com/swiftbit/know-agent/common"
	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/chat/adapter"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/prompt"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/trace"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
	"github.com/swiftbit/know-agent/internal/domain/chat/support"
	doclog "github.com/swiftbit/know-agent/internal/domain/document/logic"
	dvo "github.com/swiftbit/know-agent/internal/domain/document/model/vo"
	errorx "github.com/swiftbit/know-agent/internal/error"
	"github.com/swiftbit/know-agent/internal/svc"
)

const (
	chatRunningLeasePrefix        = "conversation:running:"
	chatRunningLeaseRenewInterval = 10 * time.Second
	channelBufferSize             = 100
)

// LogicImpl 聊天业务逻辑实现
type LogicImpl struct {
	repo              adapter.ChatRepository
	orchestratorLogic logic.ChatPreparationOrchestratorLogic
	promptTempLogic   logic.PromptTemplateLogic
	tracer            *trace.ConversationTraceRecorder
	eventBuilder      *support.StreamEventBuilder
	runtimeRegistry   *support.ChatRuntimeRegistry
	executorRegistry  *ExecutorRegistry
	lifecycleLogic    doclog.LifecycleLogic
	recommendLogic    logic.RecommendationLogic
	memoryLogic       logic.SessionMemoryLogic
	distributedLock   adapter.DistributedLock
	options           *options
}

// NewChatLogic 创建聊天逻辑实例
func NewChatLogic(svcCtx *svc.ServiceContext,
	repo adapter.ChatRepository,
	executorRegistry *ExecutorRegistry,
	lifecycleLogic doclog.LifecycleLogic,
	orchestratorLogic logic.ChatPreparationOrchestratorLogic,
	promptTempLogic logic.PromptTemplateLogic,
	recommendLogic logic.RecommendationLogic,
	memoryLogic logic.SessionMemoryLogic,
	distributedLock adapter.DistributedLock,
) *LogicImpl {
	return &LogicImpl{
		repo:              repo,
		executorRegistry:  executorRegistry,
		orchestratorLogic: orchestratorLogic,
		promptTempLogic:   promptTempLogic,
		tracer:            trace.NewConversationTraceRecorder(repo),
		eventBuilder:      &support.StreamEventBuilder{},
		runtimeRegistry:   &support.ChatRuntimeRegistry{},
		lifecycleLogic:    lifecycleLogic,
		recommendLogic:    recommendLogic,
		memoryLogic:       memoryLogic,
		distributedLock:   distributedLock,
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
func (c *LogicImpl) OpenConversationStream(ctx context.Context, cmd *vo.ChatCommand) (stream <-chan string) {
	cmdJSON, _ := json.Marshal(cmd)
	logx.Infof("====== request 内容：%s", string(cmdJSON))

	leaseKey := chatRunningLeasePrefix + cmd.ConversationId
	defer func() {
		if r := recover(); r != nil {
			if err, ok := r.(error); ok {
				c.unlockConversationLock(ctx, leaseKey)
				logx.Errorf("会话启动失败, conversationId=%s, question=%s, err=%v",
					cmd.ConversationId, cmd.Question, err)
				stream = c.rejectStream(err.Error(), cmd.ConversationId, 0)
			}
		}
	}()

	// 获取分布式租约
	if err := c.distributedLock.TryLock(ctx, leaseKey); err != nil {
		panic(fmt.Errorf("该会话当前正在执行中，请稍后再试"))
	}

	// 构建启动计划
	plan, err := c.buildLaunchPlan(ctx, cmd)
	if err != nil {
		panic(err)
	}

	// 启动会话：创建 exchange + 注册运行上下文
	stream, err = c.bootstrapConversation(ctx, plan)
	if err != nil {
		panic(err)
	}

	return
}

// ListKnowledgeDocumentOptions 获取知识文档选项列表
func (c *LogicImpl) ListKnowledgeDocumentOptions(ctx context.Context) ([]*chat.KnowledgeDocumentOptionResp, error) {
	docs, err := c.lifecycleLogic.ListRetrievableDocuments(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*chat.KnowledgeDocumentOptionResp, 0, len(docs))
	for _, doc := range docs {
		if doc == nil {
			continue
		}
		result = append(result, &chat.KnowledgeDocumentOptionResp{
			DocumentId:   doc.DocumentId,
			DocumentName: doc.DocumentName,
		})
	}
	return result, nil
}

// StopConversation 停止会话
func (c *LogicImpl) StopConversation(ctx context.Context, conversationId string) (bool, string, error) {
	convCtx, ok := c.runtimeRegistry.Get(conversationId)
	if !ok {
		return false, "没有找到正在执行的会话", nil
	}
	stopTask := c.stopTask(ctx, convCtx, "用户已停止生成")
	return stopTask.Stopped, stopTask.Message, nil
}

// GetSession 获取会话详情
func (c *LogicImpl) GetSession(ctx context.Context, conversationId string) (*chat.ConversationSessionResp, error) {
	record, err := c.repo.SelectSessionRecord(ctx, conversationId)
	if err != nil {
		return nil, err
	}
	latestMessage := ""
	if len(record.Exchanges) > 0 {
		last := record.Exchanges[len(record.Exchanges)-1]
		if strutil.IsNotBlank(last.Answer) {
			latestMessage = last.Answer
		} else {
			latestMessage = last.Question
		}
	}
	return &chat.ConversationSessionResp{
		ConversationId: record.ConversationId,
		Title:          buildSessionTitle(record, latestMessage),
		LatestMessage:  latestMessage,
		CreateTime:     record.CreatedAt.Format(time.DateTime),
		UpdateTime:     record.UpdatedAt.Format(time.DateTime),
	}, nil
}

// GetExchangeDetail 获取对话详情（含阶段追踪）
func (c *LogicImpl) GetExchangeDetail(ctx context.Context, conversationId string, exchangeId int64) (*chat.ConversationExchangeDetailResp, error) {
	exchanges, err := c.repo.ListExchanges(ctx, conversationId)
	if err != nil {
		return nil, err
	}
	var matched *entity.ChatExchange
	for _, ex := range exchanges {
		if ex != nil && ex.ID == exchangeId {
			matched = ex
			break
		}
	}
	if matched == nil {
		return nil, fmt.Errorf("轮次不存在: %d", exchangeId)
	}

	stages, err := c.repo.SelectStages(ctx, conversationId, exchangeId)
	if err != nil {
		Warnf("获取阶段追踪失败, conversationId=%s, exchangeId=%d, err=%v", conversationId, exchangeId, err)
		stages = nil
	}
	stageResps := make([]*chat.ConversationTraceStageResp, 0, len(stages))
	for _, s := range stages {
		if s == nil {
			continue
		}
		stageResps = append(stageResps, &chat.ConversationTraceStageResp{
			StageId:       s.ID,
			StageCode:     s.StageCode,
			StageName:     s.StageName,
			ExecutionMode: s.ExecutionMode,
			ErrorMessage:  utils.PointerOrDefault(s.ErrorMessage, ""),
		})
	}

	thinking := jsonStrings(matched.ThinkingSteps)
	var refs []*chat.SearchReferenceResp
	if len(matched.ReferenceList) > 0 {
	}
	recommendations := jsonStrings(matched.RecommendationList)
	usedTools := jsonStrings(matched.UsedToolList)

	return &chat.ConversationExchangeDetailResp{
		ConversationId: conversationId,
		Exchange: &chat.ConversationExchangeResp{
			ExchangeId:          matched.ID,
			Question:            matched.Question,
			Answer:              matched.Answer,
			ThinkingSteps:       thinking,
			References:          refs,
			Recommendations:     recommendations,
			UsedTools:           usedTools,
			Status:              matched.TurnStatus,
			ErrorMessage:        matched.ErrorMessage,
			FirstResponseTimeMs: matched.FirstResponseTimeMs,
			TotalResponseTimeMs: matched.TotalResponseTimeMs,
			CreateTime:          matched.CreateTime.Format(time.DateTime),
			UpdateTime:          matched.UpdateTime.Format(time.DateTime),
		},
		StageTraces: stageResps,
	}, nil
}

// ListSessions 获取会话列表（分页）
func (c *LogicImpl) ListSessions(ctx context.Context, req *chat.ConversationSessionListReq) (*chat.ConversationSessionListResp, error) {
	pageNo := req.PageNo
	if pageNo <= 0 {
		pageNo = 1
	}
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 10
	}

	records, total, err := c.repo.ListSessionRecordPage(ctx, "", pageNo, pageSize, 0, 0)
	if err != nil {
		return nil, err
	}

	result := make([]*chat.ConversationSessionResp, 0, len(records))
	for _, r := range records {
		if r == nil {
			continue
		}
		latestMessage := ""
		if len(r.Exchanges) > 0 {
			last := r.Exchanges[len(r.Exchanges)-1]
			if strutil.IsNotBlank(last.Answer) {
				latestMessage = last.Answer
			} else {
				latestMessage = last.Question
			}
		}
		result = append(result, &chat.ConversationSessionResp{
			ConversationId: r.ConversationId,
			Title:          buildSessionTitle(r, latestMessage),
			LatestMessage:  latestMessage,
			CreateTime:     r.CreatedAt.Format(time.DateTime),
			UpdateTime:     r.UpdatedAt.Format(time.DateTime),
		})
	}
	return &chat.ConversationSessionListResp{
		PageNo:   pageNo,
		PageSize: pageSize,
		Total:    total,
		Records:  result,
	}, nil
}

// ResetConversation 重置会话：停止并清除所有相关落库数据
func (c *LogicImpl) ResetConversation(ctx context.Context, conversationId string) (*vo.ConversationReset, error) {
	stopResult := &vo.ConversationStop{}
	// 停止正在运行的会话
	if convCtx, ok := c.runtimeRegistry.Get(conversationId); ok {
		stopResult = c.stopTask(ctx, convCtx, "会话被重置")
	}

	// 删除会话及关联 exchange
	dialogueCount, exchangeCount, err := c.repo.DeleteSession(ctx, conversationId)
	if err != nil {
		return nil, err
	}

	// 删除记忆摘要
	_ = c.memoryLogic.DeleteConversationSummary(ctx, conversationId)

	return &vo.ConversationReset{
		ConversationId:         conversationId,
		StoppedRunningTask:     stopResult.Stopped,
		RemovedDialogueCount:   int(dialogueCount),
		RemovedExchangeCount:   int(exchangeCount),
		RemovedCheckpointCount: 0,
		Message:                "会话被重置",
	}, nil
}

// RebuildConversationSummary 重建会话摘要
func (c *LogicImpl) RebuildConversationSummary(ctx context.Context, conversationId string) (*chat.ConversationMemorySummaryResp, error) {
	sum, err := c.memoryLogic.RebuildConversationSummary(ctx, conversationId)
	if err != nil {
		return nil, err
	}
	text := ""
	if sum != nil {
		text = sum.SummaryText
		if text == "" && sum.Summary != nil {
			text = sum.Summary.Summary
		}
	}
	return &chat.ConversationMemorySummaryResp{
		ConversationId: conversationId,
		Summary:        text,
	}, nil
}

// GetRetrievalResults 获取检索结果
func (c *LogicImpl) GetRetrievalResults(ctx context.Context, conversationId string, exchangeId int64) ([]*chat.RetrievalResultResp, error) {
	// 注：检索结果观测属 domain/rag，此处保留扩展入口，避免耦合过重
	return []*chat.RetrievalResultResp{}, nil
}

// GetChannelExecutions 获取渠道执行结果
func (c *LogicImpl) GetChannelExecutions(ctx context.Context, conversationId string, exchangeId int64) ([]*chat.ChannelExecutionResp, error) {
	return []*chat.ChannelExecutionResp{}, nil
}

// GetStageBenchmarks 获取阶段基准
func (c *LogicImpl) GetStageBenchmarks(ctx context.Context) ([]*chat.StageBenchmarkResp, error) {
	return []*chat.StageBenchmarkResp{}, nil
}

// ---------------------------------------------------------------------------
// 内部实现：启动计划 / 启动会话 / 执行激活 / 执行计划 / 收尾
// ---------------------------------------------------------------------------

// buildLaunchPlan 构建启动计划
//
// 负责规范化：
//   - conversationId（空则生成 uuid）
//   - selectedDocument（仅在文档模式下必填且需命中可用文档）
func (c *LogicImpl) buildLaunchPlan(ctx context.Context, cmd *vo.ChatCommand) (*vo.StreamLaunchPlan, error) {
	// 规范化会话ID
	conversationId := strutil.Trim(cmd.ConversationId)
	if conversationId == "" {
		conversationId = utils.GenerateUUIDWithoutHyphen()
	}

	plan := &vo.StreamLaunchPlan{
		Question:       cmd.Question,
		ConversationId: conversationId,
		ChatMode:       cmd.ChatMode,
	}
	plan.FillCurrentDate()

	if cmd.SelectedDocumentId != 0 {
		documents, err := c.lifecycleLogic.ListRetrievableDocuments(ctx)
		if err != nil {
			return nil, err
		}
		selectedDocument, ok := slice.FindBy(documents, func(index int, doc *dvo.KnowledgeDocument) bool {
			return doc.DocumentId == cmd.SelectedDocumentId
		})
		if !ok {
			return nil, errorx.ErrDocumentIndexUnavailable.Format(cmd.SelectedDocumentId)
		}
		plan.SelectedDocumentId = selectedDocument.DocumentId
		plan.SelectedDocumentName = selectedDocument.DocumentName
		plan.SelectedTaskId = selectedDocument.LastIndexTaskId
	}
	return plan, nil
}

// bootstrapConversation 启动会话，创建对话记录并注册到运行注册表。若注册失败，说明会话正被其他执行接管，则拒绝。
func (c *LogicImpl) bootstrapConversation(ctx context.Context, plan *vo.StreamLaunchPlan) (<-chan string, error) {
	exchange, err := c.startExchange(ctx, plan)
	if err != nil {
		return nil, err
	}

	convCtx := c.buildConversationCtx(plan, exchange)
	cancelCtx, cancelFunc := context.WithCancel(ctx)
	convCtx.CancelFunc = cancelFunc

	if !c.runtimeRegistry.Register(convCtx) {
		// 已存在正在执行的会话，回写失败状态并拒绝
		failExchange := &entity.ChatExchange{
			ID:             exchange.ID,
			ConversationId: plan.ConversationId,
			TurnStatus:     vo.ChatTurnStatusFailed,
			ErrorMessage:   "该会话当前正在执行中，请稍后再试",
		}
		if err = c.completeExchange(ctx, failExchange); err != nil {
			logx.Errorf("保存 exchange 失败, conversationId=%s, err=%v", plan.ConversationId, err)
		}
		return nil, fmt.Errorf("该会话当前正在执行中，请稍后再试")
	}
	go c.activateGeneration(cancelCtx, convCtx)

	return convCtx.Channel, nil
}

// activateGeneration 激活生成逻辑
//
// 1. 启动租约续期
// 2. 发送 thinking 事件
// 3) 构建执行计划（改写 + 路由 + 记忆 + 文档）并写回上下文
// 4) 生成推荐追问（如启用）
// 执行过程中若触发停止或失败，则走失败/停止分支。
func (c *LogicImpl) activateGeneration(ctx context.Context, convCtx *vo.ConversationContext) {
	if convCtx.Finalized.Load() {
		return
	}

	// 启动租约续期
	go c.startLeaseRenewal(ctx, convCtx)
	if convCtx.Finalized.Load() {
		convCtx.ReleaseResources()
		return
	}
	resultCh, err := c.buildConversationExecution(convCtx)(ctx)
	if err != nil {
		logx.Errorf("执行出现异常, conversationId=%s, exchangeId=%d, err=%v", convCtx.ConversationId, convCtx.ExchangeId, err)
		c.finishWithFailure(ctx, convCtx, fmt.Errorf("执行出现异常: %v", err))
		return
	}

	for {
		select {
		case <-ctx.Done():
			c.stopTask(ctx, convCtx, "客户端已取消请求")
			return
		case chunk, ok := <-resultCh:
			if !ok {
				c.finishSuccessfully(ctx, convCtx)
				return
			}
			if err = c.emitModelChunk(convCtx, chunk); err != nil {
				logx.Errorf("执行出现异常, conversationId=%s, exchangeId=%d, err=%v", convCtx.ConversationId, convCtx.ExchangeId, err)
				c.finishWithFailure(ctx, convCtx, err)
				return
			}
			return
		}
	}
}

// buildConversationExecution 构建对话执行（执行计划构建的外层封装）
func (c *LogicImpl) buildConversationExecution(convCtx *vo.ConversationContext) func(ctx context.Context) (<-chan string, error) {
	return func(ctx context.Context) (<-chan string, error) {
		// 发送 "正在分析问题上下文" 事件
		thinkingEvent := c.eventBuilder.ThinkingWithMetadata("正在分析问题上下文。", convCtx.ConversationId, convCtx.ExchangeId)
		if err := support.SafeEmitNext(convCtx.Channel, thinkingEvent); err != nil {
			return nil, err
		}

		plan, err := c.prepareExecutionPlan(ctx, convCtx)
		if err != nil {
			return nil, err
		}
		convCtx.ExecutionPlan.Store(plan)

		// 发送执行计划信息（用于前端调试/感知）
		thinkingEvent = c.eventBuilder.ThinkingWithMetadata("上下文分析完成，已准备执行计划。", convCtx.ConversationId, convCtx.ExchangeId)
		if err = support.SafeEmitNext(convCtx.Channel, thinkingEvent); err != nil {
			return nil, err
		}
		executor, err := c.executorRegistry.Get(plan.Mode)
		if err != nil {
			return nil, err
		}
		resultCh, err := executor.Execute(ctx, convCtx)
		if err != nil {
			return nil, err
		}
		return resultCh, nil
	}
}

// startLeaseRenewal 启动租约续期，若续期失败则自动停止当前会话并终止生成
func (c *LogicImpl) startLeaseRenewal(ctx context.Context, convCtx *vo.ConversationContext) {
	ticker := time.NewTicker(chatRunningLeaseRenewInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			// 外部调用取消函数，停止续期
			return
		case <-ticker.C:
			if convCtx.Finalized.Load() {
				return
			}
			// 执行续期逻辑
			if err := c.distributedLock.Extend(ctx, convCtx.LeaseKey); err != nil {
				Warnf("会话租约续期失败，准备停止当前会话, conversationId=%s, exchangeId=%d",
					convCtx.ConversationId, convCtx.ExchangeId)
				c.stopTask(ctx, convCtx, "会话租约已失效，已停止生成")
				return
			}
		}
	}
}

// prepareExecutionPlan 准备执行计划
//
// 1) 调用编排器准备基础计划（改写、路由、历史记忆等）
// 2) 使用 prompt 模板构造 agent 问题（包含当前日期/上下文提示/历史摘要）
// 3) 根据所选文档刷新会话范围（在文档模式下）
// 4) 初始化调试轨迹
func (c *LogicImpl) prepareExecutionPlan(ctx context.Context, convCtx *vo.ConversationContext) (*vo.ConversationExecutionPlan, error) {
	execPlan, err := c.orchestratorLogic.Prepare(ctx, convCtx)
	if err != nil {
		Warnf("执行计划准备失败, conversationId=%s, err=%v", convCtx.ConversationId, err)
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
	agentQuestion, err := c.promptTempLogic.Render(prompt.AgentQuestion, variables)
	if err != nil {
		return nil, err
	}
	execPlan.AgentQuestion = agentQuestion

	// 文档模式下若 selectedDocumentId 发生变化，则刷新会话范围
	if execPlan.SelectedDocumentId > 0 && execPlan.SelectedDocumentId != convCtx.SelectedDocumentId {
		dialogue := &entity.ChatDialogue{
			ConversationId:       convCtx.ConversationId,
			ChatMode:             vo.ChatQueryModeValue(execPlan.ChatMode),
			SelectedDocumentId:   execPlan.SelectedDocumentId,
			SelectedDocumentName: execPlan.SelectedDocumentName,
		}
		if err = c.repo.RefreshSessionScope(ctx, dialogue); err != nil {
			Warnf("刷新会话范围失败, conversationId=%s, err=%v", convCtx.ConversationId, err)
			return nil, err
		}
	}

	debugTrace := vo.NewChatDebugTrace(execPlan)
	convCtx.DebugTrace.Store(debugTrace)
	convCtx.ExecutionPlan.Store(execPlan)

	return execPlan, nil
}

// emitModelChunk 发出模型输出块（text 事件），并更新首响应时间
func (c *LogicImpl) emitModelChunk(convCtx *vo.ConversationContext, chunk string) error {
	convCtx.WriteAnswerBuffer(chunk)
	convCtx.FirstResponseTimeMs.CompareAndSwap(0, time.Since(convCtx.StartTime).Milliseconds())
	textEvent := c.eventBuilder.TextWithMetadata(chunk, convCtx.ConversationId, convCtx.ExchangeId)
	return support.SafeEmitNext(convCtx.Channel, textEvent)
}

// stopTask 停止任务：原子切换状态 -> 发送停止事件 -> 落库 -> 清理
func (c *LogicImpl) stopTask(ctx context.Context, convCtx *vo.ConversationContext, reason string) *vo.ConversationStop {
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
		c.memoryLogic.RefreshConversationSummaryAsync(ctx, convCtx.ConversationId)
		c.cleanup(ctx, convCtx)
	}()

	// todo 中断 ReactAgent
	//        try {
	//         businessChatReactAgent.interrupt(taskInfo.runnableConfig());
	//     } catch (RuntimeException exception) {
	//         log.debug("中断 ReactAgent 时出现异常，继续释放资源", exception);
	//     }
	responseMessage := "已停止会话生成"
	finalizeStage, _ := c.tracer.StartStage(ctx, convCtx.Trace, vo.ConversationTraceStageFinalize,
		convCtx.ExecutionModeName(), "正在收尾停止中的会话。", nil)

	// 发送 status 事件
	statusEvent := c.eventBuilder.StatusWithMetadata("⏹ "+reason, convCtx.ConversationId, convCtx.ExchangeId)
	if err := support.SafeEmitNext(convCtx.Channel, statusEvent); err != nil {
		Warnf("发送停止事件失败, conversationId=%s, exchangeId=%d, err=%v", convCtx.ConversationId, convCtx.ExchangeId, err)
		responseMessage = "会话已停止，停止事件发送失败"
	}

	// 刷新调试轨迹统计
	c.refreshDebugTraceRuntimeStats(convCtx)

	// 构造停止态 exchange 并落库
	stopExchange := c.buildCurrentChatExchange(convCtx, vo.ChatTurnStatusStopped, reason)
	if err := c.completeExchange(ctx, stopExchange); err == nil {
		metadata := map[string]any{
			"finalStatus":  vo.ChatTurnStatusName(vo.ChatTurnStatusStopped),
			"reason":       reason,
			"answerLength": convCtx.AnswerLength(),
		}
		_ = c.tracer.CompleteStage(ctx, finalizeStage, "会话已按停止状态收尾", metadata)
	} else {
		logx.Errorf("停止会话落库失败, conversationId=%s, exchangeId=%d, err=%v",
			convCtx.ConversationId, convCtx.ExchangeId, err)
		responseMessage = "会话已停止，收尾落库失败"
		_ = c.tracer.FailStage(ctx, finalizeStage, "会话已按停止状态收尾", err, nil)
	}

	return &vo.ConversationStop{
		ConversationId: convCtx.ConversationId,
		Stopped:        true,
		Message:        responseMessage,
	}
}

// finishSuccessfully 以成功状态完成当前会话交互（exchange）
//
// 执行流程：
//  1. 原子检查 Finalized 标志（CAS：false → true），避免重复收尾
//  2. defer 中异步刷新会话摘要并执行清理（任何返回路径都会执行）
//  3. 启动 finalize 与 recommendation 两个追踪阶段（忽略其错误）
//  4. 生成推荐追问：需要澄清时返回澄清选项，否则由 recommendLogic 生成
//  5. 向客户端流补发引用事件、推荐事件，并发送流 Complete
//  6. 刷新 DebugTrace 运行时统计
//  7. 组装成功态 ChatExchange，调用 completeExchange 落库；根据落库结果完成或标记追踪阶段
func (c *LogicImpl) finishSuccessfully(ctx context.Context, convCtx *vo.ConversationContext) {
	// 原子检查 Finalized 标志（CAS），确保仅首次调用生效，避免重复收尾
	if !convCtx.Finalized.CompareAndSwap(false, true) {
		return
	}

	// defer 中刷新会话摘要 + 执行清理
	// 使用 defer 确保即便后续步骤出错，这两个清理动作也会执行
	defer func() {
		_ = recover()
		c.memoryLogic.RefreshConversationSummaryAsync(ctx, convCtx.ConversationId)
		c.cleanup(ctx, convCtx)
	}()

	// 从 convCtx 中取出当前答案与去重后的引用列表（供后续发送事件与落库使用）
	answer := convCtx.Answer()
	uniqueReferences := convCtx.UniqueReferences()

	// 启动 finalize 与 recommendation 两个追踪阶段（忽略 tracer 错误）
	finalizeStage, _ := c.tracer.StartStage(ctx, convCtx.Trace, vo.ConversationTraceStageFinalize,
		convCtx.ExecutionModeName(), "正在收尾已完成会话。", nil)

	recommendationsStage, _ := c.tracer.StartStage(ctx, convCtx.Trace, vo.ConversationTraceStageRecommendation,
		convCtx.ExecutionModeName(), "正在生成推荐追问。", nil)

	// 生成推荐追问
	// - 若本次交互是澄清（NeedClarification 为真），则直接使用澄清选项作为推荐
	// - 否则，拉取最近交互记录，由 recommendLogic 基于当前问答与历史生成推荐
	var recommendations []string
	if convCtx.NeedClarification() {
		recommendations = convCtx.ClarificationOptions()
	} else {
		recentExchanges := c.fetchRecentExchanges(ctx, convCtx.ConversationId, convCtx.ExchangeId)
		recommendations = c.recommendLogic.GenerateRecommendations(ctx, convCtx.Question, answer, recentExchanges, convCtx.Trace)
	}

	// 完成 recommendation 追踪阶段，并写入推荐数量快照
	snapshot := map[string]any{"recommendationCount": len(recommendations), "recommendations": recommendations}
	_ = c.tracer.CompleteStage(ctx, recommendationsStage, "推荐追问生成完成。", snapshot)

	// 向客户端流补发引用事件 / 推荐事件，最后发送流 Complete 信号
	if len(uniqueReferences) > 0 {
		referencesEvent := c.eventBuilder.ReferencesWithMetadata(uniqueReferences, convCtx.ConversationId, convCtx.ExchangeId)
		if err := support.SafeEmitNext(convCtx.Channel, referencesEvent); err != nil {
			Warnf("发送引用事件失败, conversationId=%s, exchangeId=%d, err=%v", convCtx.ConversationId, convCtx.ExchangeId, err)
		}
	}
	if len(recommendations) > 0 {
		recommendationsEvent := c.eventBuilder.RecommendationsWithMetadata(recommendations, convCtx.ConversationId, convCtx.ExchangeId)
		if err := support.SafeEmitNext(convCtx.Channel, recommendationsEvent); err != nil {
			Warnf("发送推荐事件失败, conversationId=%s, exchangeId=%d, err=%v", convCtx.ConversationId, convCtx.ExchangeId, err)
		}
	}

	// 刷新 DebugTrace 的运行时统计
	c.refreshDebugTraceRuntimeStats(convCtx)

	// 组装成功态 ChatExchange，调用 completeExchange 落库；并根据落库结果完成或标记 finalize 追踪阶段
	successExchange := c.buildCurrentChatExchange(convCtx, vo.ChatTurnStatusCompleted, "")
	successExchange.RecommendationList = common.ToJSONArray(recommendations)
	if err := c.completeExchange(ctx, successExchange); err == nil {
		// 落库成功：完成 finalize 追踪阶段，写入完成快照（含推荐、引用、答案长度）
		snapshot = map[string]any{
			"finalStatus":         vo.ChatTurnStatusName(vo.ChatTurnStatusCompleted),
			"recommendationCount": len(recommendations),
			"recommendations":     recommendations,
			"referenceCount":      len(uniqueReferences),
			"answerLength":        len(answer),
		}
		_ = c.tracer.CompleteStage(ctx, finalizeStage, "会话已按完成状态收尾。", snapshot)
	} else {
		// 落库失败：打印错误日志并标记 finalize 追踪阶段为失败
		logx.Errorf("成功会话收尾落库失败, conversationId=%s, exchangeId=%d, err=%v",
			convCtx.ConversationId, convCtx.ExchangeId, err)
		_ = c.tracer.FailStage(ctx, finalizeStage, "会话收尾落库失败", err, nil)
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
func (c *LogicImpl) finishWithFailure(ctx context.Context, convCtx *vo.ConversationContext, err error) {
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
		c.memoryLogic.RefreshConversationSummaryAsync(ctx, convCtx.ConversationId)
		c.cleanup(ctx, convCtx)
	}()

	// 启动 finalize 追踪阶段（失败时忽略 tracer 错误，不影响主流程）
	errorMessage := err.Error()
	finalizeStage, _ := c.tracer.StartStage(ctx, convCtx.Trace, vo.ConversationTraceStageFinalize,
		convCtx.ExecutionModeName(), "正在收尾失败会话。", nil)

	// 向失败事件 + 流 Complete 信号；发送失败仅告警
	errorEvent := c.eventBuilder.ErrorWithMetadata(errorMessage, convCtx.ConversationId, convCtx.ExchangeId)
	if err = support.SafeEmitNext(convCtx.Channel, errorEvent); err != nil {
		Warnf("发送失败事件失败, conversationId=%s, exchangeId=%d, error=%v", convCtx.ConversationId, convCtx.ExchangeId, err)
	}

	// 刷新 DebugTrace 的运行时统计
	c.refreshDebugTraceRuntimeStats(convCtx)

	// 组装失败 ChatExchange（保留已生成的答案/引用/思考链），调用 completeExchange 落库
	failExchange := c.buildCurrentChatExchange(convCtx, vo.ChatTurnStatusFailed, errorMessage)
	if err = c.completeExchange(ctx, failExchange); err == nil {
		// 落库成功：完成 finalize 追踪阶段，写入失败快照
		snapshot := map[string]any{
			"finalStatus":  vo.ChatTurnStatusName(vo.ChatTurnStatusFailed),
			"errorMessage": errorMessage,
			"answerLength": convCtx.AnswerLength(),
		}
		_ = c.tracer.CompleteStage(ctx, finalizeStage, "会话已按失败状态收尾。", snapshot)
	} else {
		// 落库失败：打印错误日志并标记 finalize 追踪阶段为失败
		logx.Errorf("失败会话落库失败, conversationId=%s, exchangeId=%d, err=%v",
			convCtx.ConversationId, convCtx.ExchangeId, err)
		_ = c.tracer.FailStage(ctx, finalizeStage, "失败态收尾失败。", err, nil)
	}
}

// refreshDebugTraceRuntimeStats 刷新调试轨迹中的统计信息
func (c *LogicImpl) refreshDebugTraceRuntimeStats(convCtx *vo.ConversationContext) {
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

// startExchange 开始新一轮会话交互（exchange）
func (c *LogicImpl) startExchange(ctx context.Context, plan *vo.StreamLaunchPlan) (*entity.ChatExchange, error) {
	// 构造对话实体（按 ConversationId 聚合整个会话），状态初始化为 Running
	dialogue := &entity.ChatDialogue{
		ConversationId:       plan.ConversationId,
		Question:             plan.Question,
		ChatMode:             vo.ChatQueryModeValue(plan.ChatMode),
		SelectedDocumentId:   plan.SelectedDocumentId,
		SelectedDocumentName: plan.SelectedDocumentName,
		SessionStatus:        vo.ChatSessionStatusRunning,
	}
	// 构造本轮交互实体，状态 Running（使用雪花 ID 保证全局唯一）
	chatExchange := &entity.ChatExchange{
		ID:             utils.GetSnowflakeNextID(),
		ConversationId: plan.ConversationId,
		Question:       plan.Question,
		TurnStatus:     vo.ChatTurnStatusRunning,
	}
	// 事务中原子执行：Upsert 对话 + 插入新交互
	startFn := func(txCtx context.Context) error {
		// Upsert：若对话记录已存在（同一 ConversationId）则更新，否则插入
		if err := c.repo.UpsertDialogue(txCtx, dialogue); err != nil {
			return err
		}
		// 插入本轮交互记录
		return c.repo.InsertExchange(txCtx, chatExchange)
	}
	if err := c.repo.Do(ctx, startFn); err != nil {
		return nil, err
	}
	return chatExchange, nil
}

// completeExchange 完成会话交互（exchange）
func (c *LogicImpl) completeExchange(ctx context.Context, exchange *entity.ChatExchange) error {
	completeFn := func(txCtx context.Context) error {
		// 更新交互记录（含答案、耗时、最终状态等，由调用方已在 exchange 对象中填充）
		if err := c.repo.UpdateExchangeById(txCtx, exchange); err != nil {
			return err
		}
		// 将对应会话的状态重置为 Idle（释放"运行中"标记）
		dialogue := &entity.ChatDialogue{SessionStatus: vo.ChatSessionStatusIdle}
		return c.repo.UpdateDialogueByConversationId(txCtx, dialogue)
	}
	return c.repo.Do(ctx, completeFn)
}

// cleanup 清理会话运行时资源（管道、子协程、分布式锁、注册表）
func (c *LogicImpl) cleanup(ctx context.Context, convCtx *vo.ConversationContext) {
	support.SafeEmitComplete(convCtx.Channel)
	convCtx.ReleaseResources()
	c.unlockConversationLock(ctx, convCtx.LeaseKey)
	c.runtimeRegistry.Remove(convCtx.ConversationId, convCtx)
}

// unlockConversationLock 释放会话运行锁
func (c *LogicImpl) unlockConversationLock(ctx context.Context, leaseKey string) {
	err := c.distributedLock.Unlock(ctx, leaseKey)
	if err != nil && errors.Is(err, errorx.ErrDistributedLockNotFound) {
		Warnf("会话分布式锁释放失败, leaseKey=%s, err=%v", leaseKey, err)
	}
}

// ---------------------------------------------------------------------------
// 辅助方法（构建上下文、流辅助、JSON 辅助等）
// ---------------------------------------------------------------------------

// buildConversationCtx 构建对话运行上下文
func (c *LogicImpl) buildConversationCtx(plan *vo.StreamLaunchPlan, exchange *entity.ChatExchange) *vo.ConversationContext {
	convCtx := vo.NewConversationContext(plan)
	convCtx.ExchangeId = exchange.ID
	convCtx.TraceId = utils.GenerateUUIDWithoutHyphen()
	convCtx.DebugTrace.Store(vo.NewChatDebugTrace(nil))
	convCtx.Trace = vo.NewConversationTrace(plan.ConversationId, exchange.ID, convCtx.TraceId)
	convCtx.Channel = make(chan string, channelBufferSize)
	convCtx.LeaseKey = chatRunningLeasePrefix + plan.ConversationId
	return convCtx
}

// rejectStream 生成一个仅含错误事件的只读流
func (c *LogicImpl) rejectStream(message, conversationId string, exchangeId int64) <-chan string {
	stream := make(chan string, 1)
	defer close(stream)
	stream <- c.eventBuilder.ErrorWithMetadata(message, conversationId, exchangeId)
	return stream
}

// fetchRecentExchanges 获取最近的历史轮次（排除当前）
func (c *LogicImpl) fetchRecentExchanges(ctx context.Context, conversationId string, excludeExchangeId int64) []*entity.ChatExchange {
	recent, err := c.repo.ListRecentExchanges(ctx, conversationId, c.options.historyPreviewTurns)
	if err != nil {
		Warnf("列出最近轮次失败, conversationId=%s, err=%v", conversationId, err)
		return nil
	}
	return slice.Filter(recent, func(_ int, ex *entity.ChatExchange) bool {
		return ex != nil && ex.ID != excludeExchangeId
	})
}

// buildCurrentChatExchange 构建当前会话交互（exchange）
func (c *LogicImpl) buildCurrentChatExchange(convCtx *vo.ConversationContext, turnStatus int, errorMsg string) *entity.ChatExchange {
	return &entity.ChatExchange{
		ID:                  convCtx.ExchangeId,
		ConversationId:      convCtx.ConversationId,
		Question:            convCtx.Question,
		Answer:              convCtx.Answer(),
		ThinkingSteps:       common.ToJSONArray(convCtx.SnapshotThinkingSteps()),
		ReferenceList:       common.ToJSONArray(convCtx.UniqueReferences()),
		UsedToolList:        common.ToJSONArray(convCtx.SnapshotUsedTools()),
		DebugTraceJson:      convCtx.DebugTraceJSON(),
		TurnStatus:          turnStatus,
		ErrorMessage:        errorMsg,
		FirstResponseTimeMs: convCtx.FirstResponseTimeMs.Load(),
		TotalResponseTimeMs: time.Since(convCtx.StartTime).Milliseconds(),
	}
}

// ---------------------------------------------------------------------------
// 纯函数/工具方法
// ---------------------------------------------------------------------------

// buildSessionTitle 为会话构造一个展示标题，取最新问题截断若干字符
func buildSessionTitle(record *vo.ConversationArchiveRecord, defaultText string) string {
	if record == nil {
		return ""
	}
	for i := len(record.Exchanges) - 1; i >= 0; i-- {
		ex := record.Exchanges[i]
		if ex != nil && strutil.IsNotBlank(ex.Question) {
			q := ex.Question
			if len(q) > 30 {
				return q[:30]
			}
			return q
		}
	}
	if len(defaultText) > 30 {
		return defaultText[:30]
	}
	return defaultText
}

// jsonStrings 把 common.JSONArray 解析为字符串切片；若失败返回空切片
func jsonStrings(raw any) []string {
	if raw == nil {
		return []string{}
	}
	switch v := raw.(type) {
	case []byte:
		if len(v) == 0 {
			return []string{}
		}
		var out []string
		if err := json.Unmarshal(v, &out); err == nil {
			return out
		}
		return []string{}
	case string:
		if v == "" {
			return []string{}
		}
		var out []string
		if err := json.Unmarshal([]byte(v), &out); err == nil {
			return out
		}
		return []string{}
	default:
		return []string{}
	}
}

func Warnf(format string, args ...any) {
	logx.Alert(fmt.Sprintf(format, args...))
}
