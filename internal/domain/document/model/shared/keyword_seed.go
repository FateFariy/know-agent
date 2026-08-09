package shared

import (
	"regexp"
	"strings"

	"github.com/duke-git/lancet/v2/strutil"
)

type Tokenizer interface {
	// SegmentWords 返回词文本列表，适用于搜索索引等场景
	SegmentWords(text string) []string
}

var (
	sectionSplitter = regexp.MustCompile(`[/\\.]`)
)

// KeywordSeed 表示构建关键词所需的输入上下文
type KeywordSeed struct {
	title       string
	sectionPath string
	text        string
}

func NewKeywordSeed(title, sectionPath, text string) *KeywordSeed {
	return &KeywordSeed{
		title:       strings.TrimSpace(title),
		sectionPath: strings.TrimSpace(sectionPath),
		text:        strings.TrimSpace(text),
	}
}

func (k *KeywordSeed) Build(tokenizer Tokenizer) []string {
	seen := make(map[string]bool, 12)
	keywords := make([]string, 0, 12)

	add := func(words ...string) {
		for i := 0; i < len(words) && len(keywords) < 12; i++ {
			w := strutil.Trim(words[i])
			if len(w) >= 2 && !seen[w] {
				seen[w] = true
				keywords = append(keywords, w)
			}
		}
	}

	add(k.title)

	if k.sectionPath != "" {
		add(sectionSplitter.Split(k.sectionPath, -1)...)
	}
	add(tokenizer.SegmentWords(k.text)...)

	return keywords
}
