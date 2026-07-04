package logic

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/duke-git/lancet/v2/slice"
	"github.com/zeromicro/go-zero/core/logx"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/config"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
	"github.com/swiftbit/know-agent/internal/svc"
)

// ChatModelImpl 可观测的聊天模型服务, 封装模型调用, 提供使用量统计、耗时追踪和错误记录能力
type ChatModelImpl[M adk.MessageType] struct {
	chatModel      model.BaseModel[M]
	config         *config.LLMConf
	defaultOptions *model.Options
}

// NewObservedChatModelImpl 创建可观测聊天模型实例
func NewObservedChatModelImpl[M adk.MessageType](svcCtx *svc.ServiceContext, chatModel model.BaseModel[M]) *ChatModelImpl[M] {
	conf := svcCtx.Config.ChatModel[resolveProvider(chatModel)]
	return &ChatModelImpl[M]{
		chatModel: chatModel,
		config:    conf,
		defaultOptions: &model.Options{
			Model:       &conf.Model,
			Temperature: &conf.Temperature,
			MaxTokens:   &conf.MaxTokens,
			TopP:        &conf.TopP,
		},
	}
}

// Generate 同步调用模型，返回文本响应
func (o *ChatModelImpl[M]) Generate(ctx context.Context, systemPrompt, userPrompt string, opts ...model.Option) (string, error) {
	// 调用底层模型执行生成
	response, err := o.chatModel.Generate(ctx, o.buildPrompt(systemPrompt, userPrompt), opts...)
	if err != nil {
		return "", err
	}

	// 从响应中提取文本内容
	responseText := extractResponseText(response)

	return responseText, nil
}

// GenerateWithTrace 同步调用模型，返回文本响应，同时记录使用量轨迹
func (o *ChatModelImpl[M]) GenerateWithTrace(ctx context.Context, stage, systemPrompt, userPrompt string, trace *vo.ConversationTrace, opts ...model.Option) (string, error) {
	startTime := time.Now()

	// 记录当前阶段的调用选项日志
	o.logStageCallOptions(stage, opts...)

	// 预先构建失败状态的使用量轨迹，便于异常时快速记录
	usageTrace := o.buildUsageTrace(stage, nil, startTime, "FAILED", systemPrompt, userPrompt, "")

	// 调用底层模型执行生成
	response, err := o.chatModel.Generate(ctx, o.buildPrompt(systemPrompt, userPrompt))
	if err != nil {
		// 调用失败，记录使用量并返回错误
		appendUsage(trace, usageTrace)
		return "", err
	}

	// 从响应中提取文本内容
	responseText := extractResponseText(response)

	// 构建成功状态的使用量轨迹
	usageTrace = o.buildUsageTrace(stage, response, startTime, "COMPLETED", systemPrompt, userPrompt, responseText)

	// 将使用量记录添加到追踪
	appendUsage(trace, usageTrace)

	return responseText, nil
}

// StreamWithTrace 流式调用模型，返回响应通道和错误，同时记录使用量轨迹
func (o *ChatModelImpl[M]) StreamWithTrace(ctx context.Context, stage, systemPrompt, userPrompt string, trace *vo.ConversationTrace, opts ...model.Option) (<-chan string, error) {
	startTime := time.Now()
	var outputBuilder strings.Builder
	resultChan := make(chan string, 100)

	// 记录当前阶段的调用选项日志
	o.logStageCallOptions(stage, opts...)

	// 调用底层模型建立流式连接
	stream, err := o.chatModel.Stream(ctx, o.buildPrompt(systemPrompt, userPrompt), opts...)
	if err != nil {
		// 连接建立失败，记录使用量并返回错误
		usageTrace := o.buildUsageTrace(stage, nil, startTime, "FAILED", systemPrompt, userPrompt, "")
		appendUsage(trace, usageTrace)
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
				usageTrace := o.buildUsageTrace(stage, chunk, startTime, "FAILED", systemPrompt, userPrompt, outputBuilder.String())
				appendUsage(trace, usageTrace)
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
					usageTrace := o.buildUsageTrace(stage, chunk, startTime, "FAILED", systemPrompt, userPrompt, outputBuilder.String())
					appendUsage(trace, usageTrace)
					logx.Alert("由外部终止调用...")
					return
				}
			}
		}

		// 流式处理结束，构建成功状态的使用量轨迹并记录
		usageTrace := o.buildUsageTrace(stage, chunk, startTime, "COMPLETED", systemPrompt, userPrompt, outputBuilder.String())
		appendUsage(trace, usageTrace)
	}()

	return resultChan, nil
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

// logStageCallOptions 记录阶段调用选项日志
func (o *ChatModelImpl[M]) logStageCallOptions(stage string, opts ...model.Option) {
	provider := resolveProvider(o.chatModel)
	modelName := utils.PointerOrDefault(o.defaultOptions.Model, "")
	if len(opts) == 0 {
		logx.Infof("模型调用参数: stage=%s, provider=%s, model=%s", stage, provider, modelName)
		return
	}
	options := model.GetCommonOptions(o.defaultOptions, opts...)

	temperature := "nil"
	if options.Temperature != nil {
		temperature = fmt.Sprintf("%.2f", *options.Temperature)
	}

	topP := "nil"
	if options.TopP != nil {
		topP = fmt.Sprintf("%.2f", *options.TopP)
	}

	logx.Infof("模型调用参数: stage=%s, provider=%s, model=%s, temperature=%s, topP=%s", stage, provider, modelName, temperature, topP)
}

// appendUsage 添加使用量记录
func appendUsage(trace *vo.ConversationTrace, usageTrace *vo.ChatModelUsageTrace) {
	if trace != nil && usageTrace != nil {
		trace.AddModelUsageTrace(usageTrace)
	}
}

// buildUsageTrace 构建使用量轨迹
func (o *ChatModelImpl[M]) buildUsageTrace(stage string, resp any, start time.Time, status, systemPrompt, userPrompt, responseText string) *vo.ChatModelUsageTrace {
	provider := resolveProvider(o.chatModel)
	tokenUsage := resolveTokenUsage(resp)

	var promptTokens, completionTokens, totalTokens int
	if tokenUsage != nil {
		promptTokens = tokenUsage.PromptTokens
		completionTokens = tokenUsage.CompletionTokens
		totalTokens = tokenUsage.TotalTokens
	}

	if promptTokens <= 0 {
		promptTokens = estimateTokens(systemPrompt) + estimateTokens(userPrompt)
	}
	if completionTokens <= 0 {
		completionTokens = estimateTokens(responseText)
	}
	if totalTokens <= 0 {
		totalTokens = promptTokens + completionTokens
	}

	estimatedCost := o.estimateCost(promptTokens, completionTokens)

	return &vo.ChatModelUsageTrace{
		StageName:        stage,
		Provider:         provider,
		Model:            utils.PointerOrDefault(o.defaultOptions.Model, ""),
		PromptTokens:     promptTokens,
		CompletionTokens: completionTokens,
		TotalTokens:      totalTokens,
		EstimatedCost:    estimatedCost,
		DurationMs:       time.Since(start).Milliseconds(),
		Status:           status,
	}
}

// resolveTokenUsage 解析Token使用量
func resolveTokenUsage(resp any) *schema.TokenUsage {
	switch expr := resp.(type) {
	case *schema.AgenticMessage:
		return expr.ResponseMeta.TokenUsage
	case *schema.Message:
		return expr.ResponseMeta.Usage
	}
	return nil
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
		return utils.Ternary(len(blocks) > 0, blocks[0].AssistantGenText.Text, "")
	default:
		return ""
	}
}

// estimateTokens 估算Token数量
func estimateTokens(content string) int {
	return (len(strings.TrimSpace(content)) + 3) / 4
}

// resolveProvider 解析模型提供商
func resolveProvider[M adk.MessageType](chatModel model.BaseModel[M]) string {
	if provider, ok := components.GetType(chatModel); ok {
		return provider
	}
	return "unknow"
}

// estimateCost 估算调用成本
func (o *ChatModelImpl[M]) estimateCost(promptTokens, completionTokens int) float64 {
	if promptTokens <= 0 && completionTokens <= 0 {
		return 0
	}
	promptCost := float64(promptTokens) / 1000.0 * o.config.InputTokenCost1k
	completionCost := float64(completionTokens) / 1000.0 * o.config.OutputTokenCost1k
	return promptCost + completionCost
}
