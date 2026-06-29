package semantic

import (
	"context"
	"regexp"
	"strings"

	"github.com/duke-git/lancet/v2/strutil"
)

var (
	englishWordPattern = regexp.MustCompile("[A-Za-z0-9]{2,}") // 至少 2 个字母数字
)

// JaccardSimilarity 基于 Jaccard 计算相似度，以“英文单词 + 中文字符”为词元，计算两边词元集合的交集并集比值
type JaccardSimilarity struct{}

var _ Similarity = (*JaccardSimilarity)(nil)

// Calculate 两段文本的 Jaccard 相似度，文本为空时返回 0
func (s *JaccardSimilarity) Calculate(_ context.Context, text1, text2 string) (float64, error) {
	if strutil.Trim(text1) == "" || strutil.Trim(text2) == "" {
		return 0, nil
	}
	left, right := s.extractTokens(text1), s.extractTokens(text2)
	return s.jaccard(left, right), nil
}

// extractTokens 提取文本的词元集合（英文单词 + 中文字符），不区分英文大小写
func (s *JaccardSimilarity) extractTokens(text string) map[string]bool {
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

// jaccard 计算两个词元集合的 Jaccard 相似度（交集/并集）
func (s *JaccardSimilarity) jaccard(left, right map[string]bool) float64 {
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
