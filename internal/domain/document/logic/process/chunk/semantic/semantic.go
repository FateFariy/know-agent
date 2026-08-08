package semantic

import (
	"context"
	"strings"

	"github.com/duke-git/lancet/v2/strutil"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/document/logic/process/chunk"
	"github.com/swiftbit/know-agent/internal/domain/document/logic/process/chunk/semantic/similarity"
	"github.com/swiftbit/know-agent/internal/domain/document/model/vo"
)

const (
	Name = "SEMANTIC" // 名称
)

// Chunker 语义分块策略
type Chunker struct {
	opt *options
}

// NewChunker 创建语义分块策略实例，默认使用 JaccardSimilarity 实现相似度计算
func NewChunker(opts ...chunk.Option) *Chunker {
	return &Chunker{
		opt: chunk.GetSpecificOptions(&options{
			maxChars:            defaultMaxChars,
			minChars:            defaultMinChars,
			similarityThreshold: defaultSimilarityThreshold,
			calculator:          &similarity.JaccardSimilarity{},
		}, opts...),
	}
}

// Name 返回策略名称
func (s *Chunker) Name() string {
	return Name
}

// Chunk 执行语义分块
func (s *Chunker) Chunk(ctx context.Context, input *chunk.Input, opts ...chunk.Option) ([]*vo.ChunkCandidate, error) {
	if input == nil || strutil.Trim(input.Text) == "" {
		return nil, nil
	}

	opt := chunk.GetSpecificOptions(s.opt, opts...)

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
		simValue := 1.0
		var err error
		if currentLen > 0 {
			simValue, err = opt.calculator.Calculate(ctx, currentText.String(), sentence)
			if err != nil {
				return nil, err
			}
		}
		semanticBreak := currentLen >= opt.minChars && simValue < opt.similarityThreshold

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
