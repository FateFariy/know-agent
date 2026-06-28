package chunk

import (
	"context"
	"regexp"
	"strings"
	"unicode/utf8"
)

// SemanticStrategy 基于 Jaccard 相似度的语义分块策略
// 流程：
//  1. 将输入文本拆分为句子
//  2. 逐句累计到当前块，直到：
//     - 当前块达到最大字符数，或者
//     - 当前块与新句子之间的 token Jaccard 相似度低于阈值
//  3. 切出当前块
type SemanticStrategy struct {
	base baseOptions
}

const (
	// parentSemanticMaxChars 父块流水线的默认最大字符数
	parentSemanticMaxChars = 1600

	// parentSemanticMinChars 父块流水线的默认最小字符数
	parentSemanticMinChars = 480
)

// englishWordPattern 英文单词正则：至少 2 个字母数字
var englishWordPattern = regexp.MustCompile("[A-Za-z0-9]{2,}")

// NewSemanticStrategy 创建语义分块策略实例
func NewSemanticStrategy(opts ...StrategyOption) *SemanticStrategy {
	return &SemanticStrategy{
		base: applyOptions(opts),
	}
}

// Name 返回策略名称
func (s *SemanticStrategy) Name() string {
	return "SEMANTIC"
}

// Chunk 执行语义分块
func (s *SemanticStrategy) Chunk(_ context.Context, input *Input, pipelineType PipelineType) ([]*Output, error) {
	if input == nil || strings.TrimSpace(input.Text) == "" {
		return []*Output{}, nil
	}

	semanticMinChars := s.resolveMinChars(pipelineType)
	// 文本较短时保持原样，避免过碎
	if utf8.RuneCountInString(input.Text) <= semanticMinChars {
		return []*Output{
			{
				SectionPath:   strings.TrimSpace(input.SectionPath),
				CanonicalPath: strings.TrimSpace(input.CanonicalPath),
				ItemIndex:     input.ItemIndex,
				Text:          strings.TrimSpace(input.Text),
				SourceType:    input.SourceType,
			},
		}, nil
	}

	sentenceList := splitSentences(input.Text)
	if len(sentenceList) <= 1 {
		return []*Output{
			{
				SectionPath:   strings.TrimSpace(input.SectionPath),
				CanonicalPath: strings.TrimSpace(input.CanonicalPath),
				ItemIndex:     input.ItemIndex,
				Text:          strings.TrimSpace(input.Text),
				SourceType:    input.SourceType,
			},
		}, nil
	}

	resultList := make([]*Output, 0, len(sentenceList))
	semanticMaxChars := s.resolveMaxChars(pipelineType)

	currentChunk := strings.Builder{}
	currentTokenSet := make(map[string]bool, 64)

	for _, sentence := range sentenceList {
		sentenceTokenSet := extractTokens(sentence)

		currentLen := utf8.RuneCountInString(currentChunk.String())
		sentenceLen := utf8.RuneCountInString(sentence)
		exceedMaxChars := currentLen+sentenceLen > semanticMaxChars
		var similarity float64
		if len(currentTokenSet) == 0 {
			similarity = 1.0
		} else {
			similarity = jaccard(currentTokenSet, sentenceTokenSet)
		}
		semanticBreak := currentLen >= semanticMinChars && similarity < s.base.semanticSimilarityThreshold

		// 达到上限或语义断层则切出当前块
		if currentLen > 0 && (exceedMaxChars || semanticBreak) {
			trimmed := strings.TrimSpace(currentChunk.String())
			if trimmed != "" {
				resultList = append(resultList, &Output{
					SectionPath:   strings.TrimSpace(input.SectionPath),
					CanonicalPath: strings.TrimSpace(input.CanonicalPath),
					ItemIndex:     input.ItemIndex,
					Text:          trimmed,
					SourceType:    input.SourceType,
				})
			}
			currentChunk.Reset()
			currentTokenSet = make(map[string]bool, 64)
		}

		currentChunk.WriteString(sentence)
		for token := range sentenceTokenSet {
			currentTokenSet[token] = true
		}
	}

	// 输出最后一块
	if remaining := strings.TrimSpace(currentChunk.String()); remaining != "" {
		resultList = append(resultList, &Output{
			SectionPath:   strings.TrimSpace(input.SectionPath),
			CanonicalPath: strings.TrimSpace(input.CanonicalPath),
			ItemIndex:     input.ItemIndex,
			Text:          remaining,
			SourceType:    input.SourceType,
		})
	}

	return resultList, nil
}

// resolveMaxChars 根据流水线解析语义块最大字符数
func (s *SemanticStrategy) resolveMaxChars(pipelineType PipelineType) int {
	if pipelineType == PipelineTypeParent {
		return max(parentSemanticMaxChars, s.base.semanticMaxChars)
	}
	return s.base.semanticMaxChars
}

// resolveMinChars 根据流水线解析语义块最小字符数
func (s *SemanticStrategy) resolveMinChars(pipelineType PipelineType) int {
	if pipelineType == PipelineTypeParent {
		return max(parentSemanticMinChars, s.base.semanticMinChars)
	}
	return s.base.semanticMinChars
}

// extractTokens 提取文本的词元集合（英文单词+中文字符）
func extractTokens(text string) map[string]bool {
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
func jaccard(left, right map[string]bool) float64 {
	if len(left) == 0 || len(right) == 0 {
		return 0
	}
	unionSize := len(left)
	intersectionSize := 0
	for token := range right {
		if left[token] {
			intersectionSize++
		} else {
			unionSize++
		}
	}
	if unionSize == 0 {
		return 0
	}
	return float64(intersectionSize) / float64(unionSize)
}
