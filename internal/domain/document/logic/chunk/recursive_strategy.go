package chunk

import (
	"context"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/duke-git/lancet/v2/strutil"
)

var (
	paragraphSplitRe = regexp.MustCompile(`\n\s*\n`)    // 段落分隔符：连续换行+若干空白+换行
	lineSplitRe      = regexp.MustCompile(`\n`)         // 单行分隔符
	sentenceSplitRe  = regexp.MustCompile(`[。！？!?;；.]`) // 句末标点
)

// RecursiveStrategy 递归分块策略
// 按优先级：段落 -> 行 -> 句子 -> 固定窗口，递归地将超长段落继续切分, 支持在相邻块之间保留一段重叠文本，用于避免切分位置丢失上下文
type RecursiveStrategy struct {
	opt *recursiveOptions
}

type recursiveOptions struct {
	maxChars     int
	overlapChars int
}

// NewRecursiveStrategy 创建递归分块策略
func NewRecursiveStrategy(opts ...Option) *RecursiveStrategy {
	return &RecursiveStrategy{
		opt: GetChunkImplSpecificOptions[recursiveOptions](nil, opts...),
	}
}

// Name 返回策略名称
func (r *RecursiveStrategy) Name() string {
	return "RECURSIVE"
}

// Chunk 执行递归分块
func (r *RecursiveStrategy) Chunk(ctx context.Context, input *Input, opts ...Option) ([]*Output, error) {
	if input == nil || strutil.Trim(input.Text) == "" {
		return nil, nil
	}

	opt := GetChunkImplSpecificOptions[recursiveOptions](r.opt, opts...)

	// 先按优先级切分为若干原始块
	rawChunks := r.recursiveSplit(strutil.Trim(input.Text), opt.maxChars, opt.overlapChars)

	result := make([]*Output, 0, len(rawChunks))
	for _, text := range rawChunks {
		trimmed := strutil.Trim(text)
		if trimmed == "" {
			continue
		}
		result = append(result, &Output{
			SectionPath:   strutil.Trim(input.SectionPath),
			CanonicalPath: strutil.Trim(input.CanonicalPath),
			ItemIndex:     input.ItemIndex,
			Text:          trimmed,
			SourceType:    input.SourceType,
		})
	}
	return result, nil
}

// recursiveSplit 递归切分主入口
func (r *RecursiveStrategy) recursiveSplit(text string, maxChars, overlapChars int) []string {
	if text == "" {
		return nil
	}

	// 如果文本长度小于等于最大字符数，直接返回
	if utf8.RuneCountInString(text) <= maxChars {
		return []string{text}
	}

	// 按段落切分
	segmentList := splitByRegex(text, paragraphSplitRe)
	if len(segmentList) > 1 {
		return r.mergeAndSplit(segmentList, maxChars, overlapChars)
	}

	// 按换行切分
	segmentList = splitByRegex(text, lineSplitRe)
	if len(segmentList) > 1 {
		return r.mergeAndSplit(segmentList, maxChars, overlapChars)
	}

	// 按句子切分
	segmentList = splitSentences(text)
	if len(segmentList) > 1 {
		return r.mergeAndSplit(segmentList, maxChars, overlapChars)
	}

	// 最后兜底：固定窗口切分
	return fixedWindowSplit(text, maxChars, overlapChars)
}

// mergeAndSplit 将片段依次累加，超出 maxChars 时刷出一个块，然后继续，对单个超长片段将再次进入 recursiveSplit 做进一步切分
func (r *RecursiveStrategy) mergeAndSplit(segmentList []string, maxChars, overlapChars int) []string {
	rawResultList := make([]string, 0, len(segmentList))
	current := strings.Builder{}

	for _, segment := range segmentList {
		trimmed := strutil.Trim(segment)
		if trimmed == "" {
			continue
		}

		if utf8.RuneCountInString(trimmed) > maxChars {
			// 当前片段过长：先刷出已累积的，然后递归该片段
			if current.Len() > 0 {
				rawResultList = append(rawResultList, strutil.Trim(current.String()))
				current.Reset()
			}
			rawResultList = append(rawResultList, r.recursiveSplit(trimmed, maxChars, overlapChars)...)
			continue
		}

		// 先刷出，再开启新块
		if utf8.RuneCountInString(current.String())+utf8.RuneCountInString(trimmed)+1 > maxChars {
			rawResultList = append(rawResultList, strutil.Trim(current.String()))
			current.Reset()
		}

		current.WriteString(trimmed)
		current.WriteByte('\n')
	}

	if current.Len() > 0 {
		rawResultList = append(rawResultList, strutil.Trim(current.String()))
	}

	return applyOverlap(rawResultList, maxChars, overlapChars)
}

// splitByRegex 按正则切分文本，过滤空白片段
func splitByRegex(text string, re *regexp.Regexp) []string {
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

// splitSentences 按句末标点切分句子（标点保留在句子结尾）
func splitSentences(text string) []string {
	indices := sentenceSplitRe.FindAllStringIndex(text, -1)
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

// applyOverlap 为块列表增加重叠前缀
func applyOverlap(rawChunkList []string, maxChars, overlapChars int) []string {
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
		previous := strutil.Trim(rawChunkList[index-1])
		overlapPrefix := buildOverlapPrefix(previous, currentTrimmed, maxChars, overlapChars)
		if overlapPrefix != "" {
			overlappedChunkList = append(overlappedChunkList, overlapPrefix+"\n"+currentTrimmed)
		} else {
			overlappedChunkList = append(overlappedChunkList, currentTrimmed)
		}
	}
	return overlappedChunkList
}

// buildOverlapPrefix 取 previous 尾部作为重叠前缀，受 maxChars 约束
func buildOverlapPrefix(previous, current string, maxChars, overlapChars int) string {
	previous = strutil.Trim(previous)
	current = strutil.Trim(current)
	if previous == "" || current == "" {
		return ""
	}
	allowed := min(overlapChars, max(0, maxChars-utf8.RuneCountInString(current)-1))
	if allowed <= 0 {
		return ""
	}
	prevRunes := []rune(previous)
	startIdx := max(len(prevRunes)-allowed, 0)
	return strutil.Trim(string(prevRunes[startIdx:]))
}

// fixedWindowSplit 固定窗口切分超长文本
func fixedWindowSplit(text string, maxChars, overlapChars int) []string {
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
