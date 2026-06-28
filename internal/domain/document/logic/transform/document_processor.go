package transform

import (
	"context"
	"regexp"
	"slices"
	"strings"

	"github.com/duke-git/lancet/v2/strutil"

	"github.com/swiftbit/know-agent/internal/domain/document/logic/parse"
	"github.com/swiftbit/know-agent/internal/domain/document/model/vo"
)

var englishPattern = regexp.MustCompile(`[A-Za-z]`)
var headingPattern = regexp.MustCompile(`^#{1,6}\s+.+`)

type DocumentProcessor struct {
	registry parse.Registry
}

type processorOption struct {
	fileType string
}

// WithFileType 设置文件类型
func WithFileType(fileType string) TransformerOption {
	return WrapTransformerImplSpecificOptFn(func(opts *processorOption) {
		opts.fileType = fileType
	})
}

func NewDocumentProcessor(registry parse.Registry) *DocumentProcessor {
	return &DocumentProcessor{
		registry: registry,
	}
}

// Transform 处理文档并返回分析结果
// 包括文本提取、清理、结构分析和质量评估，用于后续切块决策
func (p *DocumentProcessor) Transform(ctx context.Context, parsedText string, opts ...TransformerOption) (any, error) {
	// 清理文本（去除多余空格、格式化等）
	cleanedText := p.cleanupText(parsedText)

	// 统计标题数量（用于评估文档结构）
	// todo: 后续使用结构节点提取器增强标题识别
	//   List<DocumentStructureNodeCandidate> structureNodes = structureNodeExtractor.extract(originalFileName, cleanedText);
	//   int headingCount = countHeadings(cleanedText, structureNodes);
	headingCount := p.countHeadings(cleanedText)

	// 提取段落列表（用于分段统计）
	paragraphList := p.extractParagraphs(cleanedText)

	// 计算最大段落长度（用于评估段落复杂度）
	maxParagraph := slices.MaxFunc(paragraphList, func(a, b string) int { return len(a) - len(b) })

	// 估算token数量（用于成本和分块决策）
	tokenCount := p.estimateTokenCount(cleanedText)

	// 评估文档结构级别（标题数量+段落数量综合判断）
	structureLevel := p.evaluateStructureLevel(headingCount, len(paragraphList))

	// 评估内容质量级别（基于文本完整性、可读性等）
	contentQualityLevel := p.evaluateContentQuality(cleanedText)

	// 返回文档分析结果
	return &vo.DocumentAnalysisResult{
		ParsedText:          cleanedText,
		CharCount:           len(cleanedText),
		TokenCount:          tokenCount,
		StructureLevel:      structureLevel,
		ContentQualityLevel: contentQualityLevel,
		HeadingCount:        headingCount,
		ParagraphCount:      len(paragraphList),
		MaxParagraphLength:  len(maxParagraph),
	}, nil
}

// cleanupText 清理文本，移除换行符、制表符、空格等
func (p *DocumentProcessor) cleanupText(rawText string) string {
	if rawText == "" {
		return ""
	}

	cleaned := rawText
	cleaned = strings.ReplaceAll(cleaned, "\r\n", "\n")
	cleaned = strings.ReplaceAll(cleaned, "\r", "\n")
	cleaned = strings.ReplaceAll(cleaned, "\x00", " ")

	cleaned = regexp.MustCompile(`[\t\x0B\f]+`).ReplaceAllString(cleaned, " ")
	cleaned = regexp.MustCompile(`\n{3,}`).ReplaceAllString(cleaned, "\n\n")
	cleaned = regexp.MustCompile(` {2,}`).ReplaceAllString(cleaned, " ")

	return strutil.Trim(cleaned)
}

// todo 待完善，结构节点提取和 heading 计数
func (p *DocumentProcessor) countHeadings(text string) int {
	count := 0
	for _, line := range strings.Split(text, "\n") {
		if p.isHeading(line) {
			count++
		}
	}
	return count
}

// todo 待完善，判断行是否为标题
func (p *DocumentProcessor) isHeading(line string) bool {
	line = strutil.Trim(line)
	if len(line) < 3 {
		return false
	}
	if headingPattern.MatchString(line) {
		return true
	}
	return false
}

// extractParagraphs 提取段落
func (p *DocumentProcessor) extractParagraphs(text string) []string {
	paragraphs := strings.Split(text, "\n\n")
	paragraphList := make([]string, 0, len(paragraphs))
	for _, paragraph := range paragraphs {
		trimmed := strutil.Trim(paragraph)
		if trimmed != "" {
			paragraphList = append(paragraphList, trimmed)
		}
	}
	return paragraphList
}

// estimateTokenCount 估计token数量
func (p *DocumentProcessor) estimateTokenCount(text string) int {
	englishWordCount, chineseCharCount := 0, 0

	for _, word := range strings.Fields(text) {
		if englishPattern.MatchString(word) {
			englishWordCount++
		}
	}

	for _, r := range text {
		if r >= '\u4e00' && r <= '\u9fa5' {
			chineseCharCount++
		}
	}

	nonChineseLength := len(text) - chineseCharCount
	return englishWordCount + chineseCharCount + max(1, nonChineseLength/4)
}

// evaluateStructureLevel 评估文档结构等级
func (p *DocumentProcessor) evaluateStructureLevel(headingCount, paragraphCount int) int {
	if headingCount >= 5 {
		return vo.StructureLevelHigh
	}
	if headingCount >= 2 {
		return vo.StructureLevelMedium
	}
	if paragraphCount >= 3 {
		return vo.StructureLevelLow
	}
	return vo.StructureLevelLow
}

// evaluateContentQuality 评估文档内容质量等级
func (p *DocumentProcessor) evaluateContentQuality(text string) int {
	charCount := len(text)
	if strutil.IsBlank(text) || charCount < 20 {
		return vo.ContentQualityLevelLow
	}

	brokenCharCount := strings.Count(text, "\uFFFD")
	brokenRatio := float64(brokenCharCount) / float64(charCount)
	if brokenRatio > 0.02 || charCount < 100 {
		return vo.ContentQualityLevelLow
	}
	if brokenRatio > 0.005 || charCount < 500 {
		return vo.ContentQualityLevelMedium
	}

	return vo.ContentQualityLevelHigh
}
