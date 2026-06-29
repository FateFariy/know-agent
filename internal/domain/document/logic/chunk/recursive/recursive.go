package recursive

import (
	"context"
	"unicode/utf8"

	"github.com/duke-git/lancet/v2/strutil"

	"github.com/swiftbit/know-agent/internal/domain/document/logic/chunk"
)

const (
	Name                = "RECURSIVE" // 策略名称
	defaultMaxChars     = 800         // 默认最大字符数
	defaultOverlapChars = 120         // 默认重叠字符数
)

// options 递归分块策略的参数配置
type options struct {
	maxChars     int
	overlapChars int
}

// Strategy 递归分块策略。
// 按优先级：段落 -> 行 -> 句子 -> 固定窗口，递归地将超长段落继续切分。
// 支持在相邻块之间保留一段重叠文本，用于避免切分位置丢失上下文。
type Strategy struct {
	opt *options
}

// NewStrategy 创建递归分块策略实例
func NewStrategy(opts ...chunk.Option) *Strategy {
	return &Strategy{
		opt: chunk.GetChunkImplSpecificOptions[options](nil, opts...),
	}
}

// WithMaxChars 设置单个块的最大字符数
func WithMaxChars(maxChars int) chunk.Option {
	return chunk.WrapChunkImplSpecificOptFn(func(o *options) {
		if maxChars <= 0 {
			maxChars = defaultMaxChars
		}
		o.maxChars = maxChars
	})
}

// WithOverlapChars 设置相邻块的重叠字符数
func WithOverlapChars(overlapChars int) chunk.Option {
	return chunk.WrapChunkImplSpecificOptFn(func(o *options) {
		if overlapChars < 0 {
			overlapChars = defaultOverlapChars
		}
		o.overlapChars = overlapChars
	})
}

// Name 返回策略名称
func (s *Strategy) Name() string {
	return Name
}

// Chunk 执行递归分块
func (s *Strategy) Chunk(ctx context.Context, input *chunk.Input, opts ...chunk.Option) ([]*chunk.Output, error) {
	if input == nil || strutil.Trim(input.Text) == "" {
		return nil, nil
	}

	// 允许通过 opts 覆盖原始配置
	opt := chunk.GetChunkImplSpecificOptions[options](s.opt, opts...)

	// 先按优先级切分为若干原始块
	rawChunks := s.split(strutil.Trim(input.Text), opt.maxChars, opt.overlapChars)

	result := make([]*chunk.Output, 0, len(rawChunks))
	for _, text := range rawChunks {
		trimmed := strutil.Trim(text)
		if trimmed == "" {
			continue
		}
		result = append(result, &chunk.Output{
			SectionPath:   strutil.Trim(input.SectionPath),
			CanonicalPath: strutil.Trim(input.CanonicalPath),
			ItemIndex:     input.ItemIndex,
			Text:          trimmed,
			SourceType:    input.SourceType,
		})
	}
	return result, nil
}

// split 递归切分主入口
func (s *Strategy) split(text string, maxChars, overlapChars int) []string {
	if text == "" {
		return nil
	}
	if utf8.RuneCountInString(text) <= maxChars {
		return []string{text}
	}

	// 按段落切分
	segmentList := chunk.SplitByRegex(text, chunk.ParagraphSplitRe())
	if len(segmentList) > 1 {
		return s.mergeAndSplit(segmentList, maxChars, overlapChars)
	}

	// 按换行切分
	segmentList = chunk.SplitByRegex(text, chunk.LineSplitRe())
	if len(segmentList) > 1 {
		return s.mergeAndSplit(segmentList, maxChars, overlapChars)
	}

	// 按句子切分
	segmentList = chunk.SplitSentences(text)
	if len(segmentList) > 1 {
		return s.mergeAndSplit(segmentList, maxChars, overlapChars)
	}

	// 最后兜底：固定窗口切分
	return chunk.FixedWindowSplit(text, maxChars, overlapChars)
}

// mergeAndSplit 将片段依次累加，超出 maxChars 时刷出一个块，然后继续
func (s *Strategy) mergeAndSplit(segmentList []string, maxChars, overlapChars int) []string {
	rawResultList := make([]string, 0, len(segmentList))
	current := make([]rune, 0, 512)

	for _, segment := range segmentList {
		trimmed := strutil.Trim(segment)
		if trimmed == "" {
			continue
		}

		if utf8.RuneCountInString(trimmed) > maxChars {
			// 当前片段过长：先刷出已累积的，然后递归该片段
			if len(current) > 0 {
				rawResultList = append(rawResultList, strutil.Trim(string(current)))
				current = current[:0]
			}
			rawResultList = append(rawResultList, s.split(trimmed, maxChars, overlapChars)...)
			continue
		}

		// 先刷出，再开启新块
		if len(current)+utf8.RuneCountInString(trimmed)+1 > maxChars {
			rawResultList = append(rawResultList, strutil.Trim(string(current)))
			current = current[:0]
		}

		if len(current) > 0 {
			current = append(current, '\n')
		}
		current = append(current, []rune(trimmed)...)
	}

	if len(current) > 0 {
		rawResultList = append(rawResultList, strutil.Trim(string(current)))
	}

	return chunk.ApplyOverlap(rawResultList, maxChars, overlapChars)
}
