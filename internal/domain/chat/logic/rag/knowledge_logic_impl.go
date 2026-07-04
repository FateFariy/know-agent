package rag

import (
	"context"
	"regexp"
	"strings"

	"github.com/duke-git/lancet/v2/strutil"

	vo2 "github.com/swiftbit/know-agent/internal/domain/chat/logic/rag/model/vo"
	"github.com/swiftbit/know-agent/internal/domain/knowledge/adapter"
)

const (
	IndexStatusBuildSuccess = 2
	MaxKeywordTerms         = 8
)

var (
	alnumTokenPattern   = regexp.MustCompile(`[a-z0-9._-]{2,}`)
	chineseTokenPattern = regexp.MustCompile(`\p{Han}{2,}`)
	chineseNoisePhrases = []string{
		"请问", "帮我", "一下子", "一下", "如何", "怎么", "什么", "哪个", "这个", "那个", "是否", "关于", "可以", "需要", "想问", "看看",
	}
	chineseSegmentSplitPattern = regexp.MustCompile(`[的和及与或]`)
	spacePattern               = regexp.MustCompile(`\s+`)
)

// DocumentKnowledgeLogicImpl 文档知识服务实现
type DocumentKnowledgeLogicImpl struct {
	repo adapter.KnowledgeRepository
	port *adapter.KnowledgePort
}

// NewDocumentKnowledgeService 构造函数
func NewDocumentKnowledgeService(repo adapter.KnowledgeRepository, port *adapter.KnowledgePort) *DocumentKnowledgeLogicImpl {
	return &DocumentKnowledgeLogicImpl{
		repo: repo,
		port: port,
	}
}

// ListRetrievableDocuments 列出可检索的文档
func (s *DocumentKnowledgeLogicImpl) ListRetrievableDocuments(ctx context.Context) ([]*vo2.KnowledgeDocument, error) {
	return s.repo.SelectRetrievableDocuments(ctx)
}

// ExtractKeywordTerms 从查询句中提取最多 MaxKeywordTerms 个关键词项
func (s *DocumentKnowledgeLogicImpl) ExtractKeywordTerms(question string) []string {
	normalized := normalizeQuestion(question)
	if strutil.IsBlank(normalized) {
		return nil
	}

	terms := make([]string, 0, MaxKeywordTerms*2)
	seen := make(map[string]struct{}, MaxKeywordTerms*2)

	// 步骤 1：提取字母/数字 token
	for _, t := range alnumTokenPattern.FindAllString(normalized, -1) {
		if _, ok := seen[t]; ok {
			continue
		}
		seen[t] = struct{}{}
		terms = append(terms, t)
	}

	// 步骤 2：提取中文 token → 按分割符拆分 → 再补充 n-gram 子段
	for _, raw := range chineseTokenPattern.FindAllString(normalized, -1) {
		for _, segment := range splitChineseSegments(raw) {
			addChineseSegmentTerms(segment, &terms, seen)
			if len(terms) >= MaxKeywordTerms*2 {
				break
			}
		}
		if len(terms) >= MaxKeywordTerms*2 {
			break
		}
	}

	// 步骤 3：长度 >=2 过滤 + 限制数量
	result := make([]string, 0, len(terms))
	for _, t := range terms {
		if len(t) >= 2 {
			result = append(result, t)
		}
		if len(result) >= MaxKeywordTerms {
			break
		}
	}
	return result
}

// splitChineseSegments 去除噪音短语后再按分隔符切分，所有片段按长度 >=2 保留
func splitChineseSegments(chineseToken string) []string {
	cleaned := removeChineseNoisePhrases(chineseToken)
	if len(cleaned) < 2 {
		return nil
	}

	seen := make(map[string]struct{})
	var segments []string
	if _, ok := seen[cleaned]; !ok {
		seen[cleaned] = struct{}{}
		segments = append(segments, cleaned)
	}
	for _, part := range chineseSegmentSplitPattern.Split(cleaned, -1) {
		normalized := strutil.Trim(part)
		if len(normalized) < 2 {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		segments = append(segments, normalized)
	}
	return segments
}

// addChineseSegmentTerms 为中文段补充：原词 + head n-gram + tail n-gram + sliding n-gram
func addChineseSegmentTerms(segment string, terms *[]string, seen map[string]struct{}) {
	if strutil.IsBlank(segment) || len(segment) < 2 {
		return
	}

	runes := []rune(segment)
	addIfAbsent := func(t string) {
		if _, ok := seen[t]; ok {
			return
		}
		seen[t] = struct{}{}
		*terms = append(*terms, t)
	}

	if len(runes) <= 12 {
		addIfAbsent(segment)
	}

	addTailNgrams(runes, addIfAbsent)
	addHeadNgrams(runes, addIfAbsent)
	addSlidingNgrams(runes, addIfAbsent)
}

func addTailNgrams(runes []rune, add func(string)) {
	maxGram := min(4, len(runes))
	for size := maxGram; size >= 2; size-- {
		if len(runes)-size < 0 {
			continue
		}
		add(string(runes[len(runes)-size:]))
	}
}

func addHeadNgrams(runes []rune, add func(string)) {
	maxGram := min(4, len(runes))
	for size := maxGram; size >= 2; size-- {
		add(string(runes[:size]))
	}
}

func addSlidingNgrams(runes []rune, add func(string)) {
	maxGram := min(4, len(runes))
	for size := maxGram; size >= 2; size-- {
		for i := 0; i <= len(runes)-size; i++ {
			add(string(runes[i : i+size]))
		}
	}
}

// normalizeQuestion 标准化查询：去除换行/制表符、合并空白、转为小写
func normalizeQuestion(question string) string {
	if strutil.IsBlank(question) {
		return ""
	}
	normalized := strings.ToLower(strutil.Trim(question))
	normalized = spacePattern.ReplaceAllString(normalized, " ")
	return normalized
}

// removeChineseNoisePhrases 从文本中去除常见的中文噪音短语
func removeChineseNoisePhrases(text string) string {
	if strutil.IsBlank(text) {
		return ""
	}
	normalized := strutil.Trim(text)
	for _, phrase := range chineseNoisePhrases {
		normalized = strings.ReplaceAll(normalized, phrase, "")
	}
	return strutil.Trim(normalized)
}

// KeywordWeight 关键词命中在 chunk_text 时的基础权重（索引越靠前权重越高）
func KeywordWeight(index int) int {
	return max(1, 6-index)
}

// SectionKeywordWeight 关键词命中在 section_path 时的额外权重（基础权重 + 2）
func SectionKeywordWeight(index int) int {
	return KeywordWeight(index) + 2
}
