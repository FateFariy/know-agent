package chunk

import (
	"context"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/duke-git/lancet/v2/strutil"

	"github.com/swiftbit/know-agent/common/utils"
)

var (
	englishWordPattern = regexp.MustCompile("[A-Za-z0-9]{2,}") // 英文单词正则：至少 2 个字母数字
)

// SemanticStrategy 基于 Jaccard 相似度的语义分块策略
/*
  流程：
  1. 将输入文本拆分为句子
  2. 逐句累计到当前块，直到：
     - 当前块达到最大字符数，或者
     - 当前块与新句子之间的 token Jaccard 相似度低于阈值
  3. 切出当前块
*/
type SemanticStrategy struct {
	opt *semanticOptions
}

type semanticOptions struct {
	maxChars            int
	minChars            int
	similarityThreshold float64
}

func WithSemanticMaxChars(maxChars int) Option {
	return WrapChunkImplSpecificOptFn(func(o *semanticOptions) {
		o.maxChars = maxChars
	})
}

func WithSemanticMinChars(minChars int) Option {
	return WrapChunkImplSpecificOptFn(func(o *semanticOptions) {
		o.minChars = minChars
	})
}

func WithSemanticSimilarityThreshold(threshold float64) Option {
	return WrapChunkImplSpecificOptFn(func(o *semanticOptions) {
		o.similarityThreshold = threshold
	})
}

// NewSemanticStrategy 创建语义分块策略实例
func NewSemanticStrategy(opts ...Option) *SemanticStrategy {
	return &SemanticStrategy{
		opt: GetChunkImplSpecificOptions[semanticOptions](nil, opts...),
	}
}

// Name 返回策略名称
func (s *SemanticStrategy) Name() string {
	return "SEMANTIC"
}

// Chunk 执行语义分块
func (s *SemanticStrategy) Chunk(ctx context.Context, input *Input, opts ...Option) ([]*Output, error) {
	if input == nil || strutil.Trim(input.Text) == "" {
		return nil, nil
	}

	opt := GetChunkImplSpecificOptions[semanticOptions](s.opt, opts...)

	// 文本较短时保持原样，避免过碎
	if utf8.RuneCountInString(input.Text) <= opt.minChars {
		return []*Output{
			{
				SectionPath:   strutil.Trim(input.SectionPath),
				CanonicalPath: strutil.Trim(input.CanonicalPath),
				ItemIndex:     input.ItemIndex,
				Text:          strutil.Trim(input.Text),
				SourceType:    input.SourceType,
			},
		}, nil
	}

	sentenceList := splitSentences(input.Text)
	if len(sentenceList) <= 1 {
		return []*Output{
			{
				SectionPath:   strutil.Trim(input.SectionPath),
				CanonicalPath: strutil.Trim(input.CanonicalPath),
				ItemIndex:     input.ItemIndex,
				Text:          strutil.Trim(input.Text),
				SourceType:    input.SourceType,
			},
		}, nil
	}

	resultList := make([]*Output, 0, len(sentenceList))

	currentChunk := strings.Builder{}
	currentTokenSet := make(map[string]bool, 64)

	for _, sentence := range sentenceList {
		sentenceTokenSet := s.extractTokens(sentence)

		currentLen := utf8.RuneCountInString(currentChunk.String())
		sentenceLen := utf8.RuneCountInString(sentence)
		exceedMaxChars := currentLen+sentenceLen > opt.maxChars
		similarity := utils.Ternary(len(currentTokenSet) == 0, 1.0, s.jaccard(currentTokenSet, sentenceTokenSet))
		semanticBreak := currentLen >= opt.minChars && similarity < opt.similarityThreshold

		// 达到上限或语义断层则切出当前块
		if currentLen > 0 && (exceedMaxChars || semanticBreak) {
			trimmed := strutil.Trim(currentChunk.String())
			if trimmed != "" {
				resultList = append(resultList, &Output{
					SectionPath:   strutil.Trim(input.SectionPath),
					CanonicalPath: strutil.Trim(input.CanonicalPath),
					ItemIndex:     input.ItemIndex,
					Text:          trimmed,
					SourceType:    input.SourceType,
				})
			}
			currentChunk.Reset()
			currentTokenSet = make(map[string]bool, 64)
		}

		// 追加句子并更新词元集合
		currentChunk.WriteString(sentence)
		for token := range sentenceTokenSet {
			currentTokenSet[token] = true
		}
	}

	// 输出最后一块
	if remaining := strutil.Trim(currentChunk.String()); remaining != "" {
		resultList = append(resultList, &Output{
			SectionPath:   strutil.Trim(input.SectionPath),
			CanonicalPath: strutil.Trim(input.CanonicalPath),
			ItemIndex:     input.ItemIndex,
			Text:          remaining,
			SourceType:    input.SourceType,
		})
	}

	return resultList, nil
}

// extractTokens 提取文本的词元集合（英文单词+中文字符）
func (s *SemanticStrategy) extractTokens(text string) map[string]bool {
	tokenSet := make(map[string]bool, 32)
	lower := strings.ToLower(text)
	matches := englishWordPattern.FindAllString(lower, -1)
	for _, m := range matches {
		tokenSet[m] = true
	}
	for _, r := range text {
		if r >= 0x4e00 && r <= 0x9fa5 {
			tokenSet[string(r)] = true
		}
	}
	return tokenSet
}

// jaccard 计算两个词元集的 Jaccard 相似度
func (s *SemanticStrategy) jaccard(left, right map[string]bool) float64 {
	if len(left) == 0 || len(right) == 0 {
		return 0
	}
	unionSize := len(left)
	intersectionSize := 0
	for token := range right {
		if left[token] {
			// 词元已存在，则加入交集
			intersectionSize++
		} else {
			// 词元不存在，则加入并集
			unionSize++
		}
	}
	if unionSize == 0 {
		return 0
	}
	return float64(intersectionSize) / float64(unionSize)
}
