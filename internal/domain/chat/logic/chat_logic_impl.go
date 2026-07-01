package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/duke-git/lancet/v2/slice"
	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/logx"

	"github.com/swiftbit/know-agent/api/chat"
	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/chat/adapter"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
	"github.com/swiftbit/know-agent/internal/domain/chat/support"
	"github.com/swiftbit/know-agent/internal/domain/knowledge/logic"
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

// ChatLogicImpl 聊天业务逻辑实现
type ChatLogicImpl struct {
	svcCtx             *svc.ServiceContext
	repo               adapter.ChatRepository
	streamEventBuilder *support.StreamEventBuilder
	runtimeRegistry    *support.ChatRuntimeRegistry
	knowledgeLogic     logic.DocumentKnowledgeLogic
	distributedLock    adapter.DistributedLock
}

// NewChatLogic 创建聊天逻辑实例
func NewChatLogic(repo adapter.ChatRepository, knowledgeLogic logic.DocumentKnowledgeLogic, distributedLock adapter.DistributedLock) *ChatLogicImpl {
	return &ChatLogicImpl{
		repo:               repo,
		streamEventBuilder: &support.StreamEventBuilder{},
		runtimeRegistry:    &support.ChatRuntimeRegistry{},
		knowledgeLogic:     knowledgeLogic,
		distributedLock:    distributedLock,
	}
}

// OpenConversationStream 打开会话流
func (c *ChatLogicImpl) OpenConversationStream(ctx context.Context, cmd *vo.ChatCommand) <-chan string {
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
func (c *ChatLogicImpl) ListKnowledgeDocumentOptions(ctx context.Context) ([]*chat.KnowledgeDocumentOptionResp, error) {
	return []*chat.KnowledgeDocumentOptionResp{
		{DocumentId: 1, DocumentName: "产品手册.pdf"},
		{DocumentId: 2, DocumentName: "技术文档.docx"},
		{DocumentId: 3, DocumentName: "用户指南.txt"},
	}, nil
}

// StopConversation 停止会话
func (c *ChatLogicImpl) StopConversation(ctx context.Context, conversationId string) (*chat.ConversationStopResp, error) {
	convCtx, ok := c.runtimeRegistry.Get(conversationId)
	if !ok {
		return &chat.ConversationStopResp{Success: false}, fmt.Errorf("没有找到正在执行的会话")
	}

	return &chat.ConversationStopResp{Success: true}, nil
}

// GetSession 获取会话详情
func (c *ChatLogicImpl) GetSession(ctx context.Context, conversationId string) (*chat.ConversationSessionResp, error) {
	return &chat.ConversationSessionResp{
		ConversationId: conversationId,
		Title:          "测试会话",
		LatestMessage:  "你好，有什么可以帮助你的？",
		CreateTime:     time.Now().Format(time.RFC3339),
		UpdateTime:     time.Now().Format(time.RFC3339),
	}, nil
}

// GetExchangeDetail 获取对话详情
func (c *ChatLogicImpl) GetExchangeDetail(ctx context.Context, conversationId, exchangeId string) (*chat.ConversationExchangeDetailResp, error) {
	return &chat.ConversationExchangeDetailResp{
		ExchangeId:     exchangeId,
		ConversationId: conversationId,
		UserMessage:    "你好",
		AgentMessage:   "你好，有什么可以帮助你的？",
		CreateTime:     time.Now().Format(time.RFC3339),
	}, nil
}

// ListSessions 获取会话列表
func (c *ChatLogicImpl) ListSessions(ctx context.Context, req *chat.ConversationSessionListReq) (*chat.ConversationSessionListResp, error) {
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
				CreateTime:     time.Now().Format(time.RFC3339),
				UpdateTime:     time.Now().Format(time.RFC3339),
			},
			{
				ConversationId: uuid.New().String(),
				Title:          "会话2",
				LatestMessage:  "消息内容2",
				CreateTime:     time.Now().Format(time.RFC3339),
				UpdateTime:     time.Now().Format(time.RFC3339),
			},
		},
	}, nil
}

// ResetConversation 重置会话
func (c *ChatLogicImpl) ResetConversation(ctx context.Context, conversationId string) (*chat.ConversationResetResp, error) {
	return &chat.ConversationResetResp{Success: true}, nil
}

// RebuildConversationSummary 重建会话摘要
func (c *ChatLogicImpl) RebuildConversationSummary(ctx context.Context, conversationId string) (*chat.ConversationMemorySummaryResp, error) {
	return &chat.ConversationMemorySummaryResp{
		ConversationId: conversationId,
		Summary:        "这是会话的摘要内容",
	}, nil
}

// GetRetrievalResults 获取检索结果
func (c *ChatLogicImpl) GetRetrievalResults(ctx context.Context, conversationId string, exchangeId int64) ([]*chat.RetrievalResultResp, error) {
	return []*chat.RetrievalResultResp{}, nil
}

// GetChannelExecutions 获取渠道执行结果
func (c *ChatLogicImpl) GetChannelExecutions(ctx context.Context, conversationId string, exchangeId int64) ([]*chat.ChannelExecutionResp, error) {
	return []*chat.ChannelExecutionResp{}, nil
}

// GetStageBenchmarks 获取阶段基准
func (c *ChatLogicImpl) GetStageBenchmarks(ctx context.Context) ([]*chat.StageBenchmarkResp, error) {
	return []*chat.StageBenchmarkResp{}, nil
}

func (c *ChatLogicImpl) buildLaunchPlan(ctx context.Context, cmd *vo.ChatCommand) (*vo.StreamLaunchPlan, error) {
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
func (c *ChatLogicImpl) bootstrapConversation(ctx context.Context, launchPlan *vo.StreamLaunchPlan) (<-chan string, error) {
	dialogue := &entity.ChatDialogue{
		ConversationId:       launchPlan.ConversationId,
		ChatMode:             launchPlan.ChatMode,
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

func (c *ChatLogicImpl) executeConversation(convCtx *vo.ConversationContext, stream chan<- string) {
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

// PrepareExecutionPlan 准备对话执行计划
func (c *ChatLogicImpl) PrepareExecutionPlan(convCtx *vo.ConversationContext) (*vo.ConversationExecutionPlan, error) {
	// 1. 调用编排器准备基础计划
	executionPlan, err := chatPreparationOrchestrator.Prepare(context.Background(), convCtx)
	if err != nil {
		logx.Errorf("准备执行计划失败, conversationId=%s, exchangeId=%d, error=%v",
			convCtx.ConversationId, convCtx.ExchangeId, err)
		return nil, err
	}

	// 2. 设置 Agent 问题
	executionPlan.AgentQuestion = buildAgentQuestion(executionPlan)

	// 3. 如果选中的文档 ID 不为空，并且与 convCtx 中的不同，则更新会话范围并存入上下文
	if executionPlan.SelectedDocumentId != nil &&
		(convCtx.SelectedDocumentId == nil || *executionPlan.SelectedDocumentId != *convCtx.SelectedDocumentId) {

		// 刷新会话范围（存储到数据库/缓存）
		if conversationArchiveStore != nil {
			err := conversationArchiveStore.RefreshSessionScope(
				context.Background(),
				convCtx.ConversationId,
				executionPlan.ChatMode,
				executionPlan.SelectedDocumentId,
				executionPlan.SelectedDocumentName,
			)
			if err != nil {
				logx.Warnf("刷新会话范围失败, conversationId=%s, error=%v", convCtx.ConversationId, err)
				// 原 Java 未抛出异常，仅记录？这里也仅记录继续执行
			}
		}

		// 将选中的文档/任务信息放入 runnableConfig 的上下文中
		if convCtx.RunnableConfig.Context == nil {
			convCtx.RunnableConfig.Context = make(map[string]interface{})
		}
		PutContextIfNotNull(convCtx.RunnableConfig.Context, "selectedDocumentId", executionPlan.SelectedDocumentId)
		PutContextIfNotBlank(convCtx.RunnableConfig.Context, "selectedDocumentName", executionPlan.SelectedDocumentName)
		PutContextIfNotNull(convCtx.RunnableConfig.Context, "selectedTaskId", executionPlan.SelectedTaskId)
	}

	// 4. 将执行计划存入 vo.ConversationContext
	convCtx.SetExecutionPlan(executionPlan)

	// 5. 初始化调试追踪并存入 vo.ConversationContext 和配置上下文
	debugTrace := initializeDebugTrace(executionPlan)
	convCtx.SetDebugTrace(debugTrace)

	if convCtx.RunnableConfig.Context == nil {
		convCtx.RunnableConfig.Context = make(map[string]interface{})
	}
	convCtx.RunnableConfig.Context["debugTrace"] = debugTrace

	return executionPlan, nil
}

// activateGeneration 激活生成逻辑
func (c *ChatLogicImpl) activateGeneration(ctx context.Context, convCtx *vo.ConversationContext) {
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

	cancelCtx, cancelFunc := context.WithCancel(ctx)
	convCtx.CancelExecute = cancelFunc

	go c.startLeaseRenewal(cancelCtx, convCtx)

	if convCtx.Finalized.Load() {
		cancelFunc()
		return
	}

	c.buildConversationExecution(convCtx)(cancelCtx)
	if convCtx.Finalized.Load() {
		cancelFunc()
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
func (c *ChatLogicImpl) startLeaseRenewal(ctx context.Context, convCtx *vo.ConversationContext) {
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
func (c *ChatLogicImpl) buildConversationExecution(convCtx *vo.ConversationContext) func(ctx context.Context) {
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

		// 获取执行器
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
func (c *ChatLogicImpl) emitModelChunk(convCtx *vo.ConversationContext, chunk string) {
	convCtx.AnswerBuffer.WriteString(chunk)
	convCtx.FirstResponseTimeMs.CompareAndSwap(0, time.Since(convCtx.StartTime).Milliseconds())

	support.SafeEmitNext(convCtx.Channel, c.streamEventBuilder.TextWithMetadata(chunk, convCtx.ConversationId, convCtx.ExchangeId))
}

// prepareExecutionPlan 准备执行计划
func (c *ChatLogicImpl) prepareExecutionPlan(ctx context.Context, conversationCtx *vo.ConversationContext) (*plan.ConversationExecutionPlan, error) {
	executionPlan, err := c.chatPreparationOrchestrator.Prepare(ctx, conversationCtx)
	if err != nil {
		return nil, err
	}
	executionPlan.AgentQuestion = c.buildAgentQuestion(executionPlan)

	if executionPlan.SelectedDocumentId != nil &&
		(conversationCtx.SelectedDocumentId == nil || *executionPlan.SelectedDocumentId != *conversationCtx.SelectedDocumentId) {
		// 刷新会话范围
		c.conversationArchiveStore.RefreshSessionScope(ctx,
			conversationCtx.ConversationId,
			executionPlan.ChatMode,
			executionPlan.SelectedDocumentId,
			executionPlan.SelectedDocumentName,
		)
		// 更新上下文
		putContextIfNotNull(conversationCtx.RunnableConfig.Context, "selectedDocumentId", executionPlan.SelectedDocumentId)
		putContextIfNotBlank(conversationCtx.RunnableConfig.Context, "selectedDocumentName", executionPlan.SelectedDocumentName)
		putContextIfNotNull(conversationCtx.RunnableConfig.Context, "selectedTaskId", executionPlan.SelectedTaskId)
	}

	conversationCtx.SetExecutionPlan(executionPlan)
	debugTrace := c.initializeDebugTrace(executionPlan)
	conversationCtx.SetDebugTrace(debugTrace)
	conversationCtx.RunnableConfig.Context["debugTrace"] = debugTrace
	return executionPlan, nil
}

// stopTask 停止任务
func (c *ChatLogicImpl) stopTask(ctx context.Context, convCtx *vo.ConversationContext, reason string) *vo.ConversationStop {
	// 原子地将 finalized 从 false 设置为 true，若已经是 true 则直接返回“会话已经结束”
	if !convCtx.Finalized.CompareAndSwap(false, true) {
		return &vo.ConversationStop{
			ConversationId: convCtx.ConversationId,
			Stopped:        false,
			Message:        "会话已经结束",
		}
	}

	// 检查当前运行时注册表中的任务是否仍是当前 convCtx，防止被新任务接管
	if currentTask, exists := c.runtimeRegistry.Get(convCtx.ConversationId); exists && currentTask != convCtx {
		return &vo.ConversationStop{
			ConversationId: convCtx.ConversationId,
			Stopped:        false,
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

	// 释放执行任务（disposable）
	if execCancel := convCtx.GetExecutionCancel(); execCancel != nil {
		execCancel()
		convCtx.SetExecutionCancel(nil) // 避免重复调用
	}

	// 准备停止响应的消息
	responseMessage := "已停止会话生成"

	// 开始追踪收尾阶段（如果 traceRecorder 存在）
	var finalizeStage interface{} // 原 Java 中的 StageHandle 对象，这里用 interface{} 占位
	if convCtx.TraceRecorder != nil {
		// TODO: 调用 traceRecorder.StartStage，需要定义相应接口
		// finalizeStage = convCtx.TraceRecorder.StartStage(...)
		// 参数：ConversationTraceStage.FINALIZE, modeName, description, metadata
	}

	// 辅助函数：安全发送状态事件
	safeEmitStatus := func() {
		defer func() {
			if r := recover(); r != nil {
				logx.Warnf("发送停止事件 panic: %v", r)
			}
		}()
		// 构造状态事件消息：⏹ + reason
		statusMsg := "⏹ " + reason
		// 调用 safeEmit，将 statusMsg 作为事件写入 sink
		err := support.SafeEmitNext(convCtx.Sink, statusMsg)
		if err != nil {
			return
		}
	}

	// 辅助函数：安全完成 sink
	safeCompleteSink := func() {
		defer func() {
			if r := recover(); r != nil {
				logx.Warnf("关闭 SSE 流 panic: %v", r)
			}
		}()
		safeComplete(convCtx.Sink)
	}

	// 发送停止事件（如果 panic 或异常则捕获并记录）
	func() {
		defer func() {
			if r := recover(); r != nil {
				logx.Warnf("发送停止事件失败, conversationId=%s, exchangeId=%d, error=%v",
					convCtx.ConversationId, convCtx.ExchangeId, r)
				responseMessage = "会话已停止，停止事件发送失败"
			}
		}()
		safeEmitStatus()
	}()

	// 最终收尾：关闭 sink、落库、清理
	func() {
		defer func() {
			// 确保 finally 中的清理总会执行
			safeRefreshConversationSummary(convCtx.ConversationId)
			cleanup(convCtx)
		}()

		// 关闭 sink（原 Java 中的 safeComplete）
		func() {
			defer func() {
				if r := recover(); r != nil {
					logx.Warnf("关闭停止中的 SSE 流失败, conversationId=%s, exchangeId=%d, error=%v",
						convCtx.ConversationId, convCtx.ExchangeId, r)
				}
			}()
			safeCompleteSink()
		}()

		// 刷新追踪运行时统计
		refreshDebugTraceRuntimeStats(convCtx)

		// 落库完整 exchange 信息
		err := conversationArchiveStore.CompleteExchange(
			context.Background(),
			convCtx.ConversationId,
			convCtx.ExchangeId,
			convCtx.GetAnswer(), // answerBuffer 内容
			snapshotStringList(convCtx.ThinkingSteps),
			deduplicateReferences(snapshotReferenceList(convCtx.References)),
			[]interface{}{}, // 空列表对应 Java 的 List.of()
			snapshotUsedTools(convCtx.UsedTools),
			convCtx.GetDebugTrace(),
			ChatTurnStatusStopped,
			reason,
			toNullable(convCtx.GetFirstResponseTimeMs()),
			time.Since(convCtx.StartTime).Milliseconds(),
		)
		if err != nil {
			logx.Errorf("停止会话落库失败, conversationId=%s, exchangeId=%d, error=%v",
				convCtx.ConversationId, convCtx.ExchangeId, err)
			responseMessage = "会话已停止，收尾落库失败"
			if convCtx.TraceRecorder != nil && finalizeStage != nil {
				// TODO: 调用 traceRecorder.FailStage
				// convCtx.TraceRecorder.FailStage(finalizeStage, "停止态收尾失败。", err.Error(), nil)
			}
		} else {
			if convCtx.TraceRecorder != nil && finalizeStage != nil {
				// TODO: 调用 traceRecorder.CompleteStage
				// convCtx.TraceRecorder.CompleteStage(finalizeStage, "会话已按停止状态收尾。", map[string]interface{}{
				//     "finalStatus": ChatTurnStatusStopped,
				//     "reason":      reason,
				//     "answerLength": len(convCtx.AnswerBuffer),
				// })
			}
		}
	}()

	return &vo.ConversationStop{
		ConversationId: convCtx.ConversationId,
		Stopped:        true,
		Message:        responseMessage,
	}
}

// finishSuccessfully 成功完成处理
func (c *ChatLogicImpl) finishSuccessfully(ctx context.Context, convCtx *vo.ConversationContext) {
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
func (c *ChatLogicImpl) finishWithFailure(ctx context.Context, conversationCtx *vo.ConversationContext, err error) {
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
func (c *ChatLogicImpl) buildErrorMessage(err error) string {
	// 简化实现，可根据需要包装
	return err.Error()
}

// refreshDebugTraceRuntimeStats 刷新调试追踪统计
func (c *ChatLogicImpl) refreshDebugTraceRuntimeStats(conversationCtx *vo.ConversationContext) {
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
func (c *ChatLogicImpl) rejectStream(message, conversationId string, exchangeId int64) <-chan string {
	stream := make(chan string, 1)
	defer close(stream)
	stream <- c.streamEventBuilder.ErrorWithMetadata(message, conversationId, exchangeId)
	return stream
}

func (c *ChatLogicImpl) buildConversationCtx(plan *vo.StreamLaunchPlan, exchange *entity.ChatExchange) *vo.ConversationContext {
	// todo runnableConfig := s.buildSessionConfig(plan.ConversationId)
	// thinkingSteps := &syncSlice{data: make([]string, 0)}
	// references := &syncSliceRef{data: make([]chat.SearchReference, 0)}
	// usedTools := &syncSet{data: make(map[string]struct{})}

	traceId := utils.GenerateUUIDWithoutHyphen()
	tracer := vo.NewConversationTrace(plan.ConversationId, exchange.ID, traceId)
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
		Tracer:  tracer,
		Channel: make(chan string, channelBufferSize),
		// EventMetadata:   eventMetadata,
		LeaseKey: chatRunningLeasePrefix + plan.ConversationId,
		// ThinkingSteps: thinkingSteps,
		// References:    references,
		// UsedTools:     usedTools,
		StartTime: time.Now(),
	}
}
