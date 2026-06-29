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

// Strategy 递归分块策略, 按优先级：段落 -> 行 -> 句子 -> 固定窗口，递归地将超长段落继续切分, 支持在相邻块之间保留一段重叠文本
type Strategy struct {
	opt *options
}

// NewStrategy 创建递归分块策略实例
func NewStrategy(opts ...chunk.Option) *Strategy {
	return &Strategy{
		opt: chunk.GetChunkImplSpecificOptions(&options{
			maxChars:     defaultMaxChars,
			overlapChars: defaultOverlapChars,
		}, opts...),
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
	segmentList := chunk.SplitByRegex(text, chunk.ParagraphSplitRe)
	if len(segmentList) > 1 {
		return s.mergeAndSplit(segmentList, maxChars, overlapChars)
	}

	// 按换行切分
	segmentList = chunk.SplitByRegex(text, chunk.LineSplitRe)
	if len(segmentList) > 1 {
		return s.mergeAndSplit(segmentList, maxChars, overlapChars)
	}

	// 按句子切分
	segmentList = chunk.SplitSentences(text)
	if len(segmentList) > 1 {
		return s.mergeAndSplit(segmentList, maxChars, overlapChars)
	}

	// 最后兜底：固定窗口切分
	return s.fixedWindowSplit(text, maxChars, overlapChars)
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

	// 为块列表增加重叠前缀
	return s.applyOverlap(rawResultList, maxChars, overlapChars)
}

// applyOverlap 为块列表增加重叠前缀
func (s *Strategy) applyOverlap(rawChunkList []string, maxChars, overlapChars int) []string {
	if len(rawChunkList) == 0 || overlapChars <= 0 {
		return rawChunkList
	}

	overlappedChunkList := make([]string, 0, len(rawChunkList))
	for index, current := range rawChunkList {
		currentTrimmed := strutil.Trim(current)
		if currentTrimmed == "" {
			continue
		}
		if index == 0 {
			overlappedChunkList = append(overlappedChunkList, currentTrimmed)
			continue
		}
		// 为当前块增加重叠前缀
		previous := strutil.Trim(rawChunkList[index-1])
		overlapPrefix := s.buildOverlapPrefix(previous, currentTrimmed, maxChars, overlapChars)
		if overlapPrefix != "" {
			overlappedChunkList = append(overlappedChunkList, overlapPrefix+"\n"+currentTrimmed)
		} else {
			overlappedChunkList = append(overlappedChunkList, currentTrimmed)
		}
	}
	return overlappedChunkList
}

// buildOverlapPrefix 取 previous 尾部作为重叠前缀，受 maxChars 约束
func (s *Strategy) buildOverlapPrefix(previous, current string, maxChars, overlapChars int) string {
	previous = strutil.Trim(previous)
	current = strutil.Trim(current)
	if previous == "" || current == "" {
		return ""
	}

	// 计算允许的重叠字符数，重叠字符数不能超过 maxChars，也不能超过 previous 的长度
	allowed := min(overlapChars, max(0, maxChars-utf8.RuneCountInString(current)-1))
	if allowed <= 0 {
		return ""

	}
	// 取 previous 尾部 allowed 个字符作为重叠前缀
	prevRunes := []rune(previous)
	startIdx := max(len(prevRunes)-allowed, 0)

	return strutil.Trim(string(prevRunes[startIdx:]))
}

// fixedWindowSplit 固定窗口切分超长文本
func (s *Strategy) fixedWindowSplit(text string, maxChars, overlapChars int) []string {
	trim := strutil.Trim(text)
	total := utf8.RuneCountInString(trim)
	if total == 0 {
		return nil
	}
	if total <= maxChars {
		return []string{trim}
	}

	runes := []rune(trim)
	result := make([]string, 0, total/maxChars+1)
	step := max(1, maxChars-overlapChars)
	for start := 0; start < total; start += step {
		end := min(start+maxChars, total)
		result = append(result, strutil.Trim(string(runes[start:end])))
		if end >= total {
			break
		}
	}
	return result
}
