package chunk

import (
	"regexp"

	"github.com/duke-git/lancet/v2/strutil"
)

var (
	ParagraphSplitRe = regexp.MustCompile(`\n\s*\n`)    // 段落分隔符：连续换行+若干空白+换行
	LineSplitRe      = regexp.MustCompile(`\n`)         // 单行分隔符
	SentenceSplitRe  = regexp.MustCompile(`[。！？!?;；.]`) // 句末标点
)

// SplitByRegex 按正则切分文本，过滤空白片段
func SplitByRegex(text string, re *regexp.Regexp) []string {
	raw := re.Split(text, -1)
	result := make([]string, 0, len(raw))
	for _, part := range raw {
		trimmed := strutil.Trim(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// SplitSentences 按句末标点切分句子（标点保留在句子结尾）
func SplitSentences(text string) []string {
	indices := SentenceSplitRe.FindAllStringIndex(text, -1)
	if len(indices) == 0 {
		trimmed := strutil.Trim(text)
		if trimmed == "" {
			return nil
		}
		return []string{trimmed}
	}
	result := make([]string, 0, len(indices)+1)
	prev := 0
	for _, idxPair := range indices {
		end := idxPair[1]
		segment := strutil.Trim(text[prev:end])
		if segment != "" {
			result = append(result, segment)
		}
		prev = end
	}
	if prev < len(text) {
		tail := strutil.Trim(text[prev:])
		if tail != "" {
			result = append(result, tail)
		}
	}
	return result
}
