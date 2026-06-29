package llm

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/cloudwego/eino/components/model"
	"github.com/duke-git/lancet/v2/strutil"

	"github.com/swiftbit/know-agent/internal/domain/document/logic/chunk"
)

const (
	Name             = "LLM"                //  策略名称
	documentLlmSplit = "document-llm-split" // 提示词模板名称
)

// ChatModel 大模型服务的最小接口
type ChatModel interface {
	// Generate 同步调用大模型
	Generate(ctx context.Context, systemPrompt, userPrompt string, opts ...model.Option) (string, error)
}

// PromptRenderer 负责将 sourceText 渲染为大模型提示词
type PromptRenderer interface {
	// Render 渲染提示词
	Render(templateName string, variables map[string]any) (string, error)
}

// Strategy 大模型智能分块策略
type Strategy struct {
	model    ChatModel
	renderer PromptRenderer
	opt      *options
}

type options struct {
	llmSplitPrompt string
}

// NewStrategy 创建大模型智能分块策略
func NewStrategy(model ChatModel, renderer PromptRenderer, opts ...chunk.Option) *Strategy {
	return &Strategy{
		opt: chunk.GetChunkImplSpecificOptions(&options{
			llmSplitPrompt: documentLlmSplit,
		}, opts...),
		model:    model,
		renderer: renderer,
	}
}

// Name 返回策略名称
func (s *Strategy) Name() string {
	return Name
}

// Chunk 执行大模型智能分块
func (s *Strategy) Chunk(ctx context.Context, input *chunk.Input, opts ...chunk.Option) ([]*chunk.Output, error) {
	if input == nil || strutil.Trim(input.Text) == "" {
		return nil, nil
	}

	opt := chunk.GetChunkImplSpecificOptions(s.opt, opts...)

	sourceTextList := []string{strutil.Trim(input.Text)}
	resultList := make([]*chunk.Output, 0, len(sourceTextList))
	for _, sourceText := range sourceTextList {
		chunks, err := s.split(ctx, opt.llmSplitPrompt, sourceText)
		if err != nil {
			return nil, err
		}
		for _, chunkText := range chunks {
			trimmed := strutil.Trim(chunkText)
			if trimmed == "" {
				continue
			}
			resultList = append(resultList, &chunk.Output{
				SectionPath:   strutil.Trim(input.SectionPath),
				CanonicalPath: strutil.Trim(input.CanonicalPath),
				ItemIndex:     input.ItemIndex,
				Text:          trimmed,
				SourceType:    input.SourceType,
			})
		}
	}
	return resultList, nil
}

// split 调用大模型，从返回文本中解析 JSON 数组
func (s *Strategy) split(ctx context.Context, promptTempName, sourceText string) ([]string, error) {
	// 渲染提示词
	prompt, err := s.renderer.Render(promptTempName, map[string]any{"text": sourceText})
	if err != nil || strutil.Trim(prompt) == "" {
		return nil, err
	}

	// 调用大模型
	content, err := s.model.Generate(ctx, "", prompt)
	if err != nil {
		return nil, err
	}

	// 从文本中抽取 JSON 数组，并解析其字符串元素
	startIdx := strings.Index(content, "[")
	endIdx := strings.LastIndex(content, "]")
	if startIdx < 0 || endIdx <= startIdx {
		return nil, nil
	}
	result := make([]string, 0)
	if err = json.Unmarshal([]byte(content[startIdx:endIdx+1]), &result); err != nil {
		return nil, err
	}

	return result, nil
}
