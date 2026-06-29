package llm

import (
	"context"
	"strings"
	"unicode/utf8"

	"github.com/duke-git/lancet/v2/strutil"

	"github.com/swiftbit/know-agent/internal/domain/document/logic/chunk"
	chunkRecursive "github.com/swiftbit/know-agent/internal/domain/document/logic/chunk/recursive"
	chunkSemantic "github.com/swiftbit/know-agent/internal/domain/document/logic/chunk/semantic"
)

const (
	// Name 策略名称，用于注册和日志标识
	Name = "LLM"

	// defaultMaxChars 默认的单次调用大模型允许的最大字符数
	defaultMaxChars = 3500
)

// LLMModel 大模型服务的最小接口，由调用方实现。
// 这样 llm 子包不依赖具体聊天模型实现，便于测试和替换。
type LLMModel interface {
	// Generate 同步调用大模型，输入 prompt，返回文本响应
	Generate(ctx context.Context, prompt string) (string, error)
}

// LLMPromptRenderer 负责将 sourceText 渲染为大模型提示词。
// 实现可自由选择模板渲染或手写提示，用于支持不同提示模板与测试桩。
type LLMPromptRenderer interface {
	// Render 渲染提示词
	Render(ctx context.Context, sourceText string) (string, error)
}

// defaultRenderer 默认的提示词渲染器
type defaultRenderer struct{}

// NewDefaultRenderer 创建默认提示词渲染器
func NewDefaultRenderer() LLMPromptRenderer {
	return &defaultRenderer{}
}

// Render 实现 LLMPromptRenderer
func (d *defaultRenderer) Render(_ context.Context, sourceText string) (string, error) {
	return strings.TrimSpace(sourceText) + "\n\n请将上面的文本按语义切分为多个段落，并以 JSON 字符串数组返回，例如：\n[\"段落一的内容\",\"段落二的内容\"]", nil
}

// options LLM 分块策略的参数配置
type options struct {
	maxChars int
	enabled  bool
}

// Strategy 大模型智能分块策略。
// 当启用时将过长的文本先按递归分块切成 <= maxChars 的片段，再逐个调用大模型。
// 当大模型失败或未启用时，降级为语义分块。
type Strategy struct {
	opt      options
	model    LLMModel
	renderer LLMPromptRenderer
}

// NewStrategy 创建大模型智能分块策略
func NewStrategy(model LLMModel, renderer LLMPromptRenderer, opts ...chunk.Option) *Strategy {
	if renderer == nil {
		renderer = NewDefaultRenderer()
	}
	applied := chunk.GetChunkImplSpecificOptions[options](nil, opts...)
	return &Strategy{
		opt:      *applied,
		model:    model,
		renderer: renderer,
	}
}

// WithEnabled 设置是否启用大模型智能分块
func WithEnabled(enabled bool) chunk.Option {
	return chunk.WrapChunkImplSpecificOptFn(func(o *options) {
		o.enabled = enabled
	})
}

// WithMaxChars 设置单次调用大模型允许的最大字符数
func WithMaxChars(maxChars int) chunk.Option {
	return chunk.WrapChunkImplSpecificOptFn(func(o *options) {
		o.maxChars = maxChars
	})
}

// Name 返回策略名称
func (s *Strategy) Name() string {
	return Name
}

// Chunk 执行大模型智能分块
func (s *Strategy) Chunk(ctx context.Context, input *chunk.Input, opts ...chunk.Option) ([]*chunk.Output, error) {
	// 未启用或未配置大模型：直接降级为语义分块
	if !s.opt.enabled || s.model == nil {
		fallback := chunkSemantic.NewStrategy(
			chunkSemantic.WithMinChars(240),
			chunkSemantic.WithMaxChars(700),
			chunkSemantic.WithSimilarityThreshold(0.18),
		)
		return fallback.Chunk(ctx, input)
	}

	if input == nil || strings.TrimSpace(input.Text) == "" {
		return nil, nil
	}

	llmMaxChars := s.opt.maxChars
	if llmMaxChars <= 0 {
		llmMaxChars = defaultMaxChars
	}

	// 对超长文本先做递归分块，再逐个调用大模型
	var sourceTextList []string
	if utf8.RuneCountInString(input.Text) > llmMaxChars {
		recursiveStrategy := chunkRecursive.NewStrategy(
			chunkRecursive.WithMaxChars(llmMaxChars),
			chunkRecursive.WithOverlapChars(0),
		)
		rawChunks, err := recursiveStrategy.Chunk(ctx, input)
		if err != nil {
			return nil, err
		}
		sourceTextList = make([]string, 0, len(rawChunks))
		for _, c := range rawChunks {
			if c == nil {
				continue
			}
			if t := strings.TrimSpace(c.Text); t != "" {
				sourceTextList = append(sourceTextList, t)
			}
		}
	} else {
		sourceTextList = []string{strings.TrimSpace(input.Text)}
	}

	resultList := make([]*chunk.Output, 0, len(sourceTextList))
	for _, sourceText := range sourceTextList {
		chunks := s.split(ctx, sourceText)
		// 大模型调用失败：降级为语义分块
		if len(chunks) == 0 {
			fallback := chunkSemantic.NewStrategy(
				chunkSemantic.WithMinChars(240),
				chunkSemantic.WithMaxChars(700),
				chunkSemantic.WithSimilarityThreshold(0.18),
			)
			fallbackInput := &chunk.Input{
				SectionPath:   input.SectionPath,
				CanonicalPath: input.CanonicalPath,
				ItemIndex:     input.ItemIndex,
				Text:          sourceText,
				SourceType:    input.SourceType,
			}
			fallbackChunks, err := fallback.Chunk(ctx, fallbackInput)
			if err != nil {
				return nil, err
			}
			resultList = append(resultList, fallbackChunks...)
			continue
		}
		for _, chunkText := range chunks {
			trimmed := strings.TrimSpace(chunkText)
			if trimmed == "" {
				continue
			}
			resultList = append(resultList, &chunk.Output{
				SectionPath:   strings.TrimSpace(input.SectionPath),
				CanonicalPath: strings.TrimSpace(input.CanonicalPath),
				ItemIndex:     input.ItemIndex,
				Text:          trimmed,
				SourceType:    input.SourceType,
			})
		}
	}
	return resultList, nil
}

// split 调用大模型，从返回文本中解析 JSON 数组
func (s *Strategy) split(ctx context.Context, sourceText string) []string {
	prompt, err := s.renderer.Render(ctx, sourceText)
	if err != nil || strings.TrimSpace(prompt) == "" {
		return nil
	}

	content, err := s.model.Generate(ctx, prompt)
	if err != nil || strings.TrimSpace(content) == "" {
		return nil
	}

	return chunk.ParseStringJSONArrayFrom(content)
}

// 避免 "strutil" 未使用告警：显式在编译期使用
var _ = strutil.Trim
