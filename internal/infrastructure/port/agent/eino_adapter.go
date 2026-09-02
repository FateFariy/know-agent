package agent

import (
	"context"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"

	"github.com/swiftbit/know-agent/internal/domain/chat/logic/conversation"
)

// EinoAdapter 将单个领域 AgentMiddleware 适配为 eino 的 ChatModelAgentMiddleware（*schema.Message）。
//
// 本类型只做框架适配与类型翻译，不含业务逻辑：
//   - 把注入在 ctx 上的 convCtx 取出并传入领域中间件
//   - 把 eino 生命周期状态翻译为领域时机（终态答复文本、指令修正回写等）
type EinoAdapter struct {
	*adk.BaseChatModelAgentMiddleware
	middleware conversation.AgentMiddleware
}

var _ adk.ChatModelAgentMiddleware = (*EinoAdapter)(nil)

// NewEinoAdapter 创建单个领域中间件的 eino 适配器。
func NewEinoAdapter(middleware conversation.AgentMiddleware) *EinoAdapter {
	return &EinoAdapter{
		BaseChatModelAgentMiddleware: &adk.BaseChatModelAgentMiddleware{},
		middleware:                   middleware,
	}
}

// BeforeAgent 在 agent 启动前调用领域中间件，并把修正后的指令回写到 eino 运行上下文
func (a *EinoAdapter) BeforeAgent(ctx context.Context, runCtx *adk.ChatModelAgentContext) (context.Context, *adk.ChatModelAgentContext, error) {
	if a.middleware == nil {
		return ctx, runCtx, nil
	}
	output, err := a.middleware.BeforeAgent(ctx, conversation.AgentContextFrom(ctx), &conversation.BeforeAgentInput{
		Instruction: runCtx.Instruction,
	})
	if err != nil {
		return ctx, runCtx, err
	}
	runCtx.Instruction = output.Instruction

	return ctx, runCtx, nil
}

// AfterModelRewriteState 在每次模型调用后，若本轮产出终态答复文本（最近的无工具调用
// Assistant 消息），则通知领域中间件。文本后续如何处置由领域中间件决策。
func (a *EinoAdapter) AfterModelRewriteState(ctx context.Context, state *adk.ChatModelAgentState, mc *adk.ModelContext) (context.Context, *adk.ChatModelAgentState, error) {
	if a.middleware == nil {
		return ctx, state, nil
	}
	text, ok := lastAssistantReply(state)
	if !ok {
		return ctx, state, nil
	}
	if err := a.middleware.OnModelText(ctx, conversation.AgentContextFrom(ctx), text); err != nil {
		return ctx, state, err
	}
	return ctx, state, nil
}

// AfterAgent 在 agent 正常进入终态后通知领域中间件。
func (a *EinoAdapter) AfterAgent(ctx context.Context, state *adk.ChatModelAgentState) (context.Context, error) {
	if a.middleware == nil {
		return ctx, nil
	}
	if err := a.middleware.AfterAgent(ctx, conversation.AgentContextFrom(ctx)); err != nil {
		return ctx, err
	}
	return ctx, nil
}

// lastAssistantReply 从会话尾部向前取最近一条无工具调用的 Assistant 消息作为终态答复文本。
func lastAssistantReply(state *adk.ChatModelAgentState) (string, bool) {
	if state == nil {
		return "", false
	}
	for i := len(state.Messages) - 1; i >= 0; i-- {
		msg := state.Messages[i]
		if msg == nil || msg.Role != schema.Assistant {
			continue
		}
		if len(msg.ToolCalls) > 0 {
			return "", false
		}
		return msg.Content, msg.Content != ""
	}
	return "", false
}
