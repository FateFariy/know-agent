package chunk

import (
	"context"
	"strings"
	"unicode/utf8"
)

// LLMModel 大模型服务的最小接口，由调用方实现
// 这样 chunk 包本身不依赖具体的聊天模型实现，便于测试和替换
type LLMModel interface {
	// Generate 同步调用大模型，输入 prompt，返回文本响应
	Generate(ctx context.Context, prompt string) (string, error)
}

// LLMPromptRenderer 负责将 sourceText 渲染为大模型提示词
// 实现可自由选择模板渲染或手写提示。用于支持不同提示模板和测试桩
type LLMPromptRenderer interface {
	// Render 渲染提示词
	Render(ctx context.Context, sourceText string) (string, error)
}

// defaultLLMRenderer 默认的提示词渲染器
// 采用简洁的 JSON 数组要求：请将输入文本按语义拆分为若干段落，以 JSON 字符串数组返回
type defaultLLMRenderer struct{}

// NewDefaultLLMRenderer 创建默认提示词渲染器
func NewDefaultLLMRenderer() LLMPromptRenderer {
	return &defaultLLMRenderer{}
}

// Render 实现 LLMPromptRenderer
func (d *defaultLLMRenderer) Render(_ context.Context, sourceText string) (string, error) {
	return strings.TrimSpace(sourceText) + "\n\n请将上面的文本按语义切分为多个段落，并以 JSON 字符串数组返回，例如：\n[\"段落一的内容\",\"段落二的内容\"]", nil
}

// LLMStrategy 大模型智能分块策略
// 当启用时将过长的文本先按递归分块切成 <= llmMaxChars 的片段，再逐个调用大模型
// 当大模型失败或未启用时，降级为语义分块
type LLMStrategy struct {
	base     baseOptions
	model    LLMModel          // 大模型服务实现，可为 nil（此时降级为语义分块）
	renderer LLMPromptRenderer // 提示词渲染器，默认使用 defaultLLMRenderer
}

// NewLLMStrategy 创建大模型智能分块策略
func NewLLMStrategy(model LLMModel, renderer LLMPromptRenderer, opts ...StrategyOption) *LLMStrategy {
	if renderer == nil {
		renderer = NewDefaultLLMRenderer()
	}
	return &LLMStrategy{
		base:     applyOptions(opts),
		model:    model,
		renderer: renderer,
	}
}

// Name 返回策略名称
func (s *LLMStrategy) Name() string {
	return "LLM"
}

// Chunk 执行大模型智能分块
func (s *LLMStrategy) Chunk(ctx context.Context, input *Input, pipelineType PipelineType) ([]*Output, error) {
	// 未启用或未配置大模型：直接降级为语义分块
	if !s.base.llmEnabled || s.model == nil {
		fallback := NewSemanticStrategy(
			WithSemantic(s.base.semanticMinChars, s.base.semanticMaxChars, s.base.semanticSimilarityThreshold),
		)
		return fallback.Chunk(ctx, input, pipelineType)
	}

	if input == nil || strings.TrimSpace(input.Text) == "" {
		return []*Output{}, nil
	}

	llmMaxChars := s.resolveLlmMaxChars(pipelineType)

	// 对超长文本先做递归分块，再逐个调用大模型
	var sourceTextList []string
	if utf8.RuneCountInString(input.Text) > llmMaxChars {
		recursive := NewRecursiveStrategy(WithRecursive(llmMaxChars, 0))
		rawChunks, err := recursive.Chunk(ctx, input, pipelineType)
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

	resultList := make([]*Output, 0, len(sourceTextList))
	for _, sourceText := range sourceTextList {
		llmChunks := s.llmSplit(ctx, sourceText)
		// 大模型调用失败：降级为语义分块
		if len(llmChunks) == 0 {
			fallback := NewSemanticStrategy(
				WithSemantic(s.base.semanticMinChars, s.base.semanticMaxChars, s.base.semanticSimilarityThreshold),
			)
			fallbackInput := &Input{
				SectionPath:   input.SectionPath,
				CanonicalPath: input.CanonicalPath,
				ItemIndex:     input.ItemIndex,
				Text:          sourceText,
				SourceType:    input.SourceType,
			}
			fallbackChunks, err := fallback.Chunk(ctx, fallbackInput, pipelineType)
			if err != nil {
				return nil, err
			}
			resultList = append(resultList, fallbackChunks...)
			continue
		}
		for _, chunkText := range llmChunks {
			trimmed := strings.TrimSpace(chunkText)
			if trimmed == "" {
				continue
			}
			resultList = append(resultList, &Output{
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

// llmSplit 调用大模型，从返回文本中解析 JSON 数组
func (s *LLMStrategy) llmSplit(ctx context.Context, sourceText string) []string {
	prompt, err := s.renderer.Render(ctx, sourceText)
	if err != nil || strings.TrimSpace(prompt) == "" {
		return []string{}
	}

	content, err := s.model.Generate(ctx, prompt)
	if err != nil || strings.TrimSpace(content) == "" {
		return []string{}
	}

	return parseStringJsonArrayFrom(content)
}

// resolveLlmMaxChars 根据流水线返回大模型单次调用允许的最大字符数
func (s *LLMStrategy) resolveLlmMaxChars(pipelineType PipelineType) int {
	if pipelineType == PipelineTypeParent {
		return max(s.base.llmMaxChars, parentBlockMaxChars)
	}
	return s.base.llmMaxChars
}

// parseStringJsonArrayFrom 从文本中抽取 JSON 数组，并解析其字符串元素
func parseStringJsonArrayFrom(content string) []string {
	startIdx := strings.Index(content, "[")
	endIdx := strings.LastIndex(content, "]")
	if startIdx < 0 || endIdx <= startIdx {
		return []string{}
	}
	inner := content[startIdx : endIdx+1]
	return parseStringJsonArray(inner)
}

// parseStringJsonArray 简易解析 JSON 字符串数组（仅处理双引号字符串元素）
func parseStringJsonArray(content string) []string {
	trimmed := strings.TrimSpace(content)
	if !strings.HasPrefix(trimmed, "[") || !strings.HasSuffix(trimmed, "]") {
		return []string{}
	}
	inner := strings.TrimSpace(trimmed[1 : len(trimmed)-1])
	if inner == "" {
		return []string{}
	}

	result := make([]string, 0)
	runes := []rune(inner)
	n := len(runes)
	i := 0
	for i < n {
		// 跳过空白和逗号
		for i < n && (runes[i] == ',' || runes[i] == ' ' || runes[i] == '\t' || runes[i] == '\r' || runes[i] == '\n') {
			i++
		}
		if i >= n {
			break
		}
		if runes[i] != '"' {
			// 跳过非字符串元素直到下一个逗号
			for i < n && runes[i] != ',' {
				i++
			}
			continue
		}
		i++ // 跳过 "
		sb := strings.Builder{}
		for i < n {
			r := runes[i]
			if r == '\\' && i+1 < n {
				next := runes[i+1]
				switch next {
				case '"':
					sb.WriteByte('"')
				case '\\':
					sb.WriteByte('\\')
				case '/':
					sb.WriteByte('/')
				case 'n':
					sb.WriteByte('\n')
				case 't':
					sb.WriteByte('\t')
				case 'r':
					sb.WriteByte('\r')
				default:
					sb.WriteRune(r)
					sb.WriteRune(next)
				}
				i += 2
				continue
			}
			if r == '"' {
				i++
				break
			}
			sb.WriteRune(r)
			i++
		}
		result = append(result, sb.String())
	}
	return result
}
