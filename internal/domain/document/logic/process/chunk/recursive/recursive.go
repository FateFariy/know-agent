package recursive

import (
	"context"
	"strings"

	"github.com/duke-git/lancet/v2/strutil"

	"github.com/swiftbit/know-agent/common"
	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/document/logic/process/chunk"
	"github.com/swiftbit/know-agent/internal/domain/document/model/enum"
)

// Chunker 递归分块策略, 按优先级：段落 -> 行 -> 句子 -> 固定窗口，递归地将超长段落继续切分, 支持在相邻块之间保留一段重叠文本
type Chunker struct {
	opt *options
}

// NewChunker 创建递归分块器
func NewChunker(opts ...common.Option) *Chunker {
	return &Chunker{
		opt: common.GetImplSpecificOptions(&options{
			maxChars:     defaultMaxChars,
			overlapChars: defaultOverlapChars,
		}, opts...),
	}
}

// Name 返回策略名称
func (s *Chunker) Name() string {
	return enum.StrategyTypeName(enum.StrategyTypeRecursive)
}

// Chunk 执行递归分块
func (s *Chunker) Chunk(_ context.Context, input string, opts ...common.Option) ([]string, error) {
	text := strings.TrimSpace(input)
	if text == "" {
		return nil, nil
	}

	// 允许通过 opts 覆盖原始配置
	opt := common.GetImplSpecificOptions(s.opt, opts...)

	// 先按优先级切分为若干原始块
	rawChunks := s.split(text, opt.maxChars, opt.overlapChars)
	i := 0
	for _, chunkText := range rawChunks {
		trimmed := strutil.Trim(chunkText)
		if trimmed != "" {
			rawChunks[i] = trimmed
			i++
		}
	}
	return rawChunks[:i], nil
}

// split 递归切分主入口
func (s *Chunker) split(text string, maxChars, overlapChars int) []string {
	if text == "" {
		return nil
	}
	if utils.Len(text) <= maxChars {
		return []string{text}
	}

	overlapChars = min(overlapChars, max(0, maxChars-1))

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
func (s *Chunker) mergeAndSplit(segmentList []string, maxChars, overlapChars int) []string {
	rawResultList := make([]string, 0, len(segmentList))
	current := strings.Builder{}

	for _, segment := range segmentList {
		trimmed := strutil.Trim(segment)
		if trimmed != "" {
			if utils.Len(trimmed) > maxChars {
				// 当前片段过长：先刷出已累积的，然后递归该片段
				if current.Len() > 0 {
					rawResultList = append(rawResultList, strutil.Trim(current.String()))
					current.Reset()
				}
				rawResultList = append(rawResultList, s.split(trimmed, maxChars, overlapChars)...)
				continue
			}

			// 先刷出，再开启新块
			if utils.Len(current.String())+utils.Len(trimmed)+1 > maxChars {
				rawResultList = append(rawResultList, strutil.Trim(current.String()))
				current.Reset()
			}
			current.WriteString(trimmed)
			current.WriteRune('\n')
		}
	}

	if current.Len() > 0 {
		rawResultList = append(rawResultList, strutil.Trim(current.String()))
	}

	// 为块列表增加重叠前缀
	return s.applyOverlap(rawResultList, maxChars, overlapChars)
}

// applyOverlap 为块列表增加重叠前缀
func (s *Chunker) applyOverlap(rawChunkList []string, maxChars, overlapChars int) []string {
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
func (s *Chunker) buildOverlapPrefix(previous, current string, maxChars, overlapChars int) string {
	previous = strutil.Trim(previous)
	current = strutil.Trim(current)
	if previous == "" || current == "" {
		return ""
	}

	// 计算允许的重叠字符数，重叠字符数不能超过 maxChars，也不能超过 previous 的长度
	allowed := min(overlapChars, max(0, maxChars-utils.Len(current)-1))
	if allowed <= 0 {
		return ""

	}
	// 取 previous 尾部 allowed 个字符作为重叠前缀
	prevRunes := []rune(previous)
	startIdx := max(len(prevRunes)-allowed, 0)

	return strutil.Trim(string(prevRunes[startIdx:]))
}

// fixedWindowSplit 固定窗口切分超长文本
func (s *Chunker) fixedWindowSplit(text string, maxChars, overlapChars int) []string {
	trim := strutil.Trim(text)
	total := utils.Len(trim)
	if total == 0 {
		return nil
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
