package agent

import (
	"context"
	"errors"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt/deep"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"

	"github.com/swiftbit/know-agent/common/logx"
	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/conversation"
	"github.com/swiftbit/know-agent/internal/svc"
)

// defaultInstruction deep agent 默认系统指令
const defaultInstruction = `你是 know-agent 的企业知识问答助手。
你必须严格基于给定证据回答，不要编造证据中没有出现的事实。
如果证据不足以支持明确结论，请直接说明资料不足。
如果需要检索企业内部知识，请调用 search_knowledge_base 工具。
如果引用了证据，请在对应句子末尾标注 [1][2] 这样的引用编号。`

// EinoAgentRunner 基于 eino deep agent 的 AgentRunner 实现
type EinoAgentRunner struct {
	name        string
	description string
	instruction string
	chatModel   model.BaseChatModel
	tools       []tool.BaseTool
	middlewares []adk.TypedChatModelAgentMiddleware[*schema.Message]

	runner  *adk.Runner
	initErr error
}

var _ conversation.AgentRunner = (*EinoAgentRunner)(nil)

// Option 配置 EinoAgentRunner 的函数式选项
type Option func(*EinoAgentRunner)

// WithName 设置 agent 名称与描述
func WithName(name, description string) Option {
	return func(r *EinoAgentRunner) {
		if utils.IsNotBlank(name) {
			r.name = name
		}
		if utils.IsNotBlank(description) {
			r.description = description
		}
	}
}

// WithInstruction 覆盖默认系统指令
func WithInstruction(instruction string) Option {
	return func(r *EinoAgentRunner) {
		if utils.IsNotBlank(instruction) {
			r.instruction = instruction
		}
	}
}

// WithTools 追加 eino 工具
func WithTools(tools ...conversation.Tool[any, any]) Option {
	return func(r *EinoAgentRunner) {
		einoTools := make([]tool.BaseTool, 0, len(tools))
		for _, t := range tools {
			einoTools = append(einoTools, NewEinoToolAdapter(t))
		}
		r.tools = append(r.tools, einoTools...)
	}
}

// WithMiddleware 追加领域层 Agent 中间件
func WithMiddleware(middlewares ...conversation.AgentMiddleware) Option {
	return func(r *EinoAgentRunner) {
		handlers := make([]adk.TypedChatModelAgentMiddleware[*schema.Message], 0, len(middlewares))
		for _, m := range middlewares {
			handlers = append(handlers, NewEinoMiddlewareAdapter(m))
		}
		r.middlewares = append(r.middlewares, handlers...)
	}
}

// NewEinoAgentRunner 构造 deep agent 执行器
func NewEinoAgentRunner(svcCtx *svc.ServiceContext, opts ...Option) *EinoAgentRunner {
	r := &EinoAgentRunner{
		name:        "know-agent-assistant",
		description: "一个可以调用知识库检索工具回答企业知识问题的助手。",
		instruction: defaultInstruction,
		chatModel:   svcCtx.ChatModel,
	}
	for _, o := range opts {
		o(r)
	}

	ctx := context.Background()
	agent, err := deep.NewTyped(ctx, r.buildConfig())
	if err != nil {
		panic("构建 deep agent 失败: " + err.Error())
	}
	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           agent,
		EnableStreaming: true,
	})
	r.runner = runner

	return r
}

// Run 执行一次 deep agent 会话，流式输出思考过程与最终回答。
// 结果与思考过程分别累积在 convCtx（Answer / thinkingSteps），并通过 Sink 实时推送前端。
func (r *EinoAgentRunner) Run(ctx context.Context, convCtx *conversation.Context) error {
	if convCtx == nil {
		return errors.New("convCtx 不能为空")
	}
	if utils.IsBlank(convCtx.Question) {
		return errors.New("用户问题不能为空")
	}
	// 注入 convCtx 到 ctx，供 eino 中间件（如 EinoAdapter）按需取用
	ctx = conversation.WithAgentContext(ctx, convCtx)

	iter := r.runner.Query(ctx, convCtx.Question)
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		if event.Err != nil {
			logx.Errorf("deep agent 事件错误: %v", event.Err)
			return event.Err
		}
		if event.Output == nil || event.Output.MessageOutput == nil {
			continue
		}
		if err := r.consumeMessageStream(convCtx, event.Output.MessageOutput.MessageStream); err != nil {
			return err
		}
	}

	return nil
}

// buildConfig 组装 deep agent 配置；无工具时不注册 ToolsNode。
func (r *EinoAgentRunner) buildConfig() *deep.TypedConfig[*schema.Message] {
	cfg := &deep.TypedConfig[*schema.Message]{
		Name:        r.name,
		Description: r.description,
		Instruction: r.instruction,
		ChatModel:   r.chatModel,
		Handlers:    r.middlewares,
	}
	if len(r.tools) > 0 {
		cfg.ToolsConfig = adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: r.tools,
			},
		}
	}
	return cfg
}

// consumeMessageStream 消费单个消息流
func (r *EinoAgentRunner) consumeMessageStream(convCtx *conversation.Context, stream *schema.StreamReader[*schema.Message]) error {
	if stream == nil {
		return nil
	}
	defer stream.Close()
	for {
		chunk, err := stream.Recv()
		if err != nil {
			break
		}
		if chunk == nil {
			continue
		}
		if utils.IsNotBlank(chunk.ReasoningContent) {
			if err = convCtx.PublishThinking(chunk.ReasoningContent); err != nil {
				return err
			}
		}
		if chunk.Content != "" {
			plan := convCtx.ExecutionPlan.Load()
			if plan == nil || plan.RetrievalResult == nil {
				return nil
			}
			refs := plan.RetrievalResult.FlattenReferences()
			if err = convCtx.PublishReferences(refs); err != nil {
				logx.Warnf("发布引用失败: %v", err)
			}
			if err = convCtx.PublishText(chunk.Content); err != nil {
				return err
			}
		}

	}
	return nil
}
