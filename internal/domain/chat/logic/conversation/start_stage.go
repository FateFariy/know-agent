package conversation

import (
	"context"
	"errors"
	"time"

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
	stop            func(context.Context, *Context, string) (string, bool)
}

var _ Stage = (*StartStage)(nil)

func NewStartStage(
	repo adapter.ChatRepository,
	runtimeRegistry *ChatRuntimeRegistry,
	distributedLock adapter.DistributedLock,
	stop func(context.Context, *Context, string) (string, bool),
) *StartStage {
	return &StartStage{
		repo:            repo,
		runtimeRegistry: runtimeRegistry,
		distributedLock: distributedLock,
		stop:            stop,
	}
}

// Name 阶段名称
func (s *StartStage) Name() string {
	return "启动会话"
}

// Execute 执行逻辑
func (s *StartStage) Execute(ctx context.Context, convCtx *Context) (err error) {
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

	return s.activateGeneration(cancelCtx, convCtx)
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

	debugTrace := vo.NewChatDebugTrace(convCtx.ExecutionPlan.Load())
	convCtx.DebugTrace.Store(debugTrace)

	// 发送"正在分析问题上下文"的思考事件，便于客户端感知流程
	if err := convCtx.PublishThinking("正在分析问题上下文。"); err != nil {
		return err
	}

	// 执行完成后再次检查 finalize（下游在执行期间被取消时）
	if convCtx.Finalized.Load() {
		convCtx.ReleaseResources()
	}

	return nil
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
				s.stop(ctx, convCtx, "会话租约已失效，已停止生成")
				return
			}
		}
	}
}

// releaseConversationLock 释放会话分布式锁
func (s *StartStage) releaseConversationLock(leaseKey string) {
	err := s.distributedLock.Unlock(leaseKey)
	if err != nil && !errors.Is(err, errorx.ErrDistributedLockNotFound) {
		logx.Warnf("会话分布式锁释放失败, leaseKey=%s, err=%v", leaseKey, err)
	}
}
