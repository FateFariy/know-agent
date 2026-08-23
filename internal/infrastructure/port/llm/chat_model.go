package llm

import (
	"context"
	"errors"
	"io"
	"strings"
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components"
	eino "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/duke-git/lancet/v2/slice"

	"github.com/swiftbit/know-agent/common"
	"github.com/swiftbit/know-agent/common/logx"
	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/callbacks"
	"github.com/swiftbit/know-agent/internal/domain/chat/adapter/model"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
	"github.com/swiftbit/know-agent/internal/infrastructure/observability"
	"github.com/swiftbit/know-agent/internal/svc"
)

// ChatModelImpl 可观测的聊天模型服务, 封装模型调用, 提供使用量统计、耗时追踪和错误记录能力
type ChatModelImpl[M adk.MessageType] struct {
	chatModel  eino.BaseModel[M]
	judgeModel eino.BaseModel[M]
	provider   string
	options    *model.Options
}

// NewChatModelImpl 创建可观测聊天模型实例（AgenticMessage 变体，用于对话问答）
func NewChatModelImpl(svcCtx *svc.ServiceContext) *ChatModelImpl[*schema.AgenticMessage] {
	observability.NewModelUsageHandler(svcCtx.Config.ChatModel)

	provider := resolveProvider(svcCtx.ChatModel)
	conf := svcCtx.Config.ChatModel[provider]
	return &ChatModelImpl[*schema.AgenticMessage]{
		chatModel:  svcCtx.ChatModel,
		judgeModel: svcCtx.JudgeModel,
		provider:   provider,
		options: &model.Options{
			Function:    "chat",
			Model:       conf.Model,
			Temperature: &conf.Temperature,
			MaxTokens:   conf.MaxTokens,
			TopP:        &conf.TopP,
		},
	}
}

// Generate 同步调用模型，返回文本响应
func (o *ChatModelImpl[M]) Generate(ctx context.Context, systemPrompt, userPrompt string, opts ...common.Option) (string, error) {
	// 调用底层模型执行生成
	response, err := o.getChatModel(opts...).Generate(ctx, o.buildPrompt(systemPrompt, userPrompt), o.convertOptions(opts...)...)
	if err != nil {
		return "", err
	}

	// 从响应中提取文本内容
	responseText := extractResponseText(response)

	return responseText, nil
}

// GenerateWithTrace 同步调用模型，返回文本响应，同时记录使用量轨迹
func (o *ChatModelImpl[M]) GenerateWithTrace(ctx context.Context, stage, systemPrompt, userPrompt string, opts ...common.Option) (string, error) {
	meta, input := o.buildModelUsageInput(stage, systemPrompt, userPrompt, opts...)
	ctx = OnStart(ctx, meta, input)

	response, err := o.getChatModel(opts...).Generate(ctx, o.buildPrompt(systemPrompt, userPrompt), o.convertOptions(opts...)...)
	if err != nil {
		ctx = callbacks.OnError(ctx, err)
		return "", err
	}

	responseText := extractResponseText(response)
	ctx = callbacks.OnEnd(ctx, &vo.ModelCallOutput{
		SystemPrompt: systemPrompt,
		UserPrompt:   userPrompt,
		Response:     response,
		ResponseText: responseText,
	})
	return responseText, nil
}

// Stream 流式调用模型，返回响应通道和错误，同时记录使用量轨迹
func (o *ChatModelImpl[M]) Stream(ctx context.Context, stage, systemPrompt, userPrompt string, opts ...common.Option) (<-chan string, error) {
	meta, input := o.buildModelUsageInput(stage, systemPrompt, userPrompt, opts...)
	ctx = OnStart(ctx, meta, input)

	var outputBuilder strings.Builder
	resultChan := make(chan string, 100)

	// 调用底层模型建立流式连接
	stream, err := o.getChatModel(opts...).Stream(ctx, o.buildPrompt(systemPrompt, userPrompt), o.convertOptions(opts...)...)
	if err != nil {
		ctx = callbacks.OnError(ctx, err)
		return nil, err
	}

	// 在goroutine中处理流式响应
	go func() {
		// 确保通道在退出时关闭
		defer close(resultChan)
		defer stream.Close()

		var chunk any
		for {
			// 接收流式数据块
			chunk, err = stream.Recv()

			// 检查是否到达流的末尾
			if errors.Is(err, io.EOF) {
				break
			}

			// 处理接收过程中的错误
			if err != nil {
				ctx = callbacks.OnError(ctx, err)
				logx.Errorf("模型调用失败: %v", err)
				return
			}

			// 从数据块中提取文本
			text := extractResponseText(chunk)
			if text != "" {
				outputBuilder.WriteString(text)

				select {
				case resultChan <- text:
					// 成功发送到通道
				case <-ctx.Done():
					// 外部主动取消，记录终止日志和使用量
					ctx = callbacks.OnError(ctx, err)
					logx.Warn("由外部终止调用...")
					return
				}
			}
		}

		ctx = callbacks.OnEnd(ctx, &vo.ModelCallOutput{
			SystemPrompt: systemPrompt,
			UserPrompt:   userPrompt,
			Response:     chunk,
			ResponseText: outputBuilder.String(),
		})
	}()

	return resultChan, nil
}

// buildModelUsageInput 构建模型使用量追踪的元信息和输入参数
func (o *ChatModelImpl[M]) buildModelUsageInput(stage, systemPrompt, userPrompt string, opts ...common.Option) (*vo.ModelCallMeta, *vo.ModelCallInput) {
	options := common.GetImplSpecificOptions(o.options, opts...)
	meta := &vo.ModelCallMeta{
		Stage:     stage,
		Provider:  o.provider,
		ModelName: o.options.Model,
	}
	input := &vo.ModelCallInput{
		Temperature: utils.PointerOrDefault(options.Temperature, 0.0),
		TopP:        utils.PointerOrDefault(options.TopP, 0.0),
	}
	return meta, input
}

// buildPrompt 构建提示词
func (o *ChatModelImpl[M]) buildPrompt(systemPrompt, userPrompt string) []M {
	if userPrompt == "" {
		panic("userPrompt is empty")
	}
	var zero M
	switch any(zero).(type) {
	case *schema.AgenticMessage:
		messages := []*schema.AgenticMessage{
			schema.UserAgenticMessage(userPrompt),
		}
		if systemPrompt != "" {
			messages = append(messages, schema.SystemAgenticMessage(systemPrompt))
		}
		return any(messages).([]M)
	default:
		messages := []*schema.Message{
			schema.UserMessage(userPrompt),
		}
		if systemPrompt != "" {
			messages = append(messages, schema.SystemMessage(systemPrompt))
		}
		return any(messages).([]M)
	}
}

func (o *ChatModelImpl[M]) getChatModel(opts ...common.Option) eino.BaseModel[M] {
	opt := common.GetImplSpecificOptions(o.options, opts...)
	if opt.Function == "judge" {
		return o.judgeModel
	}
	return o.chatModel
}

// convertOptions 转换模型调用选项
func (o *ChatModelImpl[M]) convertOptions(opts ...common.Option) []eino.Option {
	options := make([]eino.Option, len(opts))
	opt := common.GetImplSpecificOptions(o.options, opts...)
	if opt.Model != "" {
		options = append(options, eino.WithModel(opt.Model))
	}
	if opt.Temperature != nil {
		options = append(options, eino.WithTemperature(*opt.Temperature))
	}
	if opt.TopP != nil {
		options = append(options, eino.WithTopP(*opt.TopP))
	}
	if opt.MaxTokens != 0 {
		options = append(options, eino.WithMaxTokens(opt.MaxTokens))
	}
	return options
}

// extractResponseText 提取响应文本
func extractResponseText(response any) string {
	if response == nil {
		return ""
	}
	switch resp := response.(type) {
	case *schema.Message:
		return resp.Content
	case *schema.AgenticMessage:
		blocks := slice.Filter(resp.ContentBlocks, func(index int, item *schema.ContentBlock) bool {
			return item.Type == schema.ContentBlockTypeAssistantGenText
		})
		if len(blocks) == 0 {
			return ""
		}
		return blocks[0].AssistantGenText.Text
	default:
		return ""
	}
}

// resolveProvider 解析模型提供商
func resolveProvider[M adk.MessageType](chatModel eino.BaseModel[M]) string {
	if provider, ok := components.GetType(chatModel); ok {
		return provider
	}
	return "unknow"
}

// OnStart 构造模型使用量的 RunInfo 并调用 callbacks.OnStart，meta 存入 Payload 供三阶段访问
func OnStart(ctx context.Context, meta *vo.ModelCallMeta, input *vo.ModelCallInput) context.Context {
	runInfo := &callbacks.RunInfo{
		StageId:   utils.GetSnowflakeNextID(),
		Payload:   meta,
		StartTime: time.Now(),
		Component: "model_usage",
	}
	ctx = callbacks.EnsureRunInfo(ctx, runInfo)
	ctx = callbacks.OnStart(ctx, input)
	return ctx
}
