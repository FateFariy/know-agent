package parser

import (
	"context"
	"strings"

	"github.com/swiftbit/know-agent/internal/domain/document/logic/process/parse"
	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
)

const Text = "native_text"

type TextParser struct {
}

var _ parse.Parser = (*TextParser)(nil)

func (p *TextParser) Name() string {
	return Text
}

func (p *TextParser) Parse(_ context.Context, sourceText []byte) (entity.DocumentBlocks, error) {
	text := decodeText(sourceText)
	blocks := make(entity.DocumentBlocks, 0)
	blockNo := 1
	for _, paragraph := range splitParagraphs(text) {
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
