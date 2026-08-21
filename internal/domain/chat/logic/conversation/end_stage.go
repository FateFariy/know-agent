package conversation

import (
	"context"

	"github.com/swiftbit/know-agent/common"
	"github.com/swiftbit/know-agent/common/logx"
	"github.com/swiftbit/know-agent/internal/domain/chat/adapter"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/enum"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
)

type End struct {
	repo adapter.ChatRepository
}

var _ Stage = (*End)(nil)

func NewEnd(repo adapter.ChatRepository) *End {
	return &End{
		repo: repo,
	}
}

// Name 阶段名称
func (e *End) Name() string {
	return enum.ConversationTraceStageFinalize.Name
}

// Execute 执行逻辑
func (e *End) Execute(ctx context.Context, convCtx *Context) error {
	// 原子检查 Finalized 标志（CAS），确保仅首次调用生效，避免重复收尾
	if !convCtx.Finalized.CompareAndSwap(false, true) {
		return nil
	}

	// 启动 finalize 与 recommendation 两个追踪阶段
	finalizeCtx := vo.OnStart(ctx, enum.ConversationTraceStageFinalize,
		convCtx.ExecutionModeName(), &vo.StageInput{SummaryText: "正在收尾已完成会话。"})

	answer := convCtx.Answer()
	uniqueReferences := convCtx.UniqueReferences()
	recommendations := convCtx.Recommendations

	// 组装成功态 ChatExchange，调用 completeExchange 落库；并根据落库结果完成或标记 finalize 追踪阶段
	successExchange := convCtx.BuildChatExchange(enum.ChatTurnStatusCompleted, "")
	successExchange.Recommendations = common.ToJSONArray(recommendations)
	if err := e.completeExchange(ctx, successExchange); err == nil {
		// 落库成功：完成 finalize 追踪阶段，写入完成快照（含推荐、引用、答案长度）
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

	return nil
}

// completeExchange 完成会话交互（exchange）
func (e *End) completeExchange(ctx context.Context, exchange *entity.ChatExchange) error {
	completeFn := func(txCtx context.Context) error {
		// 更新交互记录（含答案、耗时、最终状态等，由调用方已在 exchange 对象中填充）
		if err := e.repo.UpdateExchangeById(txCtx, exchange); err != nil {
			return err
		}
		// 将对应会话的状态重置为 Idle（释放"运行中"标记）
		dialogue := &entity.ChatDialogue{
			ConversationId: exchange.ConversationId,
			SessionStatus:  enum.ChatSessionStatusIdle,
		}
		return e.repo.UpdateDialogueByConversationId(txCtx, dialogue)
	}
	if err := e.repo.Do(ctx, completeFn); err != nil {
		logx.Errorf("会话落库失败, conversationId=%s, exchangeId=%d, err=%v",
			exchange.ConversationId, exchange.ID, err)
		return err
	}
	return nil
}
