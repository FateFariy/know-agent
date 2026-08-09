package shared

import (
	"strings"

	"github.com/swiftbit/know-agent/common/utils"
)

// RichContentSeed 表示构建带权重富文本内容所需的输入上下文
type RichContentSeed struct {
	text                  string
	sectionPath           string
	title                 string
	chunkType             string
	keywords              []string
	questions             []string
	parserWeightedContent string
}

func NewRichContentSeed(text, sectionPath, title, chunkType, parserWeightedContent string, keywords, questions []string) *RichContentSeed {
	return &RichContentSeed{
		text:                  strings.TrimSpace(text),
		sectionPath:           strings.TrimSpace(sectionPath),
		title:                 strings.TrimSpace(title),
		chunkType:             strings.TrimSpace(chunkType),
		keywords:              utils.FilterBlank(keywords),
		questions:             utils.FilterBlank(questions),
		parserWeightedContent: strings.TrimSpace(parserWeightedContent),
	}
}

// Build 组装带权重的富文本内容。
func (s *RichContentSeed) Build() string {
	var parts []string

	if s.title != "" {
		parts = append(parts, "[TITLE]\n"+s.title)
	}

	if s.sectionPath != "" {
		parts = append(parts, "[SECTION]\n"+s.sectionPath)
	}

	if s.chunkType != "" {
		parts = append(parts, "[CHUNK_TYPE]\n"+s.chunkType)
	}

	if len(s.keywords) > 0 {
		parts = append(parts, "[KEYWORDS]\n"+strings.Join(s.keywords, ";"))
	}

	if len(s.questions) > 0 {
		parts = append(parts, "[QUESTIONS]\n"+strings.Join(s.questions, ";"))
	}

	body := s.parserWeightedContent
	if body == "" {
		body = s.text
	}
	if body != "" {
		parts = append(parts, "[CONTENT]\n"+body)
	}

	return strings.TrimSpace(strings.Join(parts, "\n\n"))
}
