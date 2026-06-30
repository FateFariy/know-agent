package logic

import (
	"context"
	"regexp"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/duke-git/lancet/v2/slice"
	"github.com/duke-git/lancet/v2/strutil"

	"github.com/swiftbit/know-agent/internal/domain/document/logic/parse"
	"github.com/swiftbit/know-agent/internal/domain/document/logic/transform"
	"github.com/swiftbit/know-agent/internal/domain/document/model/vo"
	"github.com/swiftbit/know-agent/internal/domain/document/support"
)

var (
	// englishPattern      = regexp.MustCompile(`[A-Za-z]`)     // 匹配英文字母
	controlSpaceRegex   = regexp.MustCompile(`[\t\x0B\f]+`) // 匹配制表符、垂直制表符、换页符等空白字符（但不包括空格、换行）
	multiNewlineRegex   = regexp.MustCompile(`\n{3,}`)      // 匹配连续3个或更多换行符
	multiSpaceRegex     = regexp.MustCompile(` {2,}`)       // 匹配连续2个或更多空格
	paragraphSplitRegex = regexp.MustCompile(`\n\s*\n`)     // 匹配多个换行符
)

type TextPreProcessLogicImpl struct {
	registry          parse.Registry
	signalExtractor   *transform.SignalExtractor   // 信号抽取：将原始文本拆分为标题/列表/正文等结构信号
	ambiguityResolver *transform.AmbiguityResolver // 歧义消解：对候选标题进行 LLM 二次判定（若配置启用）
	hierarchyResolver *transform.HierarchyResolver // 层级构建：基于信号流组装父子关系与嵌套列表
	treeValidator     *transform.TreeValidator     // 树验证：规范化父子关系、深度、路径与兄弟链表
	classifier        *support.DocumentLineClassifier
}

func NewTextPreProcessLogicImpl(registry parse.Registry, signalExtractor *transform.SignalExtractor, ambiguityResolver *transform.AmbiguityResolver,
	hierarchyResolver *transform.HierarchyResolver, treeValidator *transform.TreeValidator) *TextPreProcessLogicImpl {
	return &TextPreProcessLogicImpl{
		registry:          registry,
		signalExtractor:   signalExtractor,
		ambiguityResolver: ambiguityResolver,
		hierarchyResolver: hierarchyResolver,
		treeValidator:     treeValidator,
		classifier:        &support.DocumentLineClassifier{},
	}
}

// PreProcess 处理文档并返回分析结果，包括文本提取、清理、结构分析和质量评估，用于后续切块决策
func (p *TextPreProcessLogicImpl) PreProcess(ctx context.Context, documentTitle, parsedText, fileType string, opts ...transform.TransformerOption) (*vo.DocumentAnalysisResult, error) {
	// 解析文本
	parser := p.registry.Get(fileType)
	parsedText, err := parser.Parse(ctx, []byte(parsedText))
	if err != nil {
		return nil, err
	}

	// 清理文本（去除多余空格、格式化等）
	cleanedText := p.cleanupText(parsedText)

	// 统计标题数量（用于评估文档结构）
	structureNodes := p.extractStructureNodes(ctx, documentTitle, cleanedText, opts...)
	headingCount := p.countHeadings(cleanedText, structureNodes)

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
		StructureNodes:      structureNodes,
	}, nil
}

// extractStructureNodes 执行文档结构节点抽取：输入文档标题与原始文本，输出结构候选节点列表。
//
// 整体流程：
//  1. 规范化标题与正文文本，提供稳定的后续处理输入
//  2. 短文本退化处理：当正文为空时，直接返回单一文档根节点（避免后续组件做无意义计算）
//  3. 信号抽取：SignalExtractor 将文本切分为逻辑行并识别结构信号（标题/列表/正文等）
//  4. 歧义消解：AmbiguityResolver 对不确定的候选标题做 LLM 判定
//  5. 层级构建：HierarchyResolver 将扁平信号流组织成带父子关系的节点草稿
//  6. 树验证：TreeValidator 规范化 Draft 树并输出最终的候选节点列表
func (p *TextPreProcessLogicImpl) extractStructureNodes(ctx context.Context, documentTitle, parsedText string, opts ...transform.TransformerOption) []*vo.DocumentStructureNodeCandidate {
	normalizedTitle := strutil.Trim(documentTitle)
	if normalizedTitle == "" {
		normalizedTitle = "文档"
	}
	normalizedText := strutil.Trim(parsedText)

	// 短文本退化：无正文内容时直接返回文档根节点（避免噪声处理）
	if normalizedText == "" {
		return []*vo.DocumentStructureNodeCandidate{
			{
				NodeNo:        1,
				NodeType:      vo.NodeTypeDocument,
				Title:         normalizedTitle,
				AnchorText:    normalizedTitle,
				CanonicalPath: "/document",
			},
		}
	}

	// 信号抽取 — 获得结构信号批量（含原始上下文行）
	signalBatch := p.signalExtractor.Transform(ctx, parsedText, opts...)

	// 歧义消解 — 对信号中的候选项做 LLM 二次判定（若配置/实例可用）
	resolvedSignals, _ := p.ambiguityResolver.Transform(ctx, documentTitle, signalBatch.ContextLines, signalBatch.Signals, opts...)
	if resolvedSignals == nil {
		resolvedSignals = signalBatch.Signals
	}

	// 层级构建 — 将信号流转为草稿节点树（含根节点）
	drafts := p.hierarchyResolver.Transform(normalizedTitle, resolvedSignals, opts...)

	// 树验证与规范化 — 生成最终候选节点列表
	return p.treeValidator.Transform(normalizedTitle, drafts, opts...)
}

// cleanupText 清理文本，移除换行符、制表符、空格等
func (p *TextPreProcessLogicImpl) cleanupText(rawText string) string {
	if rawText == "" {
		return ""
	}

	cleaned := rawText
	cleaned = strings.ReplaceAll(cleaned, "\r\n", "\n")
	cleaned = strings.ReplaceAll(cleaned, "\r", "\n")
	cleaned = strings.ReplaceAll(cleaned, "\x00", " ")

	cleaned = controlSpaceRegex.ReplaceAllString(cleaned, " ")
	cleaned = multiNewlineRegex.ReplaceAllString(cleaned, "\n\n")
	cleaned = multiSpaceRegex.ReplaceAllString(cleaned, " ")

	return strutil.Trim(cleaned)
}

// countHeadings 统计标题数量
func (p *TextPreProcessLogicImpl) countHeadings(text string, structureNodes []*vo.DocumentStructureNodeCandidate) int {
	if len(structureNodes) != 0 {
		count := slice.CountBy(structureNodes, func(_ int, node *vo.DocumentStructureNodeCandidate) bool {
			return node != nil && node.NodeType == vo.NodeTypeSection && node.Depth > 0
		})
		if count > 0 {
			return count
		}
	}
	count := 0
	for _, line := range strings.Split(text, "\n") {
		if p.classifier.Classify(line).IsHeading() {
			count++
		}
	}
	return count
}

// extractParagraphs 提取段落
func (p *TextPreProcessLogicImpl) extractParagraphs(text string) []string {
	rawParagraphs := paragraphSplitRegex.Split(text, -1)
	paragraphList := make([]string, 0, len(rawParagraphs))
	for _, paragraph := range rawParagraphs {
		trimmed := strutil.Trim(paragraph)
		if trimmed != "" {
			paragraphList = append(paragraphList, trimmed)
		}
	}
	return paragraphList
}

// estimateTokenCount 估计token数量
func (p *TextPreProcessLogicImpl) estimateTokenCount(text string) int {
	englishCount, chineseCount := 0, 0

	// 统计英文单词数量
	for _, word := range strings.Fields(text) {
		if englishPattern.MatchString(word) {
			englishCount++
		}
	}

	// 统计中文字符数量
	for _, r := range text {
		if unicode.Is(unicode.Han, r) {
			chineseCount++
		}
	}

	// 非中英文字符按 4 字符折算 1 Token
	baseToken := max(1, (utf8.RuneCountInString(text)-chineseCount-englishCount)/4)

	return chineseCount + englishCount + baseToken
}

// evaluateStructureLevel 评估文档结构等级
func (p *TextPreProcessLogicImpl) evaluateStructureLevel(headingCount, paragraphCount int) int {
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
func (p *TextPreProcessLogicImpl) evaluateContentQuality(text string) int {
	charCount := utf8.RuneCountInString(text)
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
