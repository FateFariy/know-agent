package llm

import (
	"context"
	"errors"
	"strings"

	"github.com/duke-git/lancet/v2/strutil"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/chat/adapter/model"
	"github.com/swiftbit/know-agent/internal/domain/document/logic/process/chunk"
	"github.com/swiftbit/know-agent/internal/domain/document/logic/process/chunk/recursive"
)

const (
	Name               = "LLM"                //  名称
	documentLlmSplit   = "document-llm-split" // 提示词模板名称
	defaultLLMMaxChars = 3500                 // 默认大模型最大字符数
)

// PromptRenderer 负责将 sourceText 渲染为大模型提示词
type PromptRenderer interface {
	// Render 渲染提示词
	Render(templateName string, variables map[string]any) (string, error)
}

// Chunker 大模型智能分块器
type Chunker struct {
	model     model.ChatModel
	renderer  PromptRenderer
	recursive *recursive.Chunker
	opt       *options
}

// NewChunker 创建大模型智能分块器
func NewChunker(model model.ChatModel, renderer PromptRenderer, opts ...chunk.Option) *Chunker {
	return &Chunker{
		opt: chunk.GetSpecificOptions(&options{
			llmSplitPrompt: documentLlmSplit,
			llmMaxChars:    defaultLLMMaxChars,
		}, opts...),
		model:    model,
		renderer: renderer,
		recursive: recursive.NewChunker(
			recursive.WithOverlapChars(0),
			recursive.WithMaxChars(defaultLLMMaxChars),
		),
	}
}

// Name 返回策略名称
func (s *Chunker) Name() string {
	return Name
}

// Chunk 执行大模型智能分块
func (s *Chunker) Chunk(ctx context.Context, input string, opts ...chunk.Option) ([]string, error) {
	text := strings.TrimSpace(input)
	if text == "" {
		return nil, nil
	}

	opt := chunk.GetSpecificOptions(s.opt, opts...)

	if !opt.enabled || s.model == nil {
		return nil, errors.New("大模型文本分块器已禁用或者模型为空")
	}

	// 输入过长 → 先以递归切块拆分到 LLM 上限
	var sourceTextList []string
	if utils.Len(text) > opt.llmMaxChars {
		sourceTextList, _ = s.recursive.Chunk(ctx, input, recursive.WithOverlapChars(0), recursive.WithMaxChars(opt.llmMaxChars))
	} else {
		sourceTextList = []string{text}
	}

	resultList := make([]string, 0, len(sourceTextList))
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
			resultList = append(resultList, trimmed)
		}
	}
	return resultList, nil
}

// split 调用大模型，从返回文本中解析 JSON 数组
func (s *Chunker) split(ctx context.Context, promptTempName, sourceText string) ([]string, error) {
	// 渲染提示词
	prompt, err := s.renderer.Render(promptTempName, map[string]any{"sourceText": sourceText})
	if err != nil || strutil.Trim(prompt) == "" {
		return nil, err
	}

	// 调用大模型
	content, err := s.model.Generate(ctx, "", prompt)
	if err != nil {
		return nil, err
	}

	result := make([]string, 0)
	if err = utils.Unmarshal(content, &result); err != nil {
		return nil, err
	}

	return result, nil
}
