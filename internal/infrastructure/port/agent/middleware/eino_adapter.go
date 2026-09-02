package middleware

import (
	"context"

	"github.com/cloudwego/eino/adk"

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

// AfterAgent 在 agent 正常进入终态后通知领域中间件
func (a *EinoAdapter) AfterAgent(ctx context.Context, state *adk.ChatModelAgentState) (context.Context, error) {
	if a.middleware == nil {
		return ctx, nil
	}
	if err := a.middleware.AfterAgent(ctx, conversation.AgentContextFrom(ctx)); err != nil {
		return ctx, err
	}
	return ctx, nil
}
