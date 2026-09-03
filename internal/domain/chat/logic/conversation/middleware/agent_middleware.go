package middleware

import (
	"context"

	"github.com/swiftbit/know-agent/internal/domain/chat/logic/conversation"
)

// BeforeAgentInput BeforeAgent 入参
type BeforeAgentInput struct {
	Instruction string // 当前系统指令
}

// BeforeAgentOutput BeforeAgent 出参
type BeforeAgentOutput struct {
	Instruction string // 修正后的系统指令
}

// AgentMiddleware 领域层 Agent 中间件接口
//
// 描述一次 deep agent 运行过程中领域关心的横切时机，签名不依赖 eino 框架类型：
//   - BeforeAgent：agent 启动前，允许修正系统指令（注入记忆摘要、当前日期、知识库范围等）
//   - AfterAgent：agent 正常进入终态后回调（收尾、澄清标记等）
type AgentMiddleware interface {
	// Name 中间件名称（日志与追踪使用）
	Name() string

	// BeforeAgent agent 启动前回调
	BeforeAgent(ctx context.Context, convCtx *conversation.Context, input *BeforeAgentInput) (*BeforeAgentOutput, error)

	// AfterAgent agent 正常进入终态后回调
	AfterAgent(ctx context.Context, convCtx *conversation.Context) error
}

// BaseAgentMiddleware 中间件空实现，嵌入后按需覆写单个时机即可
type BaseAgentMiddleware struct{}

// Name 默认空名称
func (b *BaseAgentMiddleware) Name() string { return "" }

// BeforeAgent 默认不改动指令
func (b *BaseAgentMiddleware) BeforeAgent(_ context.Context, _ *conversation.Context, input *BeforeAgentInput) (*BeforeAgentOutput, error) {
	if input == nil {
		return nil, nil
	}
	return &BeforeAgentOutput{Instruction: input.Instruction}, nil
}

// AfterAgent 默认无操作
func (b *BaseAgentMiddleware) AfterAgent(_ context.Context, _ *conversation.Context) error {
	return nil
}
