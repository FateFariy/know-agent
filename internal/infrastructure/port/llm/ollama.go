package llm

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/zeromicro/go-zero/rest/httpc"

	"github.com/swiftbit/know-agent/common"
	"github.com/swiftbit/know-agent/common/logx"
	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/callbacks"
	"github.com/swiftbit/know-agent/internal/domain/chat/adapter/model"
	"github.com/swiftbit/know-agent/internal/svc"
)

// OllamaModel 通过 Ollama 的 HTTP 接口调用本地部署的对话模型。
// 同步生成使用 /api/generate（https://docs.ollama.com/api/generate），
// 流式生成使用 /api/generate?stream=true（https://docs.ollama.com/api/chat 同款 NDJSON 流）。
type OllamaModel struct {
	endpoint string
	options  *model.Options
}

// ollamaGenerateReq 对应 Ollama /api/generate 请求体
type ollamaGenerateReq struct {
	ContentType string         `header:"Content-Type"`
	Model       string         `json:"model"`
	Prompt      string         `json:"prompt"`
	System      string         `json:"system,omitempty"`
	Stream      bool           `json:"stream"`
	Think       string         `json:"think,omitempty"`
	Options     *ollamaOptions `json:"options,omitempty"`
}

// ollamaOptions 对应 Ollama 的 options 采样参数（仅发送非默认值）
type ollamaOptions struct {
	Temperature *float32 `json:"temperature,omitempty"`
	TopP        *float32 `json:"top_p,omitempty"`
	NumPredict  int      `json:"num_predict,omitempty"`
}

// ollamaGenerateResp 对应 Ollama /api/generate 响应体（流式时按 NDJSON 逐行返回同结构）
type ollamaGenerateResp struct {
	Model      string `json:"model"`
	CreatedAt  string `json:"created_at"`
	Response   string `json:"response"`
	Done       bool   `json:"done"`
	DoneReason string `json:"done_reason"`
}

// NewOllamaModel 创建一个调用 Ollama 的模型客户端。
// baseURL 取自 ChatModel 配置中 "Ollama" 键的 BaseURL，缺失时回退到 http://localhost:11434。
func NewOllamaModel(svcCtx *svc.ServiceContext) *OllamaModel {
	conf := svcCtx.Config.ChatModel["Ollama"]
	baseURL := "http://localhost:11434"
	if conf != nil && conf.BaseURL != "" {
		baseURL = conf.BaseURL
	}

	options := &model.Options{Function: "chat"}
	if conf != nil {
		options.Model = conf.Model
		options.Temperature = &conf.Temperature
		options.MaxTokens = conf.MaxTokens
		options.TopP = &conf.TopP
		options.Think = "true"
	}

	return &OllamaModel{
		endpoint: strings.TrimRight(baseURL, "/") + "/api/generate",
		options:  options,
	}
}

// Generate 同步调用模型，返回文本响应
func (o *OllamaModel) Generate(ctx context.Context, systemPrompt, userPrompt string, opts ...common.Option) (string, error) {
	resp, err := o.doGenerate(ctx, systemPrompt, userPrompt, opts...)
	if err != nil {
		return "", err
	}
	return resp.Response, nil
}

// GenerateWithTrace 同步调用模型并返回文本响应，同时记录使用量轨迹
func (o *OllamaModel) GenerateWithTrace(ctx context.Context, stage, systemPrompt, userPrompt string, opts ...common.Option) (string, error) {
	opt := common.GetImplSpecificOptions(o.options, opts...)
	meta, input := buildTraceMeta(stage, "ollama", opt), buildTraceInput(opt)
	ctx = OnStart(ctx, meta, input)

	resp, err := o.doGenerate(ctx, systemPrompt, userPrompt, opts...)
	if err != nil {
		ctx = callbacks.OnError(ctx, err)
		return "", err
	}

	ctx = callbacks.OnEnd(ctx, &ModelCallOutput{
		SystemPrompt: systemPrompt,
		UserPrompt:   userPrompt,
		ResponseText: resp.Response,
	})
	return resp.Response, nil
}

// Stream 流式调用模型，返回响应通道，同时记录使用量轨迹
func (o *OllamaModel) Stream(ctx context.Context, stage, systemPrompt, userPrompt string, opts ...common.Option) (<-chan string, error) {
	opt := common.GetImplSpecificOptions(o.options, opts...)
	meta, input := buildTraceMeta(stage, "ollama", opt), buildTraceInput(opt)
	ctx = OnStart(ctx, meta, input)

	reqBody := o.buildReq(systemPrompt, userPrompt, true, opts...)

	httpResp, err := httpc.Do(ctx, http.MethodPost, o.endpoint, reqBody)
	if err != nil {
		ctx = callbacks.OnError(ctx, err)
		return nil, fmt.Errorf("call ollama generate: %w", err)
	}
	if httpResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(httpResp.Body)
		_ = httpResp.Body.Close()
		err = fmt.Errorf("ollama generate failed: status=%d body=%s", httpResp.StatusCode, string(body))
		ctx = callbacks.OnError(ctx, err)
		return nil, err
	}

	resultChan := make(chan string, 100)
	go func() {
		defer close(resultChan)
		defer func() { _ = httpResp.Body.Close() }()

		scanner := bufio.NewScanner(httpResp.Body)
		scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

		var full strings.Builder
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}
			var chunk ollamaGenerateResp
			if err = json.Unmarshal([]byte(line), &chunk); err != nil {
				ctx = callbacks.OnError(ctx, err)
				logx.Errorf("解析 ollama 流式响应失败: %v", err)
				return
			}
			if chunk.Response != "" {
				full.WriteString(chunk.Response)
				select {
				case resultChan <- chunk.Response:
				case <-ctx.Done():
					ctx = callbacks.OnError(ctx, ctx.Err())
					logx.Warn("由外部终止调用...")
					return
				}
			}
			if chunk.Done {
				break
			}
		}
		if err = scanner.Err(); err != nil {
			ctx = callbacks.OnError(ctx, err)
			logx.Errorf("读取 ollama 流式响应失败: %v", err)
			return
		}

		ctx = callbacks.OnEnd(ctx, &ModelCallOutput{
			SystemPrompt: systemPrompt,
			UserPrompt:   userPrompt,
			ResponseText: full.String(),
		})
	}()

	return resultChan, nil
}

// doGenerate 向 Ollama 发起 /api/generate 非流式请求并解析结果
func (o *OllamaModel) doGenerate(ctx context.Context, systemPrompt, userPrompt string, opts ...common.Option) (*ollamaGenerateResp, error) {
	reqBody := o.buildReq(systemPrompt, userPrompt, false, opts...)

	httpResp, err := httpc.Do(ctx, http.MethodPost, o.endpoint, reqBody)
	if err != nil {
		return nil, fmt.Errorf("call ollama generate: %w", err)
	}
	defer func() { _ = httpResp.Body.Close() }()

	if httpResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(httpResp.Body)
		return nil, fmt.Errorf("ollama generate failed: status=%d body=%s", httpResp.StatusCode, string(body))
	}

	var resp ollamaGenerateResp
	if err = json.NewDecoder(httpResp.Body).Decode(&resp); err != nil {
		return nil, fmt.Errorf("parse ollama generate response: %w", err)
	}
	return &resp, nil
}

// buildReq 按当前选项构造 Ollama 请求体
func (o *OllamaModel) buildReq(systemPrompt, userPrompt string, stream bool, opts ...common.Option) *ollamaGenerateReq {
	opt := common.GetImplSpecificOptions(o.options, opts...)
	req := &ollamaGenerateReq{
		ContentType: "application/json",
		Model:       opt.Model,
		Prompt:      userPrompt,
		Stream:      stream,
		System:      systemPrompt,
		Think:       opt.Think,
	}

	options := &ollamaOptions{}
	if opt.Temperature != nil {
		options.Temperature = opt.Temperature
	}
	if opt.TopP != nil {
		options.TopP = opt.TopP
	}
	if opt.MaxTokens != 0 {
		options.NumPredict = opt.MaxTokens
	}
	req.Options = options

	return req
}

// buildTraceMeta 构造模型使用量轨迹的元信息
func buildTraceMeta(stage, provider string, opt *model.Options) *ModelCallMeta {
	return &ModelCallMeta{
		Stage:     stage,
		Provider:  provider,
		ModelName: opt.Model,
	}
}

// buildTraceInput 构造模型使用量轨迹的输入参数
func buildTraceInput(opt *model.Options) *ModelCallInput {
	return &ModelCallInput{
		Temperature: utils.PointerOrDefault(opt.Temperature, 0.0),
		TopP:        utils.PointerOrDefault(opt.TopP, 0.0),
	}
}
