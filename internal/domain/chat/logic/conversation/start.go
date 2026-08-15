package conversation

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/duke-git/lancet/v2/strutil"

	"github.com/swiftbit/know-agent/common/logx"
	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/chat/adapter"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/enum"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
	errorx "github.com/swiftbit/know-agent/internal/error"
)

const (
	chatRunningLeasePrefix        = "conversation:running:"
	chatRunningLeaseRenewInterval = 10 * time.Second
)

type StartStage struct {
	repo            adapter.ChatRepository
	runtimeRegistry *ChatRuntimeRegistry
	distributedLock adapter.DistributedLock
}

var _ Stage = (*StartStage)(nil)

func NewStart(repo adapter.ChatRepository, runtimeRegistry *ChatRuntimeRegistry, distributedLock adapter.DistributedLock) *StartStage {
	return &StartStage{
		repo:            repo,
		runtimeRegistry: runtimeRegistry,
		distributedLock: distributedLock,
	}
}

// Name 阶段名称
func (s *StartStage) Name() string {
	return "启动会话"
}

// Execute 执行逻辑
func (s *StartStage) Execute(ctx context.Context, convCtx *Context) (err error) {
	panic("unimplemented")
}

// bootstrapConversation 启动会话：创建本轮 exchange 记录，构建对话上下文，注册到运行注册表，
// 最后在独立 goroutine 中激活生成逻辑，异步返回客户端可读的流式 channel。
// 并发控制：注册失败表示会话已在执行中，直接落库为失败状态并拒绝，避免同一会话重复执行。
func (s *StartStage) bootstrapConversation(ctx context.Context, convCtx *Context) error {
	// 启动本轮交互（写入 ChatDialogue + ChatExchange，状态置为 Running）
	exchange, err := s.startExchange(ctx, convCtx)
	if err != nil {
		return err
	}

	// 完善对话上下文，绑定可取消的 context（用于后续终止生成）
	convCtx.Finalize(exchange)
	cancelCtx, cancelFunc := context.WithCancel(ctx)
	convCtx.CancelFunc = cancelFunc

	// 将 ConversationTrace 存入上下文，供下游 callbacks.Handler 使用
	cancelCtx = vo.WithTrace(cancelCtx, convCtx.Trace)

	// 注册对话上下文到运行注册表；注册失败意味着会话正被其他执行接管
	if !s.runtimeRegistry.Register(convCtx) {
		return errorx.ErrSessionRunning
	}

	s.activateGeneration(cancelCtx, convCtx)

	return nil
}

func (s *StartStage) startExchange(ctx context.Context, convCtx *Context) (*entity.ChatExchange, error) {
	// 构造对话实体（按 ConversationId 聚合整个会话），状态初始化为 Running
	dialogue := &entity.ChatDialogue{
		ConversationId:                 convCtx.ConversationId,
		SessionStatus:                  enum.ChatSessionStatusRunning,
		ChatMode:                       convCtx.ChatMode,
		SelectedDocumentId:             convCtx.SelectedDocumentId,
		SelectedDocumentName:           convCtx.SelectedDocumentName,
		KnowledgeBaseSelectionMode:     convCtx.KnowledgeBaseSelectionSnapshot.SelectionModeName(),
		SelectedKnowledgeBaseIdsJson:   convCtx.KnowledgeBaseSelectionSnapshot.SelectionIDs(),
		SelectedKnowledgeBaseNamesJson: convCtx.KnowledgeBaseSelectionSnapshot.SelectionNames(),
	}
	// 构造本轮交互实体，状态 Running
	chatExchange := &entity.ChatExchange{
		ID:                             utils.GetSnowflakeNextID(),
		ConversationId:                 convCtx.ConversationId,
		Question:                       convCtx.Question,
		TurnStatus:                     enum.ChatTurnStatusRunning,
		KnowledgeBaseSelectionMode:     convCtx.KnowledgeBaseSelectionSnapshot.SelectionModeName(),
		SelectedKnowledgeBaseIdsJson:   convCtx.KnowledgeBaseSelectionSnapshot.SelectionIDs(),
		SelectedKnowledgeBaseNamesJson: convCtx.KnowledgeBaseSelectionSnapshot.SelectionNames(),
		RetrievalConfigSnapshot:        convCtx.KnowledgeBaseSelectionSnapshot.RagRuntimeConfigSnapshot(),
	}
	// 事务中原子执行：Upsert 对话 + 插入新交互
	startFn := func(txCtx context.Context) error {
		// Upsert：若对话记录已存在（同一 ConversationId）则更新，否则插入
		if err := s.repo.UpsertDialogue(txCtx, dialogue); err != nil {
			return err
		}
		// 插入本轮交互记录
		return s.repo.InsertExchange(txCtx, chatExchange)
	}
	if err := s.repo.Do(ctx, startFn); err != nil {
		return nil, err
	}
	return chatExchange, nil

}

// activateGeneration 激活生成逻辑: 执行对话的生成、流式下发与收尾工作。
//
// 执行流程：
//  1. 检查 finalized 快速返回（会话已被取消/终止）
//  2. 启动租约续期 goroutine，用于周期性延长分布式锁
//  3. 执行 buildConversationExecution 构建并执行对话生成；失败时走失败收尾
//  4. 进入 for-select 循环消费执行结果：
//     - context 被取消 → 调用 stop 中止
//     - resultCh 关闭 → 调用 finishSuccessfully 收尾成功
//     - 收到 chunk → 转发给客户端 channel；发送失败则按失败收尾
//
// 并发设计：多处 Finalized 检查确保下游在开始前即被取消时及时释放资源。
func (s *StartStage) activateGeneration(ctx context.Context, convCtx *Context) error {
	// 快速路径：会话已被前置 finalize，直接返回
	if convCtx.Finalized.Load() {
		return nil
	}

	// 启动租约续期 goroutine，用于周期性延长分布式锁
	go s.startLeaseRenewal(ctx, convCtx)

	// 再次检查 finalize（避免刚启动即被取消，此时直接释放资源并返回）
	if convCtx.Finalized.Load() {
		convCtx.ReleaseResources()
		return nil
	}

	// 发送"正在分析问题上下文"的思考事件，便于客户端感知流程
	if err := convCtx.PublishThinking("正在分析问题上下文。"); err != nil {
		return err
	}

	// 执行完成后再次检查 finalize（下游在执行期间被取消时）
	if convCtx.Finalized.Load() {
		convCtx.ReleaseResources()
		return nil
	}
}

// startLeaseRenewal 启动租约续期，若续期失败则自动停止当前会话并终止生成
func (s *StartStage) startLeaseRenewal(ctx context.Context, convCtx *Context) {
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
			if err := s.distributedLock.Extend(ctx, convCtx.LeaseKey); err != nil {
				logx.Warnf("会话租约续期失败，准备停止当前会话, conversationId=%s, exchangeId=%d, err=%v",
					convCtx.ConversationId, convCtx.ExchangeId, err)
				s.stopTask(ctx, convCtx, "会话租约已失效，已停止生成")
				return
			}
		}
	}
}

func (s *StartStage) prepareExecutionPlan(ctx context.Context, convCtx *Context) (*vo.ConversationExecutionPlan, error) {
	execPlan, err := s.preOrchestrator.Prepare(ctx, convCtx)
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
	agentQuestion, err := s.renderer.Render(enum.AgentQuestion, variables)
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
		if err = s.repo.RefreshSessionScope(ctx, dialogue); err != nil {
			logx.Warnf("刷新会话范围失败, conversationId=%s, err=%v", convCtx.ConversationId, err)
			return nil, err
		}
	}

	debugTrace := vo.NewChatDebugTrace(execPlan)
	convCtx.DebugTrace.Store(debugTrace)
	convCtx.ExecutionPlan.Store(execPlan)

	return execPlan, nil
}

// releaseConversationLock 释放会话分布式锁
func (s *StartStage) releaseConversationLock(leaseKey string) {
	ctx, cancelFunc := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelFunc()
	err := s.distributedLock.UnlockContext(ctx, leaseKey)
	if err != nil && !errors.Is(err, errorx.ErrDistributedLockNotFound) {
		logx.Warnf("会话分布式锁释放失败, leaseKey=%s, err=%v", leaseKey, err)
	}
}
