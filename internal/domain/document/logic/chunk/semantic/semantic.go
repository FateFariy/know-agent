package semantic

import (
	"context"
	"strings"

	"github.com/duke-git/lancet/v2/strutil"

	"github.com/swiftbit/know-agent/common/utils"
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
	similarity          Similarity
}

// Strategy 语义分块策略
type Strategy struct {
	opt *options
}

// NewStrategy 创建语义分块策略实例，默认使用 JaccardSimilarity 实现相似度计算
func NewStrategy(opts ...chunk.Option) *Strategy {
	return &Strategy{
		opt: chunk.GetChunkImplSpecificOptions(&options{
			maxChars:            defaultMaxChars,
			minChars:            defaultMinChars,
			similarityThreshold: defaultSimilarityThreshold,
			similarity:          &JaccardSimilarity{},
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

// WithSimilarity 注入自定义的相似度计算实现，
func WithSimilarity(sim Similarity) chunk.Option {
	return chunk.WrapChunkImplSpecificOptFn(func(o *options) {
		if sim == nil {
			return
		}
		o.similarity = sim
	})
}

// Name 返回策略名称
func (s *Strategy) Name() string {
	return Name
}

// Chunk 执行语义分块
func (s *Strategy) Chunk(ctx context.Context, input *chunk.TextBlock, opts ...chunk.Option) ([]*chunk.TextBlock, error) {
	if input == nil || strutil.Trim(input.Text) == "" {
		return nil, nil
	}

	opt := chunk.GetChunkImplSpecificOptions(s.opt, opts...)

	// 文本较短时保持原样，避免过碎
	if utils.Len(input.Text) <= opt.minChars {
		return []*chunk.TextBlock{input}, nil
	}

	// 按句子分块
	sentenceList := chunk.SplitSentences(input.Text)
	if len(sentenceList) <= 1 {
		return []*chunk.TextBlock{input}, nil
	}

	resultList := make([]*chunk.TextBlock, 0, len(sentenceList))
	currentText := strings.Builder{}
	for _, sentence := range sentenceList {
		currentLen := utils.Len(currentText.String())
		sentenceLen := utils.Len(sentence)
		exceedMaxChars := currentLen+sentenceLen > opt.maxChars
		// 计算语义相似度
		similarity := 1.0
		if currentLen > 0 {
			simValue, err := opt.similarity.Calculate(ctx, currentText.String(), sentence)
			if err != nil {
				return nil, err
			}
			similarity = simValue
		}
		semanticBreak := currentLen >= opt.minChars && similarity < opt.similarityThreshold

		// 达到上限或语义断层则切出当前块
		if currentLen > 0 && (exceedMaxChars || semanticBreak) {
			trimmed := strutil.Trim(currentText.String())
			if trimmed != "" {
				resultList = append(resultList, input.CloneWithText(trimmed))
			}
			currentText.Reset()
		}

		currentText.WriteString(sentence)
	}

	// 输出最后一块
	if remaining := strutil.Trim(currentText.String()); remaining != "" {
		resultList = append(resultList, input.CloneWithText(remaining))
	}

	return resultList, nil
}
