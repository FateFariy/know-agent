package chunk

import (
	"regexp"
	"strings"

	"github.com/duke-git/lancet/v2/strutil"
)

var (
	paragraphSplitRe = regexp.MustCompile(`\n\s*\n`)    // 段落分隔符：连续换行+若干空白+换行
	lineSplitRe      = regexp.MustCompile(`\n`)         // 单行分隔符
	sentenceSplitRe  = regexp.MustCompile(`[。！？!?;；.]`) // 句末标点
)

// ParagraphSplitRe 返回段落分隔符正则
func ParagraphSplitRe() *regexp.Regexp { return paragraphSplitRe }

// LineSplitRe 返回换行分隔符正则
func LineSplitRe() *regexp.Regexp { return lineSplitRe }

// SentenceSplitRe 返回句末标点分隔符正则
func SentenceSplitRe() *regexp.Regexp { return sentenceSplitRe }

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

// ComposeSectionPath 拼接基础路径与当前层级路径，用 " > " 分隔
func ComposeSectionPath(base, current string) string {
	baseTrimmed := strutil.Trim(base)
	currentTrimmed := strutil.Trim(current)
	if baseTrimmed == "" {
		return currentTrimmed
	}
	if currentTrimmed == "" {
		return baseTrimmed
	}
	return baseTrimmed + " > " + currentTrimmed
}

// ParseStringJSONArrayFrom 从文本中抽取 JSON 数组，并解析其字符串元素
func ParseStringJSONArrayFrom(content string) []string {
	startIdx := strings.Index(content, "[")
	endIdx := strings.LastIndex(content, "]")
	if startIdx < 0 || endIdx <= startIdx {
		return nil
	}
	inner := content[startIdx : endIdx+1]
	return parseStringJSONArray(inner)
}

// parseStringJSONArray 简易解析 JSON 字符串数组（仅处理双引号字符串元素）
func parseStringJSONArray(content string) []string {
	trimmed := strings.TrimSpace(content)
	if !strings.HasPrefix(trimmed, "[") || !strings.HasSuffix(trimmed, "]") {
		return nil
	}
	inner := strings.TrimSpace(trimmed[1 : len(trimmed)-1])
	if inner == "" {
		return nil
	}

	result := make([]string, 0)
	runes := []rune(inner)
	n := len(runes)
	i := 0
	for i < n {
		// 跳过空白和逗号
		for i < n && (runes[i] == ',' || runes[i] == ' ' || runes[i] == '\t' || runes[i] == '\r' || runes[i] == '\n') {
			i++
		}
		if i >= n {
			break
		}
		if runes[i] != '"' {
			// 跳过非字符串元素直到下一个逗号
			for i < n && runes[i] != ',' {
				i++
			}
			continue
		}
		i++ // 跳过 "
		sb := strings.Builder{}
		for i < n {
			r := runes[i]
			if r == '\\' && i+1 < n {
				next := runes[i+1]
				switch next {
				case '"':
					sb.WriteByte('"')
				case '\\':
					sb.WriteByte('\\')
				case '/':
					sb.WriteByte('/')
				case 'n':
					sb.WriteByte('\n')
				case 't':
					sb.WriteByte('\t')
				case 'r':
					sb.WriteByte('\r')
				default:
					sb.WriteRune(r)
					sb.WriteRune(next)
				}
				i += 2
				continue
			}
			if r == '"' {
				i++
				break
			}
			sb.WriteRune(r)
			i++
		}
		result = append(result, sb.String())
	}
	return result
}
