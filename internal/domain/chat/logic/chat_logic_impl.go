package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/duke-git/lancet/v2/slice"
	"github.com/zeromicro/go-zero/core/logx"

	"github.com/swiftbit/know-agent/internal/domain/chat/adapter"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
	"github.com/swiftbit/know-agent/internal/domain/chat/support"
	"github.com/swiftbit/know-agent/internal/domain/knowledge/logic"
	klvo "github.com/swiftbit/know-agent/internal/domain/knowledge/model/vo"
	errorx "github.com/swiftbit/know-agent/internal/error"
	"github.com/swiftbit/know-agent/internal/svc"
)

const (
	chatRunningLeasePrefix        = "chat:running:"
	chatRunningLeaseTTL           = 30 * time.Second
	chatRunningLeaseRenewInterval = 10 * time.Second
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
		streamEventBuilder: support.NewStreamEventBuilder(),
		runtimeRegistry:    support.NewChatRuntimeRegistry(),
		knowledgeLogic:     knowledgeLogic,
		distributedLock:    distributedLock,
	}
}

// OpenConversationStream 打开会话流
func (c *ChatLogicImpl) OpenConversationStream(ctx context.Context, cmd *vo.ChatCommand) <-chan string {
	cmdJSON, _ := json.Marshal(cmd)
	logx.Infof("======request内容：%s", string(cmdJSON))

	stream := make(chan string, 100)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				err := r.(error)
				logx.Errorf("会话启动失败, conversationId=%s, question=%s, err=%s", cmd.ConversationId, cmd.Question, err.Error())
				stream <- c.streamEventBuilder.ErrorWithMetadata(err.Error(), &vo.StreamEventMetadata{ConversationId: cmd.ConversationId})
			}
		}()

		defer close(stream)

		if err := c.distributedLock.TryLock(ctx, chatRunningLeasePrefix+cmd.ConversationId); err != nil {
			c.streamEventBuilder.ErrorWithMetadata("当前会话正在执行中，请稍后再试", &vo.StreamEventMetadata{ConversationId: cmd.ConversationId})
			return
		}
		defer func() {
			err := c.distributedLock.Unlock(ctx, chatRunningLeasePrefix+cmd.ConversationId)
			logx.Alert(fmt.Sprintf("锁释放失败: %s", err))
		}()

		launchPlan, err := c.buildLaunchPlan(ctx, cmd)
		if err != nil {
			panic(err)
		}
		conversationCtx := NewTaskInfo(cmd.ConversationId, cmd.ConversationId, cmd.Question)

		if !c.runtimeRegistry.Register(conversationCtx) {
			errMsg := c.streamEventBuilder.ErrorWithMetadata("该会话当前正在执行中，请稍后再试", conversationCtx.EventMetadata)
			stream <- errMsg
			return
		}

		defer func() {
			c.runtimeRegistry.Remove(conversationId, conversationCtx)
		}()

		c.executeConversation(conversationCtx, stream)
	}()

	return stream
}

func (c *ChatLogicImpl) buildLaunchPlan(ctx context.Context, cmd *vo.ChatCommand) (*vo.StreamLaunchPlan, error) {
	launchPlan := &vo.StreamLaunchPlan{
		Question:             cmd.Question,
		ConversationId:       cmd.ConversationId,
		ChatMode:             cmd.ChatMode,
		SelectedDocumentId:   0,
		SelectedDocumentName: "",
		SelectedTaskId:       0,
	}
	launchPlan.FillCurrentDate()
	if cmd.SelectedDocumentId != 0 {
		documents, err := c.knowledgeLogic.ListRetrievableDocuments(ctx)
		if err != nil {
			return nil, err
		}
		selectedDocument, ok := slice.FindBy(documents, func(index int, doc *klvo.KnowledgeDocumentDescriptor) bool {
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

func (c *ChatLogicImpl) bootstrapConversation(ctx context.Context, launchPlan *vo.StreamLaunchPlan) {
	defer func() {
		if r := recover(); r != nil {
			c.releaseLeaseQuietly(launchPlan.LeaseKey, launchPlan.LeaseOwnerToken)
			if exchangeView != nil {
				errMsg := fmt.Sprintf("panic: %v", r)
				c.failBootstrappedExchange(ctx, launchPlan.ConversationId, exchangeView.ExchangeId, errMsg)
			}
		}
	}()
	dialogue := &entity.ChatDialogue{
		ConversationId:       launchPlan.ConversationId,
		ChatMode:             launchPlan.ChatMode,
		SelectedDocumentId:   launchPlan.SelectedDocumentId,
		SelectedDocumentName: launchPlan.SelectedDocumentName,
		Question:             launchPlan.Question,
	}
	exchange, err := c.repo.StartExchange(ctx, dialogue)
	if err != nil {
		panic(err)
	}

	// 开始交换记录
	exchangeView = c.conversationArchiveStore.StartExchange(ctx,
		launchPlan.ConversationId,
		launchPlan.Question,
		launchPlan.ChatMode,
		launchPlan.SelectedDocumentId,
		launchPlan.SelectedDocumentName,
	)
	if exchangeView == nil {
		errMsg := "failed to start exchange"
		c.failBootstrappedExchange(ctx, launchPlan.ConversationId, 0, errMsg)
		c.releaseLeaseQuietly(launchPlan.LeaseKey, launchPlan.LeaseOwnerToken)
		return
	}

	// 创建任务信息
	conversationCtx := c.createTaskInfo(launchPlan, exchangeView)

	// 注册到运行时注册表
	if !c.chatRuntimeRegistry.Register(conversationCtx) {
		c.failBootstrappedExchange(ctx, launchPlan.ConversationId, exchangeView.ExchangeId, "该会话当前正在执行中，请稍后再试")
		c.releaseLeaseQuietly(launchPlan.LeaseKey, launchPlan.LeaseOwnerToken)
		return &bootstrapResult{RejectionMessage: "该会话当前正在执行中，请稍后再试"}
	}

	// 绑定客户端通道
	outbound := c.bindClientChannel(ctx, conversationCtx)
	return
}

func (c *ChatLogicImpl) executeConversation(conversationCtx *vo.ConversationContext,
	stream chan<- string) {
	thinkingMsg := c.streamEventBuilder.ThinkingWithMetadata("正在分析问题上下文。", conversationCtx.EventMetadata)
	stream <- thinkingMsg

	thinkingMsg = c.streamEventBuilder.ThinkingWithMetadata("正在检索相关知识...", conversationCtx.EventMetadata)
	stream <- thinkingMsg

	time.Sleep(500 * time.Millisecond)

	answer := "这是模拟的回答内容。您的问题是：" + conversationCtx.Question
	for i := 0; i < len(answer); i += 5 {
		end := i + 5
		if end > len(answer) {
			end = len(answer)
		}
		textMsg := c.streamEventBuilder.TextWithMetadata(answer[i:end], conversationCtx.EventMetadata)
		stream <- textMsg
		conversationCtx.AnswerBuffer.WriteString(answer[i:end])
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
	refMsg := c.streamEventBuilder.ReferencesWithMetadata(references, conversationCtx.EventMetadata)
	stream <- refMsg

	recommendations := []string{
		"您想了解更多相关信息吗？",
		"是否需要深入探讨这个话题？",
	}
	recMsg := c.streamEventBuilder.RecommendationsWithMetadata(recommendations, conversationCtx.EventMetadata)
	stream <- recMsg
}

// ListKnowledgeDocumentOptions 获取知识文档选项列表
func (c *ChatLogicImpl) ListKnowledgeDocumentOptions(ctx
context.Context) ([]*chat.KnowledgeDocumentOptionResp, error) {
return []*chat.KnowledgeDocumentOptionResp{
{DocumentId: 1, DocumentName: "产品手册.pdf"},
{DocumentId: 2, DocumentName: "技术文档.docx"},
{DocumentId: 3, DocumentName: "用户指南.txt"},
}, nil
}

// StopConversation 停止会话
func (c *ChatLogicImpl) StopConversation(ctx
context.Context, conversationId
string) (*chat.ConversationStopResp, error) {
conversationCtx, ok := c.runtimeRegistry.Get(conversationId)
if !ok {
return &chat.ConversationStopResp{Success: false}, fmt.Errorf("没有找到正在执行的会话")
}

return &chat.ConversationStopResp{Success: true}, nil
}

// GetSession 获取会话详情
func (c *ChatLogicImpl) GetSession(ctx
context.Context, conversationId
string) (*chat.ConversationSessionResp, error) {
return &chat.ConversationSessionResp{
ConversationId: conversationId,
Title:          "测试会话",
LatestMessage:  "你好，有什么可以帮助你的？",
CreateTime:     time.Now().Format(time.RFC3339),
UpdateTime:     time.Now().Format(time.RFC3339),
}, nil
}

// GetExchangeDetail 获取对话详情
func (c *ChatLogicImpl) GetExchangeDetail(ctx
context.Context, conversationId, exchangeId
string) (*chat.ConversationExchangeDetailResp, error) {
return &chat.ConversationExchangeDetailResp{
ExchangeId:     exchangeId,
ConversationId: conversationId,
UserMessage:    "你好",
AgentMessage:   "你好，有什么可以帮助你的？",
CreateTime:     time.Now().Format(time.RFC3339),
}, nil
}

// ListSessions 获取会话列表
func (c *ChatLogicImpl) ListSessions(ctx
context.Context, req * chat.ConversationSessionListReq) (*chat.ConversationSessionListResp, error) {
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
func (c *ChatLogicImpl) ResetConversation(ctx
context.Context, conversationId
string) (*chat.ConversationResetResp, error) {
return &chat.ConversationResetResp{Success: true}, nil
}

// RebuildConversationSummary 重建会话摘要
func (c *ChatLogicImpl) RebuildConversationSummary(ctx
context.Context, conversationId
string) (*chat.ConversationMemorySummaryResp, error) {
return &chat.ConversationMemorySummaryResp{
ConversationId: conversationId,
Summary:        "这是会话的摘要内容",
}, nil
}

// GetRetrievalResults 获取检索结果
func (c *ChatLogicImpl) GetRetrievalResults(ctx
context.Context, conversationId
string, exchangeId
int64) ([]*chat.RetrievalResultResp, error) {
return []*chat.RetrievalResultResp{}, nil
}

// GetChannelExecutions 获取渠道执行结果
func (c *ChatLogicImpl) GetChannelExecutions(ctx
context.Context, conversationId
string, exchangeId
int64) ([]*chat.ChannelExecutionResp, error) {
return []*chat.ChannelExecutionResp{}, nil
}

// GetStageBenchmarks 获取阶段基准
func (c *ChatLogicImpl) GetStageBenchmarks(ctx
context.Context) ([]*chat.StageBenchmarkResp, error) {
return []*chat.StageBenchmarkResp{}, nil
}

func startLeaseRenewal(conversationCtx *vo.ConversationContext)
context.CancelFunc{
// 创建一个可取消的上下文，用于控制 ticker 的生命周期
ctx, cancel, := context.WithCancel(context.Background())

go func (){
ticker := time.NewTicker(chatRunningLeaseRenewInterval)
defer ticker.Stop()

for{
select{
case <-ctx.Done():
// 外部调用取消函数，停止续期
return
case <-ticker.C:
// 执行续期逻辑，捕获 panic 避免 goroutine 崩溃
func (){
defer func (){
if r := recover(); r != nil{
// 记录 panic 日志，类似 Java 中的 log.warn
zap.L().Warn("租约续期任务出现 panic",
zap.String("conversationId", conversationCtx.ConversationId),
zap.Int64("exchangeId", conversationCtx.ExchangeId),
zap.Any("panic", r),
)
}
}()
renewLeaseOrStop(conversationCtx)
}()
}
}
}()

return cancel
}

// PrepareExecutionPlan 准备对话执行计划
// 对应 Java 方法：private ConversationExecutionPlan prepareExecutionPlan(vo.ConversationContext conversationCtx)
// 返回：执行计划对象
func PrepareExecutionPlan(conversationCtx *vo.ConversationContext) (*ConversationExecutionPlan, error) {
	// 1. 调用编排器准备基础计划
	executionPlan, err := chatPreparationOrchestrator.Prepare(context.Background(), conversationCtx)
	if err != nil {
		logx.Errorf("准备执行计划失败, conversationId=%s, exchangeId=%d, error=%v",
			conversationCtx.ConversationId, conversationCtx.ExchangeId, err)
		return nil, err
	}

	// 2. 设置 Agent 问题
	executionPlan.AgentQuestion = buildAgentQuestion(executionPlan)

	// 3. 如果选中的文档 ID 不为空，并且与 conversationCtx 中的不同，则更新会话范围并存入上下文
	if executionPlan.SelectedDocumentId != nil &&
		(conversationCtx.SelectedDocumentId == nil || *executionPlan.SelectedDocumentId != *conversationCtx.SelectedDocumentId) {

		// 刷新会话范围（存储到数据库/缓存）
		if conversationArchiveStore != nil {
			err := conversationArchiveStore.RefreshSessionScope(
				context.Background(),
				conversationCtx.ConversationId,
				executionPlan.ChatMode,
				executionPlan.SelectedDocumentId,
				executionPlan.SelectedDocumentName,
			)
			if err != nil {
				logx.Warnf("刷新会话范围失败, conversationId=%s, error=%v", conversationCtx.ConversationId, err)
				// 原 Java 未抛出异常，仅记录？这里也仅记录继续执行
			}
		}

		// 将选中的文档/任务信息放入 runnableConfig 的上下文中
		if conversationCtx.RunnableConfig.Context == nil {
			conversationCtx.RunnableConfig.Context = make(map[string]interface{})
		}
		PutContextIfNotNull(conversationCtx.RunnableConfig.Context, "selectedDocumentId", executionPlan.SelectedDocumentId)
		PutContextIfNotBlank(conversationCtx.RunnableConfig.Context, "selectedDocumentName", executionPlan.SelectedDocumentName)
		PutContextIfNotNull(conversationCtx.RunnableConfig.Context, "selectedTaskId", executionPlan.SelectedTaskId)
	}

	// 4. 将执行计划存入 vo.ConversationContext
	conversationCtx.SetExecutionPlan(executionPlan)

	// 5. 初始化调试追踪并存入 vo.ConversationContext 和配置上下文
	debugTrace := initializeDebugTrace(executionPlan)
	conversationCtx.SetDebugTrace(debugTrace)

	if conversationCtx.RunnableConfig.Context == nil {
		conversationCtx.RunnableConfig.Context = make(map[string]interface{})
	}
	conversationCtx.RunnableConfig.Context["debugTrace"] = debugTrace

	return executionPlan, nil
}

// activateGeneration 激活生成逻辑
func (s *BusinessChatService) activateGeneration(ctx
context.Context, conversationCtx * vo.ConversationContext) {
defer func () {
if r := recover(); r != nil {
s.finishWithFailure(ctx, conversationCtx, fmt.Errorf("panic in activateGeneration: %v", r))
}
}()

if conversationCtx.IsFinalized() {
return
}

leaseRenewalCancel := s.startLeaseRenewal(ctx, conversationCtx)
conversationCtx.SetLeaseRenewalCancel(leaseRenewalCancel)
if conversationCtx.IsFinalized() && leaseRenewalCancel != nil {
leaseRenewalCancel()
return
}

execFunc, execCancel := s.buildConversationExecution(conversationCtx)
conversationCtx.SetExecutionCancel(execCancel)
if conversationCtx.IsFinalized() && execCancel != nil {
execCancel()
return
}

// 异步执行
go func () {
defer func () {
if r := recover(); r != nil {
s.finishWithFailure(ctx, conversationCtx, fmt.Errorf("execution panic: %v", r))
}
}()
execFunc(ctx)
}()
}

// stopTask 停止任务
func (c *ChatLogicImpl) stopTask(conversationCtx *vo.ConversationContext,
	reason
string) *vo.ConversationStop{
// 原子地将 finalized 从 false 设置为 true，若已经是 true 则直接返回“会话已经结束”
if !conversationCtx.Finalized.CompareAndSwap(false, true){
return &vo.ConversationStop{
ConversationId: conversationCtx.ConversationId,
Stopped:        false,
Message:        "会话已经结束",
}
}

// 检查当前运行时注册表中的任务是否仍是当前 conversationCtx，防止被新任务接管
if chatRuntimeRegistry != nil{
if currentTask, exists := chatRuntimeRegistry.Get(conversationCtx.ConversationId); exists && currentTask != conversationCtx{
return &vo.ConversationStop{
ConversationId: conversationCtx.ConversationId,
Stopped:        false,
Message:        "会话已由新的执行接管",
}
}
}

// 中断 ReactAgent
if businessChatReactAgent != nil{
// 使用带超时的 context 避免永久阻塞
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()
if err := businessChatReactAgent.Interrupt(ctx, conversationCtx.RunnableConfig); err != nil{
logx.Debugf("中断 ReactAgent 时出现异常，继续释放资源: %v", err)
}
}

// 释放执行任务（disposable）
if execCancel := conversationCtx.GetExecutionCancel(); execCancel != nil{
execCancel()
conversationCtx.SetExecutionCancel(nil) // 避免重复调用
}

// 准备停止响应的消息
responseMessage := "已停止会话生成"

// 开始追踪收尾阶段（如果 traceRecorder 存在）
var finalizeStage interface{} // 原 Java 中的 StageHandle 对象，这里用 interface{} 占位
if conversationCtx.TraceRecorder != nil{
// TODO: 调用 traceRecorder.StartStage，需要定义相应接口
// finalizeStage = conversationCtx.TraceRecorder.StartStage(...)
// 参数：ConversationTraceStageCode.FINALIZE, modeName, description, metadata
}

// 辅助函数：安全发送状态事件
safeEmitStatus := func (){
defer func (){
if r := recover(); r != nil{
logx.Warnf("发送停止事件 panic: %v", r)
}
}()
// 构造状态事件消息：⏹ + reason
statusMsg := "⏹ " + reason
// 调用 safeEmit，将 statusMsg 作为事件写入 sink
err := support.SafeEmitNext(conversationCtx.Sink, statusMsg)
if err != nil{
return
}
}

// 辅助函数：安全完成 sink
safeCompleteSink := func (){
defer func (){
if r := recover(); r != nil{
logx.Warnf("关闭 SSE 流 panic: %v", r)
}
}()
safeComplete(conversationCtx.Sink)
}

// 发送停止事件（如果 panic 或异常则捕获并记录）
func (){
defer func (){
if r := recover(); r != nil{
logx.Warnf("发送停止事件失败, conversationId=%s, exchangeId=%d, error=%v",
conversationCtx.ConversationId, conversationCtx.ExchangeId, r)
responseMessage = "会话已停止，停止事件发送失败"
}
}()
safeEmitStatus()
}()

// 最终收尾：关闭 sink、落库、清理
func (){
defer func (){
// 确保 finally 中的清理总会执行
safeRefreshConversationSummary(conversationCtx.ConversationId)
cleanup(conversationCtx)
}()

// 关闭 sink（原 Java 中的 safeComplete）
func (){
defer func (){
if r := recover(); r != nil{
logx.Warnf("关闭停止中的 SSE 流失败, conversationId=%s, exchangeId=%d, error=%v",
conversationCtx.ConversationId, conversationCtx.ExchangeId, r)
}
}()
safeCompleteSink()
}()

// 刷新追踪运行时统计
refreshDebugTraceRuntimeStats(conversationCtx)

// 落库完整 exchange 信息
err := conversationArchiveStore.CompleteExchange(
context.Background(),
conversationCtx.ConversationId,
conversationCtx.ExchangeId,
conversationCtx.GetAnswer(), // answerBuffer 内容
snapshotStringList(conversationCtx.ThinkingSteps),
deduplicateReferences(snapshotReferenceList(conversationCtx.References)),
[]interface{}{}, // 空列表对应 Java 的 List.of()
snapshotUsedTools(conversationCtx.UsedTools),
conversationCtx.GetDebugTrace(),
ChatTurnStatusStopped,
reason,
toNullable(conversationCtx.GetFirstResponseTimeMs()),
time.Since(conversationCtx.StartTime).Milliseconds(),
)
if err != nil{
logx.Errorf("停止会话落库失败, conversationId=%s, exchangeId=%d, error=%v",
conversationCtx.ConversationId, conversationCtx.ExchangeId, err)
responseMessage = "会话已停止，收尾落库失败"
if conversationCtx.TraceRecorder != nil && finalizeStage != nil{
// TODO: 调用 traceRecorder.FailStage
// conversationCtx.TraceRecorder.FailStage(finalizeStage, "停止态收尾失败。", err.Error(), nil)
}
} else{
if conversationCtx.TraceRecorder != nil && finalizeStage != nil{
// TODO: 调用 traceRecorder.CompleteStage
// conversationCtx.TraceRecorder.CompleteStage(finalizeStage, "会话已按停止状态收尾。", map[string]interface{}{
//     "finalStatus": ChatTurnStatusStopped,
//     "reason":      reason,
//     "answerLength": len(conversationCtx.AnswerBuffer),
// })
}
}
}()

return &vo.ConversationStop{
ConversationId: conversationCtx.ConversationId,
Stopped:        true,
Message:        responseMessage,
}
}
