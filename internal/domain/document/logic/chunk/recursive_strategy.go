package chunk

import (
	"context"
	"regexp"
	"strings"
	"unicode/utf8"
)

const (
	// parentBlockMaxChars 父块流水线的默认最大字符数（较大）
	parentBlockMaxChars = 2200

	// parentBlockOverlapChars 父块流水线的默认重叠字符数
	parentBlockOverlapChars = 180
)

var (
	// paragraphSplitRe 段落分隔符：连续换行+若干空白+换行
	paragraphSplitRe = regexp.MustCompile(`\n\s*\n`)

	// lineSplitRe 单行分隔符
	lineSplitRe = regexp.MustCompile(`\n`)

	// sentenceSplitRe 按句末标点切分
	sentenceSplitRe = regexp.MustCompile(`[。！？!?;；.]`)
)

// RecursiveStrategy 递归分块策略
// 按优先级：段落 -> 行 -> 句子 -> 固定窗口，递归地将超长段落继续切分
// 支持在相邻块之间保留一段重叠文本，用于避免切分位置丢失上下文
type RecursiveStrategy struct {
	base baseOptions
}

// NewRecursiveStrategy 创建递归分块策略
func NewRecursiveStrategy(opts ...StrategyOption) *RecursiveStrategy {
	return &RecursiveStrategy{
		base: applyOptions(opts),
	}
}

// Name 返回策略名称
func (r *RecursiveStrategy) Name() string {
	return "RECURSIVE"
}

// Chunk 执行递归分块
// 1. 对每个输入文本进行多级分割并合并为不超过 maxChars 的块
// 2. 最终对整个输出列表进行重叠处理
func (r *RecursiveStrategy) Chunk(_ context.Context, input *Input, pipelineType PipelineType) ([]*Output, error) {
	if input == nil || strings.TrimSpace(input.Text) == "" {
		return []*Output{}, nil
	}

	maxChars := r.resolveMaxChars(pipelineType)
	overlapChars := r.resolveOverlapChars(maxChars, pipelineType)

	// 先按优先级切分为若干原始块
	rawChunks := r.recursiveSplit(strings.TrimSpace(input.Text), maxChars, overlapChars)

	result := make([]*Output, 0, len(rawChunks))
	for _, text := range rawChunks {
		trimmed := strings.TrimSpace(text)
		if trimmed == "" {
			continue
		}
		result = append(result, &Output{
			SectionPath:   strings.TrimSpace(input.SectionPath),
			CanonicalPath: strings.TrimSpace(input.CanonicalPath),
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
		return []string{}
	}
	if utf8.RuneCountInString(text) <= maxChars {
		return []string{text}
	}

	// 1) 按段落切分
	segmentList := splitByRegex(text, paragraphSplitRe)
	if len(segmentList) > 1 {
		return r.mergeAndSplit(segmentList, maxChars, overlapChars)
	}

	// 2) 按换行切分
	segmentList = splitByRegex(text, lineSplitRe)
	if len(segmentList) > 1 {
		return r.mergeAndSplit(segmentList, maxChars, overlapChars)
	}

	// 3) 按句子切分
	segmentList = splitSentences(text)
	if len(segmentList) > 1 {
		return r.mergeAndSplit(segmentList, maxChars, overlapChars)
	}

	// 4) 最后兜底：固定窗口切分
	return fixedWindowSplit(text, maxChars, overlapChars)
}

// mergeAndSplit 将片段依次累加，超出 maxChars 时刷出一个块，然后继续
// 对单个超长片段将再次进入 recursiveSplit 做进一步切分
func (r *RecursiveStrategy) mergeAndSplit(segmentList []string, maxChars, overlapChars int) []string {
	rawResultList := make([]string, 0, len(segmentList))
	current := strings.Builder{}

	for _, segment := range segmentList {
		trimmed := strings.TrimSpace(segment)
		if trimmed == "" {
			continue
		}

		if utf8.RuneCountInString(trimmed) > maxChars {
			// 当前片段过长：先刷出已累积的，然后递归该片段
			if current.Len() > 0 {
				rawResultList = append(rawResultList, strings.TrimSpace(current.String()))
				current.Reset()
			}
			rawResultList = append(rawResultList, r.recursiveSplit(trimmed, maxChars, overlapChars)...)
			continue
		}

		// 先刷出，再开启新块
		if utf8.RuneCountInString(current.String())+utf8.RuneCountInString(trimmed)+1 > maxChars {
			rawResultList = append(rawResultList, strings.TrimSpace(current.String()))
			current.Reset()
		}

		current.WriteString(trimmed)
		current.WriteByte('\n')
	}

	if current.Len() > 0 {
		rawResultList = append(rawResultList, strings.TrimSpace(current.String()))
	}

	return applyOverlap(rawResultList, maxChars, overlapChars)
}

// resolveMaxChars 根据流水线类型返回块大小
func (r *RecursiveStrategy) resolveMaxChars(pipelineType PipelineType) int {
	if pipelineType == PipelineTypeParent {
		return parentBlockMaxChars
	}
	return r.base.recursiveMaxChars
}

// resolveOverlapChars 根据流水线类型返回重叠字符数
func (r *RecursiveStrategy) resolveOverlapChars(maxChars int, pipelineType PipelineType) int {
	if pipelineType == PipelineTypeParent {
		return min(parentBlockOverlapChars, max(0, maxChars-1))
	}
	if r.base.recursiveOverlapChars <= 0 {
		return 0
	}
	return min(r.base.recursiveOverlapChars, max(0, maxChars-1))
}

// splitByRegex 按正则切分文本，过滤空白片段
func splitByRegex(text string, re *regexp.Regexp) []string {
	raw := re.Split(text, -1)
	result := make([]string, 0, len(raw))
	for _, part := range raw {
		trimmed := strings.TrimSpace(part)
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
		trimmed := strings.TrimSpace(text)
		if trimmed == "" {
			return []string{}
		}
		return []string{trimmed}
	}
	result := make([]string, 0, len(indices)+1)
	prev := 0
	for _, idxPair := range indices {
		end := idxPair[1]
		segment := strings.TrimSpace(text[prev:end])
		if segment != "" {
			result = append(result, segment)
		}
		prev = end
	}
	if prev < len(text) {
		tail := strings.TrimSpace(text[prev:])
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
		currentTrimmed := strings.TrimSpace(current)
		if currentTrimmed == "" {
			continue
		}
		if index == 0 {
			overlappedChunkList = append(overlappedChunkList, currentTrimmed)
			continue
		}
		previous := strings.TrimSpace(rawChunkList[index-1])
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
	previous = strings.TrimSpace(previous)
	current = strings.TrimSpace(current)
	if previous == "" || current == "" {
		return ""
	}
	allowed := min(overlapChars, max(0, maxChars-utf8.RuneCountInString(current)-1))
	if allowed <= 0 {
		return ""
	}
	prevRunes := []rune(previous)
	startIdx := len(prevRunes) - allowed
	if startIdx < 0 {
		startIdx = 0
	}
	return strings.TrimSpace(string(prevRunes[startIdx:]))
}

// fixedWindowSplit 固定窗口切分超长文本
func fixedWindowSplit(text string, maxChars, overlapChars int) []string {
	runes := []rune(strings.TrimSpace(text))
	total := len(runes)
	if total == 0 {
		return []string{}
	}
	if total <= maxChars {
		return []string{string(runes)}
	}

	result := make([]string, 0, total/maxChars+1)
	step := max(1, maxChars-overlapChars)
	start := 0
	for start < total {
		end := start + maxChars
		if end > total {
			end = total
		}
		result = append(result, strings.TrimSpace(string(runes[start:end])))
		if end >= total {
			break
		}
		start += step
	}
	return result
}
