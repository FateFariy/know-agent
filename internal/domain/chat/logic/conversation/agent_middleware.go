package conversation

import "context"

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
//   - OnModelText：模型产出终态答复文本时的事件通知
//   - AfterAgent：agent 正常进入终态后回调（收尾、澄清标记等）
//
// 每个方法只描述一次「时机」，是否发布文本、吞并、与其他中间件协作等业务逻辑
// 由中间件实现自行决策。一个领域中间件对应挂载一个基础设施层适配器，多个中间件
// 的执行顺序由 eino Handlers 注册顺序保证。
type AgentMiddleware interface {
	// Name 中间件名称（日志与追踪使用）
	Name() string

	// BeforeAgent agent 启动前回调。Instruction 随输入结构体传入，修正后的指令
	// 经输出结构体返回；返回 nil 输出表示不改动指令。
	BeforeAgent(ctx context.Context, convCtx *Context, input *BeforeAgentInput) (*BeforeAgentOutput, error)

	// OnModelText 模型产出文本时的事件通知。文本是否发布（如调用 convCtx.PublishText）
	// 由中间件自行决策，适配层不代做。
	OnModelText(ctx context.Context, convCtx *Context, text string) error

	// AfterAgent agent 正常进入终态后回调。
	AfterAgent(ctx context.Context, convCtx *Context) error
}

// BaseAgentMiddleware 中间件空实现，嵌入后按需覆写单个时机即可
type BaseAgentMiddleware struct{}

// Name 默认空名称
func (b *BaseAgentMiddleware) Name() string { return "" }

// BeforeAgent 默认不改动指令
func (b *BaseAgentMiddleware) BeforeAgent(_ context.Context, _ *Context, input *BeforeAgentInput) (*BeforeAgentOutput, error) {
	if input == nil {
		return nil, nil
	}
	return &BeforeAgentOutput{Instruction: input.Instruction}, nil
}

// OnModelText 默认忽略文本
func (b *BaseAgentMiddleware) OnModelText(_ context.Context, _ *Context, _ string) error { return nil }

// AfterAgent 默认无操作
func (b *BaseAgentMiddleware) AfterAgent(_ context.Context, _ *Context) error { return nil }
