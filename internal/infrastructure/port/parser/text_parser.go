package parser

import (
	"context"
	"regexp"
	"strings"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/document/logic/process/parse"
	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
)

const Text = "native_text"

var (
	// 匹配连续空行：换行 + 任意空白 + 至少一个换行
	emptyLinesRegex = regexp.MustCompile(`\n\s*\n+`)

	// 匹配文档序号开头：中文序号、第X章节条、多级数字编号
	headingPrefixRegex = regexp.MustCompile(`^([一二三四五六七八九十]+、|第[一二三四五六七八九十0-9]+[章节条]|[0-9]+(\.[0-9]+)*[、.])`)

	// 匹配标题开头：# 开头，1-6 个 # 号
	titleRegex = regexp.MustCompile(`^#{1,6}\s+`)
)

type TextParser struct {
}

var _ parse.Parser = (*TextParser)(nil)

func (p *TextParser) Name() string {
	return Text
}

func (p *TextParser) Parse(_ context.Context, sourceText []byte) (entity.DocumentBlocks, error) {
	blocks := make(entity.DocumentBlocks, 0)
	blockNo := 1
	for _, paragraph := range splitParagraphs(string(sourceText)) {
		blockType := classifyTextBlock(paragraph)
		// 去除标题标记（如 # 开头）
		if blockType == "TITLE" {
			// 移除可能的前导 # 标记（最多6个）
			paragraph = titleRegex.ReplaceAllString(paragraph, "")
			paragraph = strings.TrimSpace(paragraph)
		}
		blocks = append(blocks, &entity.DocumentBlock{
			BlockNo:   blockNo,
			BlockType: blockType,
			Text:      paragraph,
			Metadata:  map[string]any{"parser": Text},
		})
		blockNo++
	}
	return blocks, nil
}

// splitParagraphs 将文本分割为段落列表
func splitParagraphs(text string) []string {
	cleaned := cleanupText(text)
	if cleaned == "" {
		return []string{}
	}
	// 按两个以上连续换行符（含空格）分割
	parts := emptyLinesRegex.Split(cleaned, -1)
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// cleanupText 清理文本中的常见空白字符
func cleanupText(text string) string {
	if text == "" {
		return ""
	}
	replacer := strings.NewReplacer("\r\n", "\n", "\r", "\n", "\x00", " ", "\t", " ")
	text = replacer.Replace(text)
	return strings.TrimSpace(text)
}

// classifyTextBlock 判断文本块类型
func classifyTextBlock(text string) string {
	stripped := strings.TrimSpace(text)
	if stripped == "" {
		return "Text"
	}
	if utils.Len(stripped) <= 80 {
		// 正则匹配中文序号、章节条、数字序号
		if headingPrefixRegex.MatchString(stripped) {
			return "TITLE"
		}
		// 以章、节、：、: 结尾
		if strings.HasSuffix(stripped, "章") ||
			strings.HasSuffix(stripped, "节") ||
			strings.HasSuffix(stripped, "：") ||
			strings.HasSuffix(stripped, ":") {
			return "TITLE"
		}
	}
	return "Text"
}
