package semantic

import (
	"context"
	"unicode/utf8"

	"github.com/duke-git/lancet/v2/strutil"

	"github.com/swiftbit/know-agent/internal/domain/document/logic/chunk"
)

const (
	Name                       = "SEMANTIC" // 策略名称，用于注册和日志标识
	defaultMaxChars            = 700        // 默认单个块最大字符数
	defaultMinChars            = 240        // 触发语义切分的最小字符数
	defaultSimilarityThreshold = 0.18       // 默认的 Jaccard 相似度阈值
)

// options 语义分块策略的参数配置
type options struct {
	maxChars            int
	minChars            int
	similarityThreshold float64
}

// Strategy 基于 Jaccard 相似度的语义分块策略
type Strategy struct {
	opt *options
}

// NewStrategy 创建语义分块策略实例
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

// WithMinChars 设置触发语义切分的最小字符数
func WithMinChars(minChars int) chunk.Option {
	return chunk.WrapChunkImplSpecificOptFn(func(o *options) {
		if minChars <= 0 {
			minChars = defaultMinChars
		}
		o.minChars = minChars
	})
}

// WithSimilarityThreshold 设置语义相似度阈值
func WithSimilarityThreshold(threshold float64) chunk.Option {
	return chunk.WrapChunkImplSpecificOptFn(func(o *options) {
		if threshold <= 0 || threshold >= 1 {
			threshold = defaultSimilarityThreshold
		}
		o.similarityThreshold = threshold
	})
}

// Name 返回策略名称
func (s *Strategy) Name() string {
	return Name
}

// Chunk 执行语义分块
func (s *Strategy) Chunk(ctx context.Context, input *chunk.Input, opts ...chunk.Option) ([]*chunk.Output, error) {
	if input == nil || strutil.Trim(input.Text) == "" {
		return nil, nil
	}

	opt := chunk.GetChunkImplSpecificOptions[options](s.opt, opts...)

	// 文本较短时保持原样，避免过碎
	if utf8.RuneCountInString(input.Text) <= opt.minChars {
		return []*chunk.Output{
			{
				SectionPath:   strutil.Trim(input.SectionPath),
				CanonicalPath: strutil.Trim(input.CanonicalPath),
				ItemIndex:     input.ItemIndex,
				Text:          strutil.Trim(input.Text),
				SourceType:    input.SourceType,
			},
		}, nil
	}

	sentenceList := chunk.SplitSentences(input.Text)
	if len(sentenceList) <= 1 {
		return []*chunk.Output{
			{
				SectionPath:   strutil.Trim(input.SectionPath),
				CanonicalPath: strutil.Trim(input.CanonicalPath),
				ItemIndex:     input.ItemIndex,
				Text:          strutil.Trim(input.Text),
				SourceType:    input.SourceType,
			},
		}, nil
	}

	resultList := make([]*chunk.Output, 0, len(sentenceList))

	currentChunk := make([]rune, 0, 1024)
	currentTokenSet := make(map[string]bool, 64)

	for _, sentence := range sentenceList {
		sentenceTokenSet := chunk.ExtractTokens(sentence)

		currentLen := len(currentChunk)
		sentenceLen := utf8.RuneCountInString(sentence)
		exceedMaxChars := currentLen+sentenceLen > opt.maxChars
		similarity := 1.0
		if len(currentTokenSet) > 0 {
			similarity = chunk.Jaccard(currentTokenSet, sentenceTokenSet)
		}
		semanticBreak := currentLen >= opt.minChars && similarity < opt.similarityThreshold

		// 达到上限或语义断层则切出当前块
		if currentLen > 0 && (exceedMaxChars || semanticBreak) {
			trimmed := strutil.Trim(string(currentChunk))
			if trimmed != "" {
				resultList = append(resultList, &chunk.Output{
					SectionPath:   strutil.Trim(input.SectionPath),
					CanonicalPath: strutil.Trim(input.CanonicalPath),
					ItemIndex:     input.ItemIndex,
					Text:          trimmed,
					SourceType:    input.SourceType,
				})
			}
			currentChunk = currentChunk[:0]
			currentTokenSet = make(map[string]bool, 64)
		}

		currentChunk = append(currentChunk, []rune(sentence)...)
		for token := range sentenceTokenSet {
			currentTokenSet[token] = true
		}
	}

	// 输出最后一块
	if remaining := strutil.Trim(string(currentChunk)); remaining != "" {
		resultList = append(resultList, &chunk.Output{
			SectionPath:   strutil.Trim(input.SectionPath),
			CanonicalPath: strutil.Trim(input.CanonicalPath),
			ItemIndex:     input.ItemIndex,
			Text:          remaining,
			SourceType:    input.SourceType,
		})
	}

	return resultList, nil
}
