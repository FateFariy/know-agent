package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/duke-git/lancet/v2/slice"
	"github.com/duke-git/lancet/v2/strutil"
	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/logx"

	"github.com/swiftbit/know-agent/api/chat"
	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/config"
	"github.com/swiftbit/know-agent/internal/domain/chat/adapter"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/prompt"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/trace"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
	"github.com/swiftbit/know-agent/internal/domain/chat/support"
	kllg "github.com/swiftbit/know-agent/internal/domain/knowledge/logic"
	ragmodel "github.com/swiftbit/know-agent/internal/domain/rag/model/vo"
	"github.com/swiftbit/know-agent/internal/svc"
)

const (
	chatRunningLeasePrefix        = "chat:running:"
	chatRunningLeaseRenewInterval = 10 * time.Second
	channelBufferSize             = 100
	defaultHistoryPreviewTurns    = 3
)

// LogicImpl 聊天业务逻辑实现
type LogicImpl struct {
	config             *config.Config
	repo               adapter.ChatRepository
	orchestratorLogic  logic.ChatPreparationOrchestratorLogic
	promptTempLogic    logic.PromptTemplateLogic
	tracer             *trace.ConversationTraceRecorder
	streamEventBuilder *support.StreamEventBuilder
	runtimeRegistry    *support.ChatRuntimeRegistry
	knowledgeLogic     kllg.KnowledgeLogic
	recommendLogic     logic.RecommendationLogic
	memoryLogic        logic.SessionMemoryLogic
	distributedLock    adapter.DistributedLock
}

// NewChatLogic 创建聊天逻辑实例
func NewChatLogic(svcCtx *svc.ServiceContext,
	repo adapter.ChatRepository,
	knowledgeLogic kllg.KnowledgeLogic,
	orchestratorLogic logic.ChatPreparationOrchestratorLogic,
	promptTempLogic logic.PromptTemplateLogic,
	recommendLogic logic.RecommendationLogic,
	memoryLogic logic.SessionMemoryLogic,
	distributedLock adapter.DistributedLock,
) *LogicImpl {
	return &LogicImpl{
		config:             svcCtx.Config,
		repo:               repo,
		orchestratorLogic:  orchestratorLogic,
		promptTempLogic:    promptTempLogic,
		tracer:             trace.NewConversationTraceRecorder(repo),
		streamEventBuilder: &support.StreamEventBuilder{},
		runtimeRegistry:    &support.ChatRuntimeRegistry{},
		knowledgeLogic:     knowledgeLogic,
		recommendLogic:     recommendLogic,
		memoryLogic:        memoryLogic,
		distributedLock:    distributedLock,
	}
}

// OpenConversationStream 打开会话流
// 整体流程：
//
//  1. 构建启动计划（规范化会话ID、问题、模式、文档）
//  2. 获取分布式运行租约（防止会话同时执行多次）
//  3. 启动会话（落库 exchange、注册到运行注册表）
//  4. 异步执行：
//     a. 发送 thinking 事件
//     b. 构建执行计划（改写/路由/检索/会话记忆）
//     c. 后续由执行器消费（此处仅完成计划构建的落库与上下文填充）
//  5. 成功/失败/停止 的收尾（落库 + 发送引用/推荐事件）
func (c *LogicImpl) OpenConversationStream(ctx context.Context, cmd *vo.ChatCommand) <-chan string {
	cmdJSON, _ := json.Marshal(cmd)
	logx.Infof("====== request 内容：%s", string(cmdJSON))

	// 1) 构建启动计划
	plan, err := c.buildLaunchPlan(ctx, cmd)
	if err != nil {
		logx.Errorf("会话启动计划构建失败, conversationId=%s, question=%s, err=%s",
			cmd.ConversationId, cmd.Question, err.Error())
		return c.rejectStream(err.Error(), plan.ConversationId, 0)
	}

	// 2) 获取分布式租约
	leaseKey := chatRunningLeasePrefix + plan.ConversationId
	if err = c.distributedLock.TryLock(ctx, leaseKey); err != nil {
		logx.Warnf("该会话当前正在执行中，请稍后再试, conversationId=%s", plan.ConversationId)
		return c.rejectStream("该会话当前正在执行中，请稍后再试", plan.ConversationId, 0)
	}

	// 3) 启动会话：创建 exchange + 注册运行上下文
	convCtx, err := c.bootstrapConversation(ctx, plan)
	if err != nil {
		_ = c.distributedLock.Unlock(ctx, leaseKey)
		return c.rejectStream(err.Error(), plan.ConversationId, 0)
	}

	// 4) 异步执行（流式返回）
	go func() {
		defer func() {
			// 释放租约
			if leaseErr := c.distributedLock.Unlock(context.Background(), leaseKey); leaseErr != nil {
				logx.Alert(fmt.Sprintf("锁 %s 释放失败: %s", leaseKey, leaseErr.Error()))
			}
			// 从注册表移除
			c.runtimeRegistry.Remove(plan.ConversationId, convCtx)
			// 关闭通道
			close(convCtx.Channel)
		}()

		// 激活生成：租约续期 + 执行计划构建
		c.activateGeneration(convCtx)
	}()

	return convCtx.Channel
}

// ListKnowledgeDocumentOptions 获取知识文档选项列表
func (c *LogicImpl) ListKnowledgeDocumentOptions(ctx context.Context) ([]*chat.KnowledgeDocumentOptionResp, error) {
	docs, err := c.knowledgeLogic.ListRetrievableDocuments(ctx)
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
func (c *LogicImpl) StopConversation(ctx context.Context, conversationId string) (*chat.ConversationStopResp, error) {
	convCtx, ok := c.runtimeRegistry.Get(conversationId)
	if !ok {
		return &chat.ConversationStopResp{Success: false}, fmt.Errorf("没有找到正在执行的会话")
	}
	_ = c.stopTask(convCtx, "用户已停止生成")
	return &chat.ConversationStopResp{Success: true}, nil
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
		// 备注：api/chat 中 ConversationSessionResp 字段较少，如需更多信息可扩展
	}, nil
}

// GetExchangeDetail 获取对话详情（含阶段追踪）
func (c *LogicImpl) GetExchangeDetail(ctx context.Context, conversationId, exchangeId string) (*chat.ConversationExchangeDetailResp, error) {
	exchangeIdInt, err := strconv.ParseInt(exchangeId, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("exchangeId 非法: %s", exchangeId)
	}
	exchanges, err := c.repo.ListExchanges(ctx, conversationId)
	if err != nil {
		return nil, err
	}
	var matched *entity.ChatExchange
	for _, ex := range exchanges {
		if ex != nil && ex.ID == exchangeIdInt {
			matched = ex
			break
		}
	}
	if matched == nil {
		return nil, fmt.Errorf("轮次不存在: %s", exchangeId)
	}

	stages, err := c.repo.SelectStages(ctx, conversationId, exchangeIdInt)
	if err != nil {
		logx.Warnf("获取阶段追踪失败, conversationId=%s, exchangeId=%s, err=%v", conversationId, exchangeId, err)
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
			StageState:    stageStateText(s.StageState),
			StartTime:     s.CreateTime.Format(time.DateTime),
			EndTime:       s.UpdateTime.Format(time.DateTime),
			ErrorMessage:  s.ErrorMessage,
		})
	}

	thinking := jsonStrings(matched.ThinkingSteps)
	refs := []*chat.SearchReferenceResp{}
	if len(matched.ReferenceList) > 0 {
		_ = json.Unmarshal([]byte(matched.ReferenceList), &refs)
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
func (c *LogicImpl) ResetConversation(ctx context.Context, conversationId string) (*chat.ConversationResetResp, error) {
	// 停止正在运行的会话
	if convCtx, ok := c.runtimeRegistry.Get(conversationId); ok {
		_ = c.stopTask(convCtx, "会话被重置")
	}

	// 删除会话及关联 exchange
	dialogueCount, _, err := c.repo.DeleteSession(ctx, conversationId)
	_ = dialogueCount
	if err != nil {
		return &chat.ConversationResetResp{Success: false}, err
	}

	// 删除记忆摘要
	_ = c.memoryLogic.DeleteConversationSummary(ctx, conversationId)

	return &chat.ConversationResetResp{Success: true}, nil
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
// 对应 Java 的 buildLaunchPlan，负责规范化：
//   - conversationId（空则生成 uuid）
//   - question（trim 后不能为空）
//   - chatMode（根据 ChatCommand.modeStr 或 int 解析）
//   - selectedDocument（仅在文档模式下必填且需命中可用文档）
func (c *LogicImpl) buildLaunchPlan(ctx context.Context, cmd *vo.ChatCommand) (*vo.StreamLaunchPlan, error) {
	// 规范化问题
	question := strutil.Trim(cmd.Question)
	if question == "" {
		return nil, fmt.Errorf("question 不能为空")
	}

	// 规范化会话ID
	conversationId := strutil.Trim(cmd.ConversationId)
	if conversationId == "" {
		conversationId = strings.ReplaceAll(uuid.New().String(), "-", "")
	}

	plan := &vo.StreamLaunchPlan{
		Question:       question,
		ConversationId: conversationId,
		ChatMode:       cmd.ChatMode,
	}
	plan.FillCurrentDate()

	// 处理文档选择：仅在文档模式下要求 selectedDocumentId
	if cmd.ChatMode == nil {
		// 默认走自动模式
		plan.ChatMode = vo.ChatQueryModeAutoDocument
	}

	if cmd.SelectedDocumentId != 0 {
		documents, err := c.knowledgeLogic.ListRetrievableDocuments(ctx)
		if err != nil {
			return nil, err
		}
		selectedDocument, ok := slice.FindBy(documents, func(index int, doc *ragmodel.KnowledgeDocument) bool {
			return doc != nil && doc.DocumentId == cmd.SelectedDocumentId
		})
		if !ok || selectedDocument == nil {
			return nil, fmt.Errorf("所选文档当前不可检索: %d", cmd.SelectedDocumentId)
		}
		plan.SelectedDocumentId = selectedDocument.DocumentId
		plan.SelectedDocumentName = selectedDocument.DocumentName
		plan.SelectedTaskId = selectedDocument.LastIndexTaskId
	}
	return plan, nil
}

// bootstrapConversation 启动会话
// 创建对话记录并注册到运行注册表。若注册失败，说明会话正被其他执行接管，则拒绝。
func (c *LogicImpl) bootstrapConversation(ctx context.Context, plan *vo.StreamLaunchPlan) (*vo.ConversationContext, error) {
	dialogue := &entity.ChatDialogue{
		ConversationId:       plan.ConversationId,
		ChatMode:             chatModeValue(plan.ChatMode),
		SelectedDocumentId:   plan.SelectedDocumentId,
		SelectedDocumentName: plan.SelectedDocumentName,
		Question:             plan.Question,
	}
	exchange, err := c.repo.StartExchange(ctx, dialogue)
	if err != nil {
		logx.Errorf("启动 exchange 失败, conversationId=%s, err=%v", plan.ConversationId, err)
		return nil, err
	}

	convCtx := c.buildConversationCtx(plan, exchange)

	if !c.runtimeRegistry.Register(convCtx) {
		// 已存在正在执行的会话，回写失败状态并拒绝
		failExchange := &entity.ChatExchange{
			ID:             exchange.ID,
			ConversationId: plan.ConversationId,
			TurnStatus:     vo.ChatTurnStatusFailed,
			ErrorMessage:   "该会话当前正在执行中，请稍后再试",
		}
		_ = c.repo.CompleteExchange(ctx, failExchange)
		return nil, fmt.Errorf("该会话当前正在执行中，请稍后再试")
	}
	return convCtx, nil
}

// activateGeneration 激活生成逻辑
// 1) 启动租约续期守护
// 2) 发送 thinking 事件
// 3) 构建执行计划（改写 + 路由 + 记忆 + 文档）并写回上下文
// 4) 生成推荐追问（如启用）
// 执行过程中若触发停止或失败，则走失败/停止分支。
func (c *LogicImpl) activateGeneration(convCtx *vo.ConversationContext) {
	defer func() {
		if r := recover(); r != nil {
			logx.Errorf("activateGeneration panic, conversationId=%s, recover=%v", convCtx.ConversationId, r)
			c.finishWithFailure(convCtx, fmt.Errorf("执行出现异常: %v", r))
		}
	}()
	if convCtx.Finalized.Load() {
		return
	}

	// 启动租约续期守护
	leaseDone := make(chan struct{})
	go c.startLeaseRenewal(convCtx, leaseDone)
	defer close(leaseDone)

	// 发送 "正在分析问题上下文" 事件
	safeEmit(convCtx.Channel, c.streamEventBuilder.ThinkingWithMetadata(
		"正在分析问题上下文。", convCtx.ConversationId, convCtx.ExchangeId))

	// 构建执行计划
	execPlan, err := c.prepareExecutionPlan(convCtx)
	if err != nil {
		logx.Errorf("构建执行计划失败, conversationId=%s, exchangeId=%d, err=%v",
			convCtx.ConversationId, convCtx.ExchangeId, err)
		c.finishWithFailure(convCtx, err)
		return
	}
	convCtx.ExecutionPlan.Store(execPlan)

	// 发送执行计划信息（用于前端调试/感知）
	safeEmit(convCtx.Channel, c.streamEventBuilder.ThinkingWithMetadata(
		"上下文分析完成，已准备执行计划。", convCtx.ConversationId, convCtx.ExchangeId))

	// 若外部尚未终止，则按成功路径收尾（真正的模型回答由上层 Agent 继续注入 AnswerBuffer）
	// 这里保留 Java BusinessChatService 的语义：完成 prepare 后将执行控制权交给上层，
	// 由上层流式写入 AnswerBuffer 并调用 CompleteExchange 落库。
	// 为保持与旧实现的行为一致，这里不主动结束会话。
}

// startLeaseRenewal 租约续期守护，对应 Java 的 startLeaseRenewal
// 若续期失败则自动停止当前会话并终止生成。
func (c *LogicImpl) startLeaseRenewal(convCtx *vo.ConversationContext, done <-chan struct{}) {
	ticker := time.NewTicker(chatRunningLeaseRenewInterval)
	defer ticker.Stop()
	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			if convCtx.Finalized.Load() {
				return
			}
			if err := c.distributedLock.Extend(context.Background(), convCtx.LeaseKey); err != nil {
				logx.Alert(fmt.Sprintf("会话租约续期失败, conversationId=%s, exchangeId=%d, err=%v",
					convCtx.ConversationId, convCtx.ExchangeId, err))
				_ = c.stopTask(convCtx, "会话租约已失效，已停止生成")
				return
			}
		}
	}
}

// prepareExecutionPlan 准备执行计划
// 1) 调用编排器准备基础计划（改写、路由、历史记忆等）
// 2) 使用 prompt 模板构造 agent 问题（包含当前日期/上下文提示/历史摘要）
// 3) 根据所选文档刷新会话范围（在文档模式下）
// 4) 初始化调试轨迹
func (c *LogicImpl) prepareExecutionPlan(convCtx *vo.ConversationContext) (*vo.ConversationExecutionPlan, error) {
	execPlan, err := c.orchestratorLogic.Prepare(context.Background(), convCtx)
	if err != nil {
		return nil, err
	}

	variables := map[string]any{
		"currentDateText":              execPlan.CurrentDateText,
		"requiresCurrentDateAnchoring": false,
		"requiresRealTimeSearch":       false,
		"hasHistorySummary":            strutil.IsNotBlank(execPlan.HistorySummary),
		"historySummary":               execPlan.HistorySummary,
		"question":                     execPlan.OriginalQuestion,
	}
	agentQuestion, renderErr := c.promptTempLogic.Render(prompt.AgentQuestion, variables)
	if renderErr != nil {
		logx.Errorf("渲染 agent 问题失败, conversationId=%s, exchangeId=%d, err=%v",
			convCtx.ConversationId, convCtx.ExchangeId, renderErr)
		// 渲染失败不致命，退化为原始问题
		agentQuestion = execPlan.OriginalQuestion
	}
	execPlan.AgentQuestion = agentQuestion

	// 文档模式下若 selectedDocumentId 发生变化，则刷新会话范围
	if execPlan.SelectedDocumentId > 0 && execPlan.SelectedDocumentId != convCtx.SelectedDocumentId {
		dialogue := &entity.ChatDialogue{
			ConversationId:       convCtx.ConversationId,
			ChatMode:             chatModeValue(execPlan.ChatMode),
			SelectedDocumentId:   execPlan.SelectedDocumentId,
			SelectedDocumentName: execPlan.SelectedDocumentName,
		}
		if refreshErr := c.repo.RefreshSessionScope(context.Background(), dialogue); refreshErr != nil {
			logx.Warnf("刷新会话范围失败, conversationId=%s, err=%v", convCtx.ConversationId, refreshErr)
		}
	}

	debugTrace := vo.NewChatDebugTrace(execPlan)
	convCtx.DebugTrace.Store(debugTrace)
	return execPlan, nil
}

// buildConversationExecution 构建对话执行（执行计划构建的外层封装）
func (c *LogicImpl) buildConversationExecution(convCtx *vo.ConversationContext) func(ctx context.Context) {
	return func(ctx context.Context) {
		if convCtx.Finalized.Load() {
			return
		}
		safeEmit(convCtx.Channel, c.streamEventBuilder.ThinkingWithMetadata(
			"正在分析问题上下文。", convCtx.ConversationId, convCtx.ExchangeId))

		plan, err := c.prepareExecutionPlan(convCtx)
		if err != nil {
			c.finishWithFailure(convCtx, err)
			return
		}
		convCtx.ExecutionPlan.Store(plan)
	}
}

// emitModelChunk 发出模型输出块（text 事件），并更新首响应时间
func (c *LogicImpl) emitModelChunk(convCtx *vo.ConversationContext, chunk string) {
	if strutil.IsBlank(chunk) {
		return
	}
	convCtx.AnswerBuffer.WriteString(chunk)
	convCtx.FirstResponseTimeMs.CompareAndSwap(0, time.Since(convCtx.StartTime).Milliseconds())
	safeEmit(convCtx.Channel, c.streamEventBuilder.TextWithMetadata(
		chunk, convCtx.ConversationId, convCtx.ExchangeId))
}

// stopTask 停止任务：原子切换状态 -> 发送停止事件 -> 落库 -> 清理
func (c *LogicImpl) stopTask(convCtx *vo.ConversationContext, reason string) *vo.ConversationStop {
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

	// 发送 status 事件
	safeEmit(convCtx.Channel, c.streamEventBuilder.StatusWithMetadata(
		"⏹ "+reason, convCtx.ConversationId, convCtx.ExchangeId))

	// 刷新调试轨迹统计
	c.refreshDebugTraceRuntimeStats(convCtx)

	// 构造停止态 exchange 并落库
	firstResponse := int64(0)
	if v := convCtx.FirstResponseTimeMs.Load(); v > 0 {
		firstResponse = v
	}
	stopExchange := &entity.ChatExchange{
		ID:                  convCtx.ExchangeId,
		ConversationId:      convCtx.ConversationId,
		Question:            convCtx.Question,
		Answer:              convCtx.AnswerBuffer.String(),
		ThinkingSteps:       toJSONArray(snapshotStrings(convCtx.ThinkingSteps)),
		ReferenceList:       toJSONArray(snapshotReferences(convCtx.References)),
		RecommendationList:  toJSONArray([]string{}),
		UsedToolList:        toJSONArray(snapshotUsedTools(convCtx.UsedTools)),
		DebugTraceJson:      debugTraceJSON(convCtx),
		TurnStatus:          vo.ChatTurnStatusStopped,
		ErrorMessage:        reason,
		FirstResponseTimeMs: firstResponse,
		TotalResponseTimeMs: time.Since(convCtx.StartTime).Milliseconds(),
	}
	if err := c.repo.CompleteExchange(context.Background(), stopExchange); err != nil {
		logx.Errorf("停止会话落库失败, conversationId=%s, exchangeId=%d, err=%v",
			convCtx.ConversationId, convCtx.ExchangeId, err)
	}

	// 异步刷新会话摘要
	go func() {
		defer func() { _ = recover() }()
		c.memoryLogic.RefreshConversationSummaryAsync(context.Background(), convCtx.ConversationId)
	}()

	return &vo.ConversationStop{
		ConversationId: convCtx.ConversationId,
		Stopped:        true,
		Message:        "已停止会话生成",
	}
}

// finishSuccessfully 成功完成：发送引用/推荐事件 -> 落库 -> 清理
// 对应 Java 的 finishSuccessfully
func (c *LogicImpl) finishSuccessfully(convCtx *vo.ConversationContext) {
	if !convCtx.Finalized.CompareAndSwap(false, true) {
		return
	}
	answer := convCtx.AnswerBuffer.String()
	uniqueReferences := snapshotReferences(convCtx.References)

	// 生成推荐追问
	execPlan := convCtx.ExecutionPlan.Load()
	var recommendations []string
	if execPlan != nil &&
		execPlan.Mode != nil &&
		execPlan.Mode == vo.ExecutionModeClarification &&
		len(execPlan.ClarificationOptions) > 0 {
		recommendations = execPlan.ClarificationOptions
	} else {
		recentExchanges := c.fetchRecentExchanges(convCtx.ConversationId, convCtx.ExchangeId)
		recommendations = c.recommendLogic.GenerateRecommendations(
			context.Background(), convCtx.Question, answer, recentExchanges, convCtx.Trace)
	}

	// 补发引用/推荐事件
	if len(uniqueReferences) > 0 {
		refs := make([]*vo.SearchReference, 0, len(uniqueReferences))
		for _, r := range uniqueReferences {
			if r != nil {
				refs = append(refs, r)
			}
		}
		safeEmit(convCtx.Channel, c.streamEventBuilder.ReferencesWithMetadata(
			refs, convCtx.ConversationId, convCtx.ExchangeId))
	}
	if len(recommendations) > 0 {
		safeEmit(convCtx.Channel, c.streamEventBuilder.RecommendationsWithMetadata(
			recommendations, convCtx.ConversationId, convCtx.ExchangeId))
	}

	// 刷新调试轨迹统计
	c.refreshDebugTraceRuntimeStats(convCtx)

	firstResponse := int64(0)
	if v := convCtx.FirstResponseTimeMs.Load(); v > 0 {
		firstResponse = v
	}
	successExchange := &entity.ChatExchange{
		ID:                  convCtx.ExchangeId,
		ConversationId:      convCtx.ConversationId,
		Question:            convCtx.Question,
		Answer:              answer,
		ThinkingSteps:       toJSONArray(snapshotStrings(convCtx.ThinkingSteps)),
		ReferenceList:       toJSONArray(uniqueReferences),
		RecommendationList:  toJSONArray(recommendations),
		UsedToolList:        toJSONArray(snapshotUsedTools(convCtx.UsedTools)),
		DebugTraceJson:      debugTraceJSON(convCtx),
		TurnStatus:          vo.ChatTurnStatusCompleted,
		ErrorMessage:        "",
		FirstResponseTimeMs: firstResponse,
		TotalResponseTimeMs: time.Since(convCtx.StartTime).Milliseconds(),
	}
	if err := c.repo.CompleteExchange(context.Background(), successExchange); err != nil {
		logx.Errorf("成功会话收尾落库失败, conversationId=%s, exchangeId=%d, err=%v",
			convCtx.ConversationId, convCtx.ExchangeId, err)
	}

	// 异步刷新会话摘要
	go func() {
		defer func() { _ = recover() }()
		c.memoryLogic.RefreshConversationSummaryAsync(context.Background(), convCtx.ConversationId)
	}()
}

// finishWithFailure 失败收尾
func (c *LogicImpl) finishWithFailure(convCtx *vo.ConversationContext, err error) {
	if !convCtx.Finalized.CompareAndSwap(false, true) {
		return
	}
	errorMessage := err.Error()
	logx.Errorf("会话执行失败, conversationId=%s, exchangeId=%d, error=%s",
		convCtx.ConversationId, convCtx.ExchangeId, errorMessage)

	safeEmit(convCtx.Channel, c.streamEventBuilder.ErrorWithMetadata(
		errorMessage, convCtx.ConversationId, convCtx.ExchangeId))

	c.refreshDebugTraceRuntimeStats(convCtx)

	firstResponse := int64(0)
	if v := convCtx.FirstResponseTimeMs.Load(); v > 0 {
		firstResponse = v
	}
	failExchange := &entity.ChatExchange{
		ID:                  convCtx.ExchangeId,
		ConversationId:      convCtx.ConversationId,
		Question:            convCtx.Question,
		Answer:              convCtx.AnswerBuffer.String(),
		ThinkingSteps:       toJSONArray(snapshotStrings(convCtx.ThinkingSteps)),
		ReferenceList:       toJSONArray(snapshotReferences(convCtx.References)),
		RecommendationList:  toJSONArray([]string{}),
		UsedToolList:        toJSONArray(snapshotUsedTools(convCtx.UsedTools)),
		DebugTraceJson:      debugTraceJSON(convCtx),
		TurnStatus:          vo.ChatTurnStatusFailed,
		ErrorMessage:        errorMessage,
		FirstResponseTimeMs: firstResponse,
		TotalResponseTimeMs: time.Since(convCtx.StartTime).Milliseconds(),
	}
	if dbErr := c.repo.CompleteExchange(context.Background(), failExchange); dbErr != nil {
		logx.Errorf("失败会话落库失败, conversationId=%s, exchangeId=%d, err=%v",
			convCtx.ConversationId, convCtx.ExchangeId, dbErr)
	}

	go func() {
		defer func() { _ = recover() }()
		c.memoryLogic.RefreshConversationSummaryAsync(context.Background(), convCtx.ConversationId)
	}()
}

// refreshDebugTraceRuntimeStats 刷新调试轨迹中的统计信息
func (c *LogicImpl) refreshDebugTraceRuntimeStats(convCtx *vo.ConversationContext) {
	if convCtx == nil {
		return
	}
	debugTrace := convCtx.DebugTrace.Load()
	if debugTrace == nil {
		return
	}
	limitStats := &vo.ChatLimitStats{
		ModelCallsUsed:        len(snapshotUsedTools(convCtx.UsedTools)),
		ToolCallsUsed:         len(snapshotUsedTools(convCtx.UsedTools)),
		ModelCallsRunLimit:    32,
		ToolCallsRunLimit:     32,
		ModelCallsThreadLimit: 64,
		ToolCallsThreadLimit:  64,
	}
	debugTrace.LimitStats = limitStats
	convCtx.DebugTrace.Store(debugTrace)
}

// ---------------------------------------------------------------------------
// 辅助方法（构建上下文、流辅助、JSON 辅助等）
// ---------------------------------------------------------------------------

// buildConversationCtx 构建对话运行上下文（对应 Java 中 TaskInfo / ConversationContext 的创建）
func (c *LogicImpl) buildConversationCtx(plan *vo.StreamLaunchPlan, exchange *entity.ChatExchange) *vo.ConversationContext {
	traceId := utils.GenerateUUIDWithoutHyphen()
	convTrace := vo.NewConversationTrace(plan.ConversationId, exchange.ID, traceId)

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
		Trace:                convTrace,
		Channel:              make(chan string, channelBufferSize),
		LeaseKey:             chatRunningLeasePrefix + plan.ConversationId,
		ThinkingSteps:        []string{},
		References:           []*vo.SearchReference{},
		UsedTools:            map[string]struct{}{},
		StartTime:            time.Now(),
	}
}

// rejectStream 生成一个仅含错误事件的只读流
func (c *LogicImpl) rejectStream(message, conversationId string, exchangeId int64) <-chan string {
	stream := make(chan string, 1)
	defer close(stream)
	stream <- c.streamEventBuilder.ErrorWithMetadata(message, conversationId, exchangeId)
	return stream
}

// fetchRecentExchanges 获取最近的历史轮次（排除当前）
func (c *LogicImpl) fetchRecentExchanges(conversationId string, excludeExchangeId int64) []*entity.ChatExchange {
	turns := defaultHistoryPreviewTurns
	if c.config != nil && c.config.Chat.Recommendation.HistoryPreviewTurns > 0 {
		turns = c.config.Chat.Recommendation.HistoryPreviewTurns
	}
	recent, err := c.repo.ListRecentExchanges(context.Background(), conversationId, turns)
	if err != nil {
		logx.Warnf("列出最近轮次失败, conversationId=%s, err=%v", conversationId, err)
		return []*entity.ChatExchange{}
	}
	result := make([]*entity.ChatExchange, 0, len(recent))
	for _, ex := range recent {
		if ex == nil || ex.ID == excludeExchangeId {
			continue
		}
		result = append(result, ex)
	}
	return result
}

// ---------------------------------------------------------------------------
// 纯函数/工具方法
// ---------------------------------------------------------------------------

// safeEmit 安全地向通道写入事件，若写入阻塞则忽略，保证生成链路不被卡住
func safeEmit(ch chan<- string, payload string) {
	defer func() { _ = recover() }()
	if ch == nil {
		return
	}
	ch <- payload
}

// chatModeValue 返回 chatMode 的整型值（给 entity 使用）
func chatModeValue(mode vo.ChatQueryMode) int {
	if mode == nil {
		return 0
	}
	return mode.Value()
}

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

// stageStateText 返回阶段状态文本
func stageStateText(state int) string {
	switch state {
	case vo.ChatTurnStatusRunning:
		return "进行中"
	case vo.ChatTurnStatusCompleted:
		return "已完成"
	case vo.ChatTurnStatusFailed:
		return "失败"
	case vo.ChatTurnStatusStopped:
		return "已停止"
	default:
		return "未知"
	}
}

// snapshotStrings 获取思考步骤的快照（拷贝一份）
func snapshotStrings(items []string) []string {
	if len(items) == 0 {
		return []string{}
	}
	out := make([]string, len(items))
	copy(out, items)
	return out
}

// snapshotReferences 生成引用快照并按 uniqueKey 去重
func snapshotReferences(refs []*vo.SearchReference) []*vo.SearchReference {
	if len(refs) == 0 {
		return []*vo.SearchReference{}
	}
	seen := sync.Map{}
	out := make([]*vo.SearchReference, 0, len(refs))
	for _, r := range refs {
		if r == nil {
			continue
		}
		key := fmt.Sprintf("%d-%d-%s", r.DocumentId, r.ChunkId, r.Snippet)
		if _, dup := seen.LoadOrStore(key, struct{}{}); dup {
			continue
		}
		out = append(out, r)
	}
	return out
}

// snapshotUsedTools 获取已用工具的快照
func snapshotUsedTools(used map[string]struct{}) []string {
	if len(used) == 0 {
		return []string{}
	}
	out := make([]string, 0, len(used))
	for k := range used {
		out = append(out, k)
	}
	return out
}

// debugTraceJSON 序列化调试轨迹
func debugTraceJSON(convCtx *vo.ConversationContext) string {
	if convCtx == nil {
		return ""
	}
	dt := convCtx.DebugTrace.Load()
	if dt == nil {
		return ""
	}
	data, err := json.Marshal(dt)
	if err != nil {
		return ""
	}
	return string(data)
}

// toJSONArray 将任意切片序列化为 common.JSONArray 所需的 JSON 数组文本
// entity.ChatExchange 的 ThinkingSteps/ReferenceList/... 为 common.JSONArray（即 []byte）
func toJSONArray[T any](items []T) []byte {
	if items == nil {
		return []byte("[]")
	}
	data, err := json.Marshal(items)
	if err != nil {
		return []byte("[]")
	}
	return data
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
