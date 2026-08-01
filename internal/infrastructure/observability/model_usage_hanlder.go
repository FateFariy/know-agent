package observability

import (
	"context"
	"sync"
	"time"

	"github.com/cloudwego/eino/schema"

	"github.com/swiftbit/know-agent/common/logx"
	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/config"
	"github.com/swiftbit/know-agent/internal/domain/callbacks"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
)

var _ callbacks.Handler = (*ModelUsageHandler)(nil)

var modelUsageRegisterOnce sync.Once

// ModelUsageHandler 模型使用量追踪 Handler，实现 callbacks.Handler 接口
type ModelUsageHandler struct {
	configs map[string]*config.LLMConf
}

// NewModelUsageHandler 创建模型使用量追踪 Handler 并注册为全局 Handler
func NewModelUsageHandler(configs map[string]*config.LLMConf) *ModelUsageHandler {
	h := &ModelUsageHandler{configs: configs}
	modelUsageRegisterOnce.Do(func() {
		callbacks.AppendGlobalHandlers(h)
	})
	return h
}

// OnStart 实现 callbacks.Handler，记录模型调用参数
func (h *ModelUsageHandler) OnStart(ctx context.Context, info *callbacks.RunInfo, input any) context.Context {
	if info.Component != "model_usage" {
		return ctx
	}
	modelInput, ok := input.(*vo.ModelCallInput)
	if !ok || modelInput == nil {
		return ctx
	}
	logStageCallOptions(modelInput)
	return ctx
}

// OnEnd 实现 callbacks.Handler，记录成功的模型使用量
func (h *ModelUsageHandler) OnEnd(ctx context.Context, info *callbacks.RunInfo, output any) context.Context {
	if info.Component != "model_usage" {
		return ctx
	}
	trace := vo.TraceFromCtx(ctx)
	if trace == nil {
		return ctx
	}
	modelOutput, ok := output.(*vo.ModelCallOutput)
	if !ok || modelOutput == nil {
		return ctx
	}
	usageTrace := h.buildUsageTrace(info, modelOutput, info.StartTime)
	trace.AddModelUsageTrace(usageTrace)
	return ctx
}

// OnError 实现 callbacks.Handler，记录失败的模型使用量
func (h *ModelUsageHandler) OnError(ctx context.Context, info *callbacks.RunInfo, _ error) context.Context {
	if info.Component != "model_usage" {
		return ctx
	}
	trace := vo.TraceFromCtx(ctx)
	if trace == nil {
		return ctx
	}
	usageTrace := h.buildFailedUsageTrace(info, info.StartTime)
	trace.AddModelUsageTrace(usageTrace)
	return ctx
}

// buildUsageTrace 构建成功的使用量轨迹
func (h *ModelUsageHandler) buildUsageTrace(info *callbacks.RunInfo, output *vo.ModelCallOutput, startTime time.Time) *vo.ChatModelUsageTrace {
	input, _ := info.StageCode.(*vo.ModelCallInput)

	tokenUsage := resolveTokenUsage(output.Response)

	var promptTokens, completionTokens, totalTokens int
	if tokenUsage != nil {
		promptTokens = tokenUsage.PromptTokens
		completionTokens = tokenUsage.CompletionTokens
		totalTokens = tokenUsage.TotalTokens
	}

	if promptTokens <= 0 && input != nil {
		promptTokens = utils.EstimateTokens(input.SystemPrompt) + utils.EstimateTokens(input.UserPrompt)
	}
	if completionTokens <= 0 {
		completionTokens = utils.EstimateTokens(output.ResponseText)
	}
	if totalTokens <= 0 {
		totalTokens = promptTokens + completionTokens
	}

	stageName := ""
	if input != nil {
		stageName = input.Stage
	}

	return &vo.ChatModelUsageTrace{
		StageName:        stageName,
		Provider:         h.resolveProvider(input),
		Model:            h.resolveModel(input),
		PromptTokens:     promptTokens,
		CompletionTokens: completionTokens,
		TotalTokens:      totalTokens,
		EstimatedCost:    h.estimateCost(input, promptTokens, completionTokens),
		DurationMs:       time.Since(startTime).Milliseconds(),
		Status:           "COMPLETED",
	}
}

// buildFailedUsageTrace 构建失败的使用量轨迹
func (h *ModelUsageHandler) buildFailedUsageTrace(info *callbacks.RunInfo, startTime time.Time) *vo.ChatModelUsageTrace {
	input, _ := info.StageCode.(*vo.ModelCallInput)
	stageName := ""
	if input != nil {
		stageName = input.Stage
	}
	return &vo.ChatModelUsageTrace{
		StageName:  stageName,
		Provider:   h.resolveProvider(input),
		Model:      h.resolveModel(input),
		DurationMs: time.Since(startTime).Milliseconds(),
		Status:     "FAILED",
	}
}

// resolveProvider 获取 provider
func (h *ModelUsageHandler) resolveProvider(input *vo.ModelCallInput) string {
	if input != nil && input.Provider != "" {
		return input.Provider
	}
	return "unknown"
}

// resolveModel 获取 model name
func (h *ModelUsageHandler) resolveModel(input *vo.ModelCallInput) string {
	if input != nil && input.ModelName != "" {
		return input.ModelName
	}
	return "unknown"
}

// estimateCost 估算调用成本
func (h *ModelUsageHandler) estimateCost(input *vo.ModelCallInput, promptTokens, completionTokens int) float64 {
	if promptTokens <= 0 && completionTokens <= 0 {
		return 0
	}
	provider := h.resolveProvider(input)
	conf, ok := h.configs[provider]
	if !ok || conf == nil {
		return 0
	}
	promptCost := float64(promptTokens) / 1000.0 * conf.InputTokenCost1k
	completionCost := float64(completionTokens) / 1000.0 * conf.OutputTokenCost1k
	return promptCost + completionCost
}

// logStageCallOptions 记录阶段调用选项日志
func logStageCallOptions(input *vo.ModelCallInput) {
	modelName := input.ModelName
	if input.Temperature == 0 && input.TopP == 0 {
		logx.Infof("模型调用参数: stage=%s, provider=%s, model=%s", input.Stage, input.Provider, modelName)
		return
	}
	logx.Infof("模型调用参数: stage=%s, provider=%s, model=%s, temperature=%.2f, topP=%.2f",
		input.Stage, input.Provider, modelName, input.Temperature, input.TopP)
}

// resolveTokenUsage 解析 Token 使用量
func resolveTokenUsage(resp any) *schema.TokenUsage {
	switch expr := resp.(type) {
	case *schema.AgenticMessage:
		if expr == nil || expr.ResponseMeta == nil {
			return nil
		}
		return expr.ResponseMeta.TokenUsage
	case *schema.Message:
		if expr == nil || expr.ResponseMeta == nil {
			return nil
		}
		return expr.ResponseMeta.Usage
	}
	return nil
}
