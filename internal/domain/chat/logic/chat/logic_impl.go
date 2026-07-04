package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/duke-git/lancet/v2/slice"
	"github.com/duke-git/lancet/v2/strutil"
	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/logx"

	"github.com/swiftbit/know-agent/api/chat"
	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/chat/adapter"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/prompt"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/trace"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
	"github.com/swiftbit/know-agent/internal/domain/chat/support"
	kllg "github.com/swiftbit/know-agent/internal/domain/knowledge/logic"
	klvo "github.com/swiftbit/know-agent/internal/domain/rag/model/vo"
	errorx "github.com/swiftbit/know-agent/internal/error"
	"github.com/swiftbit/know-agent/internal/svc"
)

const (
	chatRunningLeasePrefix        = "chat:running:"
	chatRunningLeaseTTL           = 30 * time.Second
	chatRunningLeaseRenewInterval = 10 * time.Second
	channelBufferSize             = 100
)

// LogicImpl 聊天业务逻辑实现
type LogicImpl struct {
	svcCtx             *svc.ServiceContext
	repo               adapter.ChatRepository
	orchestratorLogic  logic.ChatPreparationOrchestratorLogic
	promptTempLogic    logic.PromptTemplateLogic
	tracer             *trace.ConversationTraceRecorder
	streamEventBuilder *support.StreamEventBuilder
	runtimeRegistry    *support.ChatRuntimeRegistry
	knowledgeLogic     kllg.KnowledgeLogic
	distributedLock    adapter.DistributedLock
}

// NewChatLogic 创建聊天逻辑实例
func NewChatLogic(repo adapter.ChatRepository, knowledgeLogic kllg.KnowledgeLogic,
	orchestratorLogic logic.ChatPreparationOrchestratorLogic, promptTempLogic logic.PromptTemplateLogic,
	distributedLock adapter.DistributedLock) *LogicImpl {
	return &LogicImpl{
		repo:               repo,
		orchestratorLogic:  orchestratorLogic,
		promptTempLogic:    promptTempLogic,
		tracer:             trace.NewConversationTraceRecorder(repo),
		streamEventBuilder: &support.StreamEventBuilder{},
		runtimeRegistry:    &support.ChatRuntimeRegistry{},
		knowledgeLogic:     knowledgeLogic,
		distributedLock:    distributedLock,
	}
}

// OpenConversationStream 打开会话流
func (c *LogicImpl) OpenConversationStream(ctx context.Context, cmd *vo.ChatCommand) <-chan string {
	cmdJSON, _ := json.Marshal(cmd)
	logx.Infof("======request内容：%s", string(cmdJSON))

	leaseKey := chatRunningLeasePrefix + cmd.ConversationId
	if err := c.distributedLock.TryLock(ctx, leaseKey); err != nil {
		return c.rejectStream("当前会话正在执行中，请稍后再试", cmd.ConversationId, 0)
	}
	defer func() {
		if err := c.distributedLock.Unlock(ctx, leaseKey); err != nil {
			logx.Alert(fmt.Sprintf("锁 %s 释放失败: %s", leaseKey, err.Error()))
		}
	}()

	launchPlan, err := c.buildLaunchPlan(ctx, cmd)
	if err != nil {
		logx.Errorf("会话启动失败, conversationId=%s, question=%s, err=%s", cmd.ConversationId, cmd.Question, err.Error())
		return c.rejectStream(err.Error(), cmd.ConversationId, 0)
	}
	stream, err := c.bootstrapConversation(ctx, launchPlan)
	if err != nil {
		return c.rejectStream(err.Error(), cmd.ConversationId, 0)
	}
	return stream
}

// ListKnowledgeDocumentOptions 获取知识文档选项列表
func (c *LogicImpl) ListKnowledgeDocumentOptions(ctx context.Context) ([]*chat.KnowledgeDocumentOptionResp, error) {
	return []*chat.KnowledgeDocumentOptionResp{
		{DocumentId: 1, DocumentName: "产品手册.pdf"},
		{DocumentId: 2, DocumentName: "技术文档.docx"},
		{DocumentId: 3, DocumentName: "用户指南.txt"},
	}, nil
}

// StopConversation 停止会话
func (c *LogicImpl) StopConversation(ctx context.Context, conversationId string) (*chat.ConversationStopResp, error) {
	convCtx, ok := c.runtimeRegistry.Get(conversationId)
	if !ok {
		return &chat.ConversationStopResp{Success: false}, fmt.Errorf("没有找到正在执行的会话")
	}

	return &chat.ConversationStopResp{Success: true}, nil
}

// GetSession 获取会话详情
func (c *LogicImpl) GetSession(ctx context.Context, conversationId string) (*chat.ConversationSessionResp, error) {
	return &chat.ConversationSessionResp{
		ConversationId: conversationId,
		Title:          "测试会话",
		LatestMessage:  "你好，有什么可以帮助你的？",
		CreateTime:     time.Now().Format(time.DateTime),
		UpdateTime:     time.Now().Format(time.DateTime),
	}, nil
}

// GetExchangeDetail 获取对话详情
func (c *LogicImpl) GetExchangeDetail(ctx context.Context, conversationId, exchangeId string) (*chat.ConversationExchangeDetailResp, error) {
	return &chat.ConversationExchangeDetailResp{}, nil
}

// ListSessions 获取会话列表
func (c *LogicImpl) ListSessions(ctx context.Context, req *chat.ConversationSessionListReq) (*chat.ConversationSessionListResp, error) {
	pageNo := req.PageNo
	if pageNo <= 0 {
		pageNo = 1
	}
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 10
	}

	return &chat.ConversationSessionListResp{
		PageNo:   pageNo,
		PageSize: pageSize,
		Total:    100,
		Records: []*chat.ConversationSessionResp{
			{
				ConversationId: uuid.New().String(),
				Title:          "会话1",
				LatestMessage:  "消息内容1",
				CreateTime:     time.Now().Format(time.DateTime),
				UpdateTime:     time.Now().Format(time.DateTime),
			},
			{
				ConversationId: uuid.New().String(),
				Title:          "会话2",
				LatestMessage:  "消息内容2",
				CreateTime:     time.Now().Format(time.DateTime),
				UpdateTime:     time.Now().Format(time.DateTime),
			},
		},
	}, nil
}

// ResetConversation 重置会话
func (c *LogicImpl) ResetConversation(ctx context.Context, conversationId string) (*chat.ConversationResetResp, error) {
	return &chat.ConversationResetResp{Success: true}, nil
}

// RebuildConversationSummary 重建会话摘要
func (c *LogicImpl) RebuildConversationSummary(ctx context.Context, conversationId string) (*chat.ConversationMemorySummaryResp, error) {
	return &chat.ConversationMemorySummaryResp{
		ConversationId: conversationId,
		Summary:        "这是会话的摘要内容",
	}, nil
}

// GetRetrievalResults 获取检索结果
func (c *LogicImpl) GetRetrievalResults(ctx context.Context, conversationId string, exchangeId int64) ([]*chat.RetrievalResultResp, error) {
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

func (c *LogicImpl) buildLaunchPlan(ctx context.Context, cmd *vo.ChatCommand) (*vo.StreamLaunchPlan, error) {
	launchPlan := &vo.StreamLaunchPlan{
		Question:       cmd.Question,
		ConversationId: cmd.ConversationId,
		ChatMode:       cmd.ChatMode,
	}
	launchPlan.FillCurrentDate()
	if cmd.SelectedDocumentId != 0 {
		documents, err := c.knowledgeLogic.ListRetrievableDocuments(ctx)
		if err != nil {
			return nil, err
		}
		selectedDocument, ok := slice.FindBy(documents, func(index int, doc *klvo.KnowledgeDocument) bool {
			return doc.DocumentId == cmd.SelectedDocumentId
		})
		if !ok {
			return nil, errorx.ErrDocumentIndexUnavailable.Format(cmd.SelectedDocumentId)
		}
		launchPlan.SelectedDocumentId = selectedDocument.DocumentId
		launchPlan.SelectedDocumentName = selectedDocument.DocumentName
		launchPlan.SelectedTaskId = selectedDocument.LastIndexTaskId
	}
	return launchPlan, nil
}

// bootstrapConversation 启动会话
func (c *LogicImpl) bootstrapConversation(ctx context.Context, launchPlan *vo.StreamLaunchPlan) (<-chan string, error) {
	dialogue := &entity.ChatDialogue{
		ConversationId:       launchPlan.ConversationId,
		ChatMode:             launchPlan.ChatMode.Value(),
		SelectedDocumentId:   launchPlan.SelectedDocumentId,
		SelectedDocumentName: launchPlan.SelectedDocumentName,
		Question:             launchPlan.Question,
	}
	exchange, err := c.repo.StartExchange(ctx, dialogue)
	if err != nil {
		return nil, err
	}

	// 创建对话上下文信息
	convCtx := c.buildConversationCtx(launchPlan, exchange)
	cancelCtx, cancelFunc := context.WithCancel(ctx)
	convCtx.CancelExecute = cancelFunc

	// 注册到运行时注册表
	if !c.runtimeRegistry.Register(convCtx) {
		failChatExchange := &entity.ChatExchange{
			ID:             exchange.ID,
			ConversationId: launchPlan.ConversationId,
			TurnStatus:     vo.ChatTurnStatusFailed,
			ErrorMessage:   "当前会话正在执行中，请稍后再试",
		}
		if err = c.repo.CompleteExchange(ctx, failChatExchange); err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("当前会话正在执行中，请稍后再试")
	}
	go func() {
		c.activateGeneration(cancelCtx, convCtx)
	}()

	return convCtx.Channel, nil
}

func (c *LogicImpl) executeConversation(convCtx *vo.ConversationContext, stream chan<- string) {
	thinkingMsg := c.streamEventBuilder.ThinkingWithMetadata("正在分析问题上下文。", convCtx.EventMetadata)
	stream <- thinkingMsg

	thinkingMsg = c.streamEventBuilder.ThinkingWithMetadata("正在检索相关知识...", convCtx.EventMetadata)
	stream <- thinkingMsg

	time.Sleep(500 * time.Millisecond)

	answer := "这是模拟的回答内容。您的问题是：" + convCtx.Question
	for i := 0; i < len(answer); i += 5 {
		end := i + 5
		if end > len(answer) {
			end = len(answer)
		}
		textMsg := c.streamEventBuilder.TextWithMetadata(answer[i:end], convCtx.EventMetadata)
		stream <- textMsg
		convCtx.AnswerBuffer.WriteString(answer[i:end])
		time.Sleep(100 * time.Millisecond)
	}

	references := []*vo.SearchReference{
		{
			ReferenceId:  "ref-001",
			SourceType:   "document",
			Title:        "相关文档标题",
			Snippet:      "这是文档中的相关片段内容...",
			DocumentId:   1,
			DocumentName: "示例文档.pdf",
			Score:        0.85,
		},
	}
	refMsg := c.streamEventBuilder.ReferencesWithMetadata(references, convCtx.EventMetadata)
	stream <- refMsg

	recommendations := []string{
		"您想了解更多相关信息吗？",
		"是否需要深入探讨这个话题？",
	}
	recMsg := c.streamEventBuilder.RecommendationsWithMetadata(recommendations, convCtx.EventMetadata)
	stream <- recMsg
}

// activateGeneration 激活生成逻辑
func (c *LogicImpl) activateGeneration(ctx context.Context, convCtx *vo.ConversationContext) {
	defer func() {
		if r := recover(); r != nil {
			if err, ok := r.(error); ok {
				c.finishWithFailure(ctx, convCtx, err)
			} else {
				c.finishWithFailure(ctx, convCtx, fmt.Errorf("execution panic: %v", r))
			}
		}
	}()
	if convCtx.Finalized.Load() {
		return
	}

	go c.startLeaseRenewal(ctx, convCtx)

	c.buildConversationExecution(convCtx)(ctx)
	if convCtx.Finalized.Load() {
		return
	}

	// 异步执行
	// go func() {
	// 	defer func() {
	// 		if r := recover(); r != nil {
	// 			c.finishWithFailure(ctx, convCtx, fmt.Errorf("execution panic: %v", r))
	// 		}
	// 	}()
	// 	execFunc(ctx)
	// }()
}

// startLeaseRenewal 启动租约续期
func (c *LogicImpl) startLeaseRenewal(ctx context.Context, convCtx *vo.ConversationContext) {
	ticker := time.NewTicker(chatRunningLeaseRenewInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			// 外部调用取消函数，停止续期
			return
		case <-ticker.C:
			// 执行续期逻辑
			if err := c.distributedLock.Extend(ctx, convCtx.LeaseKey); err != nil {
				logx.Alert(fmt.Sprintf("会话租约续期失败，准备停止当前会话, conversationId=%s, exchangeId=%d",
					convCtx.ConversationId, convCtx.ExchangeId))
				c.stopTask(ctx, convCtx, "会话租约已失效，已停止生成")
				return
			}
		}
	}
}

// buildConversationExecution 构建对话执行函数
func (c *LogicImpl) buildConversationExecution(convCtx *vo.ConversationContext) func(ctx context.Context) {
	return func(ctx context.Context) {
		// 发送“正在分析问题上下文”事件
		metadata := c.streamEventBuilder.ThinkingWithMetadata("正在分析问题上下文。", convCtx.ConversationId, convCtx.ExchangeId)
		if err := support.SafeEmitNext(convCtx.Channel, metadata); err != nil {
			panic(err)
		}

		// 准备执行计划
		plan, err := c.prepareExecutionPlan(ctx, convCtx)
		if err != nil {
			panic(err)
		}

		// todo 获取执行器
		executor := c.conversationExecutorRegistry.Get(plan.Mode)
		if executor == nil {
			c.finishWithFailure(ctx, convCtx, fmt.Errorf("no executor for mode %v", plan.Mode))
			return
		}

		// 执行（返回一个 channel 用于流式输出）
		chunkChan := executor.Execute(ctx, convCtx)
		for chunk := range chunkChan {
			select {
			case <-ctx.Done():
				return
			default:
				c.emitModelChunk(convCtx, chunk)
			}
		}
	}
}

// emitModelChunk 发送模型输出块
func (c *LogicImpl) emitModelChunk(convCtx *vo.ConversationContext, chunk string) {
	convCtx.AnswerBuffer.WriteString(chunk)
	convCtx.FirstResponseTimeMs.CompareAndSwap(0, time.Since(convCtx.StartTime).Milliseconds())

	support.SafeEmitNext(convCtx.Channel, c.streamEventBuilder.TextWithMetadata(chunk, convCtx.ConversationId, convCtx.ExchangeId))
}

// prepareExecutionPlan 准备执行计划
func (c *LogicImpl) prepareExecutionPlan(ctx context.Context, convCtx *vo.ConversationContext) (*vo.ConversationExecutionPlan, error) {
	// 1. 调用编排器准备基础计划
	execPlan, err := c.orchestratorLogic.Prepare(ctx, convCtx)
	if err != nil {
		logx.Errorf("准备执行计划失败, conversationId=%s, exchangeId=%d, error=%v",
			convCtx.ConversationId, convCtx.ExchangeId, err)
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
		logx.Errorf("渲染问题失败, conversationId=%s, exchangeId=%d, error=%v",
			convCtx.ConversationId, convCtx.ExchangeId, err)
		return nil, err
	}

	// 2. 设置 Agent 问题
	execPlan.AgentQuestion = agentQuestion

	if execPlan.SelectedDocumentId > 0 && execPlan.SelectedDocumentId != convCtx.SelectedDocumentId {
		dialogue := &entity.ChatDialogue{
			ConversationId:       convCtx.ConversationId,
			ChatMode:             execPlan.ChatMode.Value(),
			SelectedDocumentId:   execPlan.SelectedDocumentId,
			SelectedDocumentName: execPlan.SelectedDocumentName,
		}
		// 刷新会话范围
		if err = c.repo.RefreshSessionScope(ctx, dialogue); err != nil {
			logx.Errorf("刷新会话范围失败, conversationId=%s, exchangeId=%d, error=%v",
				convCtx.ConversationId, convCtx.ExchangeId, err)
			return nil, err
		}

		// todo 更新上下文
		// putContextIfNotNull(convCtx.RunnableConfig.Context, "selectedDocumentId", execPlan.SelectedDocumentId)
		// putContextIfNotBlank(convCtx.RunnableConfig.Context, "selectedDocumentName", execPlan.SelectedDocumentName)
		// putContextIfNotNull(convCtx.RunnableConfig.Context, "selectedTaskId", execPlan.SelectedTaskId)
	}

	convCtx.ExecutionPlan.Store(execPlan)
	debugTrace := vo.NewChatDebugTrace(execPlan)
	convCtx.DebugTrace.Store(debugTrace)
	// convCtx.RunnableConfig.Context["debugTrace"] = debugTrace
	return execPlan, nil
}

// stopTask 停止任务
func (c *LogicImpl) stopTask(ctx context.Context, convCtx *vo.ConversationContext, reason string) *vo.ConversationStop {
	// 原子地将 finalized 从 false 设置为 true，若已经是 true 则直接返回“会话已经结束”
	if !convCtx.Finalized.CompareAndSwap(false, true) {
		return &vo.ConversationStop{
			ConversationId: convCtx.ConversationId,
			Message:        "会话已经结束",
		}
	}

	// 检查当前运行时注册表中的任务是否仍是当前 convCtx，防止被新任务接管
	if currentTask, exists := c.runtimeRegistry.Get(convCtx.ConversationId); exists && currentTask != convCtx {
		return &vo.ConversationStop{
			ConversationId: convCtx.ConversationId,
			Message:        "会话已由新的执行接管",
		}
	}

	// 中断 ReactAgent
	if businessChatReactAgent != nil {
		// 使用带超时的 context 避免永久阻塞
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := businessChatReactAgent.Interrupt(ctx, convCtx.RunnableConfig); err != nil {
			logx.Debugf("中断 ReactAgent 时出现异常，继续释放资源: %v", err)
		}
	}

	// 释放执行任务
	if execCancel := convCtx.CancelExecute; execCancel != nil {
		execCancel()
		convCtx.CancelExecute = nil // 避免重复调用
	}

	// 准备停止响应的消息
	responseMessage := "已停止会话生成"
	execPlan := convCtx.ExecutionPlan.Load()
	modeName := ""
	if execPlan != nil {
		modeName = execPlan.Mode.Name()
	}

	// 开始追踪收尾阶段
	finalizeStage, err := c.tracer.StartStage(ctx, convCtx.Trace, vo.ConversationTraceStageFinalize, modeName, "正在收尾停止中的会话。", nil)
	if err != nil {
		logx.Errorf("开始追踪收尾阶段失败, conversationId=%s, exchangeId=%d, error=%v",
			convCtx.ConversationId, convCtx.ExchangeId, err)
	}

	// 构造状态事件消息：⏹ + reason
	statusMsg := c.streamEventBuilder.StatusWithMetadata("⏹ "+reason, convCtx.ConversationId, convCtx.ExchangeId)
	// 调用 safeEmit，将 statusMsg 作为事件写入 chan
	if err = support.SafeEmitNext(convCtx.Channel, statusMsg); err != nil {
		logx.Errorf("发送停止事件失败, conversationId=%s, exchangeId=%d, error=%v",
			convCtx.ConversationId, convCtx.ExchangeId, err)
		responseMessage = "会话已停止，停止事件发送失败"
	}

	// 辅助函数：安全完成 sink
	safeCompleteSink := func() {
		defer func() {
			if r := recover(); r != nil {
				logx.Errorf("关闭 SSE 流 panic: %v", r)
			}
		}()
		support.SafeEmitComplete(convCtx.Channel)
	}

	// 最终收尾：关闭 sink、落库、清理
	func() {
		defer func() {
			// 确保 finally 中的清理总会执行
			safeRefreshConversationSummary(convCtx.ConversationId)
			cleanup(convCtx)
		}()

		// 关闭 sink
		func() {
			defer func() {
				if r := recover(); r != nil {
					logx.Errorf("关闭停止中的 SSE 流失败, conversationId=%s, exchangeId=%d, error=%v",
						convCtx.ConversationId, convCtx.ExchangeId, r)
				}
			}()
			safeCompleteSink()
		}()

		// 刷新追踪运行时统计
		refreshDebugTraceRuntimeStats(convCtx)

		exchange := &entity.ChatExchange{
			ID:             convCtx.ExchangeId,
			ConversationId: convCtx.ConversationId,
			Question:       convCtx.Question,
			// Answer:              convCtx.GetAnswer(),
			ThinkingSteps:       nil,
			ReferenceList:       nil,
			RecommendationList:  nil,
			UsedToolList:        nil,
			DebugTraceJson:      "",
			TurnStatus:          0,
			ErrorMessage:        "",
			FirstResponseTimeMs: 0,
			TotalResponseTimeMs: 0,
		}
		// 落库完整 exchange 信息
		err = c.repo.CompleteExchange(ctx, exchange)
		if err != nil {
			logx.Errorf("停止会话落库失败, conversationId=%s, exchangeId=%d, error=%v",
				convCtx.ConversationId, convCtx.ExchangeId, err)
			responseMessage = "会话已停止，收尾落库失败"

			if err = c.tracer.FailStage(ctx, finalizeStage, "停止态收尾失败。", err, nil); err != nil {
				logx.Errorf("追踪收尾阶段失败, conversationId=%s, exchangeId=%d, error=%v",
					convCtx.ConversationId, convCtx.ExchangeId, err)
				return
			}
		} else {
			// TODO: 调用 traceRecorder.CompleteStage
			// convCtx.TraceRecorder.CompleteStage(finalizeStage, "会话已按停止状态收尾。", map[string]interface{}{
			//     "finalStatus": ChatTurnStatusStopped,
			//     "reason":      reason,
			//     "answerLength": len(convCtx.AnswerBuffer),
			// })
		}
	}()

	return &vo.ConversationStop{
		ConversationId: convCtx.ConversationId,
		Stopped:        true,
		Message:        responseMessage,
	}
}

// finishSuccessfully 成功完成处理
func (c *LogicImpl) finishSuccessfully(ctx context.Context, convCtx *vo.ConversationContext) {
	if !convCtx.Finalized.CompareAndSwap(false, true) {
		return
	}
	anwser := convCtx.AnswerBuffer.String()
	uniqueReferences := c.deduplicateReferences(convCtx.References.Snapshot())

	// 追踪收尾阶段
	var finalizeStage interface{}
	if convCtx.TraceRecorder != nil {
		modeName := ""
		if convCtx.ExecutionPlan != nil {
			modeName = convCtx.ExecutionPlan.Mode.String()
		}
		finalizeStage = convCtx.TraceRecorder.StartStage(trace.StageCodeFinalize, modeName, "正在收尾已完成会话。", nil)
	}
	var recommendationStage interface{}
	if convCtx.TraceRecorder != nil {
		modeName := ""
		if convCtx.ExecutionPlan != nil {
			modeName = convCtx.ExecutionPlan.Mode.String()
		}
		recommendationStage = convCtx.TraceRecorder.StartStage(trace.StageCodeRecommendation, modeName, "正在生成推荐追问。", nil)
	}

	var recommendations []string
	if convCtx.ExecutionPlan != nil && convCtx.ExecutionPlan.Mode == execution.ModeClarification {
		recommendations = convCtx.ExecutionPlan.ClarificationOptions
		if recommendations == nil {
			recommendations = []string{}
		}
	} else {
		recommendations = c.recommendationService.GenerateRecommendations(
			ctx,
			convCtx.Question,
			answer,
			s.historicalRecentExchanges(convCtx),
			convCtx.TraceRecorder,
		)
	}
	if convCtx.TraceRecorder != nil && recommendationStage != nil {
		convCtx.TraceRecorder.CompleteStage(recommendationStage, "推荐追问生成完成。", map[string]interface{}{
			"recommendationCount": len(recommendations),
			"recommendations":     recommendations,
		})
	}

	// 发送引用和推荐
	defer func() {
		if r := recover(); r != nil {
			logx.Warnf("补发引用或推荐事件失败, conversationId=%s, exchangeId=%d, error=%v",
				convCtx.ConversationId, convCtx.ExchangeId, r)
		}
	}()
	if len(uniqueReferences) > 0 {
		s.safeEmit(convCtx.Sink, s.streamEventWriter.References(uniqueReferences, convCtx.EventMetadata))
	}
	if len(recommendations) > 0 {
		s.safeEmit(convCtx.Sink, s.streamEventWriter.Recommendations(recommendations, convCtx.EventMetadata))
	}

	// 最终收尾：关闭 sink，落库，刷新摘要，清理
	defer func() {
		s.safeRefreshConversationSummary(convCtx.ConversationId)
		s.cleanup(convCtx)
	}()
	defer func() {
		if r := recover(); r != nil {
			logx.Warnf("关闭成功完成的 SSE 流失败, conversationId=%s, exchangeId=%d, error=%v",
				convCtx.ConversationId, convCtx.ExchangeId, r)
		}
	}()
	s.safeComplete(convCtx.Sink)

	// 落库
	s.refreshDebugTraceRuntimeStats(convCtx)
	err := s.conversationArchiveStore.CompleteExchange(ctx,
		convCtx.ConversationId,
		convCtx.ExchangeId,
		answer,
		convCtx.ThinkingSteps.Snapshot(),
		uniqueReferences,
		recommendations,
		convCtx.UsedTools.Snapshot(),
		convCtx.DebugTrace,
		archive.TurnStatusCompleted,
		"",
		func() *int64 {
			if t := convCtx.GetFirstResponseTimeMs(); t > 0 {
				return &t
			}
			return nil
		}(),
		time.Since(convCtx.StartTime).Milliseconds(),
	)
	if err != nil {
		logx.Errorf("成功会话收尾落库失败, conversationId=%s, exchangeId=%d, error=%v",
			convCtx.ConversationId, convCtx.ExchangeId, err)
		if convCtx.TraceRecorder != nil && finalizeStage != nil {
			convCtx.TraceRecorder.FailStage(finalizeStage, "完成态收尾失败。", err.Error(), nil)
		}
	} else {
		if convCtx.TraceRecorder != nil && finalizeStage != nil {
			convCtx.TraceRecorder.CompleteStage(finalizeStage, "会话已按完成状态收尾。", map[string]interface{}{
				"finalStatus":         archive.TurnStatusCompleted.String(),
				"referenceCount":      len(uniqueReferences),
				"recommendationCount": len(recommendations),
				"answerLength":        len(answer),
			})
		}
	}
}

// finishWithFailure 失败处理
func (c *LogicImpl) finishWithFailure(ctx context.Context, conversationCtx *vo.ConversationContext, err error) {
	if !conversationCtx.Finalized.CompareAndSwap(false, true) {
		return
	}
	errorMessage := c.buildErrorMessage(err)
	var finalizeStage interface{}
	if conversationCtx.TraceRecorder != nil {
		modeName := ""
		if conversationCtx.ExecutionPlan != nil {
			modeName = conversationCtx.ExecutionPlan.Mode.String()
		}
		finalizeStage = conversationCtx.TraceRecorder.StartStage(trace.StageCodeFinalize, modeName, "正在收尾失败会话。", nil)
	}
	logx.Errorf("会话执行失败, conversationId=%s, exchangeId=%d, error=%s",
		conversationCtx.ConversationId, conversationCtx.ExchangeId, errorMessage)

	defer func() {
		if r := recover(); r != nil {
			logx.Warnf("发送失败事件或关闭流失败, conversationId=%s, exchangeId=%d, error=%v",
				conversationCtx.ConversationId, conversationCtx.ExchangeId, r)
		}
	}()
	c.safeEmit(conversationCtx.Sink, c.streamEventWriter.Error(errorMessage, conversationCtx.EventMetadata))
	defer func() {
		c.safeComplete(conversationCtx.Sink)
	}()

	c.refreshDebugTraceRuntimeStats(conversationCtx)
	err2 := c.conversationArchiveStore.CompleteExchange(ctx,
		conversationCtx.ConversationId,
		conversationCtx.ExchangeId,
		conversationCtx.GetAnswer(),
		conversationCtx.ThinkingSteps.Snapshot(),
		c.deduplicateReferences(conversationCtx.References.Snapshot()),
		[]string{},
		conversationCtx.UsedTools.Snapshot(),
		conversationCtx.DebugTrace,
		archive.TurnStatusFailed,
		errorMessage,
		func() *int64 {
			if t := conversationCtx.GetFirstResponseTimeMs(); t > 0 {
				return &t
			}
			return nil
		}(),
		time.Since(conversationCtx.StartTime).Milliseconds(),
	)
	if err2 != nil {
		logx.Errorf("失败会话收尾落库失败, conversationId=%s, exchangeId=%d, error=%v",
			conversationCtx.ConversationId, conversationCtx.ExchangeId, err2)
		if conversationCtx.TraceRecorder != nil && finalizeStage != nil {
			conversationCtx.TraceRecorder.FailStage(finalizeStage, "失败态收尾失败。", err2.Error(), nil)
		}
	} else {
		if conversationCtx.TraceRecorder != nil && finalizeStage != nil {
			conversationCtx.TraceRecorder.CompleteStage(finalizeStage, "会话已按失败状态收尾。", map[string]interface{}{
				"finalStatus":  archive.TurnStatusFailed.String(),
				"errorMessage": errorMessage,
				"answerLength": len(conversationCtx.GetAnswer()),
			})
		}
	}
	defer c.safeRefreshConversationSummary(conversationCtx.ConversationId)
	defer c.cleanup(conversationCtx)
}

// buildErrorMessage 构建错误消息
func (c *LogicImpl) buildErrorMessage(err error) string {
	// 简化实现，可根据需要包装
	return err.Error()
}

// refreshDebugTraceRuntimeStats 刷新调试追踪统计
func (c *LogicImpl) refreshDebugTraceRuntimeStats(conversationCtx *vo.ConversationContext) {
	if conversationCtx.DebugTrace == nil || conversationCtx.TraceRecorder == nil {
		return
	}
	modelUsageTraces := conversationCtx.TraceRecorder.SnapshotModelUsageTraces()
	conversationCtx.DebugTrace.ModelUsageTraces = modelUsageTraces
	limitStats := &debug.ChatLimitStats{
		ModelCallsUsed:        len(modelUsageTraces),
		ModelCallsRunLimit:    c.chatAgentProperties.MaxModelCallsPerRun,
		ModelCallsThreadLimit: c.chatAgentProperties.MaxModelCallsPerThread,
		ToolCallsUsed:         len(conversationCtx.UsedTools.Snapshot()),
		ToolCallsRunLimit:     c.chatAgentProperties.MaxToolCallsPerRun,
		ToolCallsThreadLimit:  c.chatAgentProperties.MaxToolCallsPerThread,
	}
	conversationCtx.DebugTrace.LimitStats = limitStats
}

//
// // cleanup 清理资源
// func (s *BusinessChatService) cleanup(conversationCtx *vo.ConversationContext) {
// 	if cancel := conversationCtx.GetLeaseRenewalCancel(); cancel != nil {
// 		cancel()
// 	}
// 	if cancel := conversationCtx.GetExecutionCancel(); cancel != nil {
// 		cancel()
// 	}
// 	s.releaseLeaseQuietly(conversationCtx.LeaseKey, conversationCtx.LeaseOwnerToken)
// 	s.chatRuntimeRegistry.Remove(conversationCtx.ConversationId, conversationCtx)
// }

// rejectStream 拒绝流式请求
func (c *LogicImpl) rejectStream(message, conversationId string, exchangeId int64) <-chan string {
	stream := make(chan string, 1)
	defer close(stream)
	stream <- c.streamEventBuilder.ErrorWithMetadata(message, conversationId, exchangeId)
	return stream
}

func (c *LogicImpl) buildConversationCtx(plan *vo.StreamLaunchPlan, exchange *entity.ChatExchange) *vo.ConversationContext {
	// todo runnableConfig := s.buildSessionConfig(plan.ConversationId)
	// thinkingSteps := &syncSlice{data: make([]string, 0)}
	// references := &syncSliceRef{data: make([]chat.SearchReference, 0)}
	// usedTools := &syncSet{data: make(map[string]struct{})}

	traceId := utils.GenerateUUIDWithoutHyphen()
	trace := vo.NewConversationTrace(plan.ConversationId, exchange.ID, traceId)
	// eventMetadata := &StreamEventMetadata{
	// 	ConversationId: plan.ConversationId,
	// 	ExchangeId:     exchange.ID,
	// }

	// todo 设置上下文
	// runnableConfig.Context["eventSink"] = channel
	// runnableConfig.Context["eventMetadata"] = eventMetadata
	// runnableConfig.Context["thinkingSteps"] = thinkingSteps
	// runnableConfig.Context["references"] = references
	// runnableConfig.Context["usedTools"] = usedTools
	// runnableConfig.Context["traceId"] = traceId
	// runnableConfig.Context["question"] = plan.Question
	// runnableConfig.Context["chatMode"] = plan.ChatMode.String()
	// runnableConfig.Context["currentDate"] = plan.CurrentDate.Format(time.RFC3339)
	// runnableConfig.Context["currentDateText"] = plan.CurrentDateText
	//
	// putContextIfNotNull(runnableConfig.Context, "selectedDocumentId", plan.SelectedDocumentId)
	// putContextIfNotBlank(runnableConfig.Context, "selectedDocumentName", plan.SelectedDocumentName)
	// putContextIfNotNull(runnableConfig.Context, "selectedTaskId", plan.SelectedTaskId)

	// debugTrace := vo.NewChatDebugTrace(nil)
	// runnableConfig.Context["debugTrace"] = debugTrace

	return &vo.ConversationContext{
		ConversationId:       plan.ConversationId,
		ExchangeId:           exchange.ID,
		Question:             plan.Question,
		ChatMode:             plan.ChatMode,
		TraceId:              traceId,
		SelectedDocumentId:   plan.SelectedDocumentId,
		SelectedDocumentName: plan.SelectedDocumentName,
		SelectedTaskId:       plan.SelectedTaskId,
		CurrentDate:          plan.CurrentDate,
		CurrentDateText:      plan.CurrentDateText,
		// ExecutionPlan:        nil,
		// DebugTrace:           debugTrace,
		// RunnableConfig:       runnableConfig,
		Trace:   trace,
		Channel: make(chan string, channelBufferSize),
		// EventMetadata:   eventMetadata,
		LeaseKey: chatRunningLeasePrefix + plan.ConversationId,
		// ThinkingSteps: thinkingSteps,
		// References:    references,
		// UsedTools:     usedTools,
		StartTime: time.Now(),
	}
}
