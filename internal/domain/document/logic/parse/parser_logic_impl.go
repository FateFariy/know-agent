package parse

import (
	"context"
	"regexp"
	"strings"

	"github.com/duke-git/lancet/v2/strutil"

	"github.com/swiftbit/know-agent/internal/domain/document/logic"
	"github.com/swiftbit/know-agent/internal/domain/document/model/vo"
)

var englishPattern = regexp.MustCompile(`[A-Za-z]`)
var headingPattern = regexp.MustCompile(`^#{1,6}\s+.+`)

type ParserLogicImpl struct {
	parserRegistry ParserRegistry
}

var _ logic.ParserLogic = (*ParserLogicImpl)(nil)

func NewParserLogicImpl(parserRegistry ParserRegistry) *ParserLogicImpl {
	return &ParserLogicImpl{
		parserRegistry: parserRegistry,
	}
}

func (p *ParserLogicImpl) Parse(ctx context.Context, bytesData []byte, originalFileName string, mimeType int, fileType int) (*vo.DocumentAnalysisResult, error) {
	rawText, err := p.extractRawText(ctx, bytesData, fileType)
	if err != nil {
		return nil, err
	}

	cleanedText := p.cleanupText(rawText)

	headingCount := p.countHeadings(cleanedText)

	paragraphList := p.extractParagraphs(cleanedText)

	maxParagraphLength := 0
	for _, para := range paragraphList {
		if len(para) > maxParagraphLength {
			maxParagraphLength = len(para)
		}
	}

	tokenCount := p.estimateTokenCount(cleanedText)

	structureLevel := p.evaluateStructureLevel(headingCount, len(paragraphList))

	contentQualityLevel := p.evaluateContentQuality(cleanedText)

	return &vo.DocumentAnalysisResult{
		ParsedText:          cleanedText,
		CharCount:           len(cleanedText),
		TokenCount:          tokenCount,
		StructureLevel:      structureLevel,
		ContentQualityLevel: contentQualityLevel,
		HeadingCount:        headingCount,
		ParagraphCount:      len(paragraphList),
		MaxParagraphLength:  maxParagraphLength,
	}, nil
}

func (p *ParserLogicImpl) extractRawText(ctx context.Context, bytesData []byte, fileType int) (string, error) {
	if parser := p.parserRegistry.Get(fileType); parser != nil {
		return parser.Parse(ctx, bytesData)
	}
	return string(bytesData), nil
}

func (p *ParserLogicImpl) cleanupText(rawText string) string {
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

func (p *ParserLogicImpl) countHeadings(text string) int {
	count := 0
	for _, line := range strings.Split(text, "\n") {
		if p.isHeading(line) {
			count++
		}
	}
	return count
}

func (p *ParserLogicImpl) isHeading(line string) bool {
	line = strutil.Trim(line)
	if len(line) < 3 {
		return false
	}
	if headingPattern.MatchString(line) {
		return true
	}
	return false
}

func (p *ParserLogicImpl) extractParagraphs(text string) []string {
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

func (p *ParserLogicImpl) estimateTokenCount(text string) int {
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

func (p *ParserLogicImpl) evaluateStructureLevel(headingCount, paragraphCount int) int {
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

func (p *ParserLogicImpl) evaluateContentQuality(text string) int {
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

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
