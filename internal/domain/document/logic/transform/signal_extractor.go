package transform

import (
	"context"
	"regexp"
	"strings"

	"github.com/duke-git/lancet/v2/strutil"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/document/model/vo"
)

var (
	// Markdown 标题：如 "## 概述"，捕获层级和标题文本
	markdownHeadingPattern = regexp.MustCompile(`^(#+)\s+(.+)$`)
	// 多级数字编号：如 "1.2.3 配置说明"，捕获编号和标题文本
	decimalHeadingPattern = regexp.MustCompile(`^(\d+(?:\.\d+)+)\s*[、.]?\s*(.+)$`)
	// 单级数字编号：如 "1、概述" 或 "2. 安装"，可能是标题或列表项
	singleLevelDigitPattern = regexp.MustCompile(`^(\d+)\s*[、.]\s*(.+)$`)
	// 中文章节标题：如 "第一章 绪论"、"第三节 方法
	chapterPattern = regexp.MustCompile(`^(第([一二三四五六七八九十百\d]+)[章节条部分])\s*(.+)$`)
	// 附录标题：如 "附录A 术语表"
	appendixPattern = regexp.MustCompile(`^(附录\s*([A-Za-z一二三四五六七八九十百\d]+))(?:\s+(.+))?$`)
	// 中文大纲编号：如 "一、项目背景"、"三、实施方案"
	chineseOutlinePattern = regexp.MustCompile(`^([一二三四五六七八九十百]+)[、.]\s*(.+)$`)
	// 显式步骤标记：如 "第一步：安装" 或 "步骤2：配置"
	explicitStepPattern = regexp.MustCompile(`^(?:第\s*([0-9一二三四五六七八九十百]+)\s*步|步骤\s*([0-9一二三四五六七八九十百]+))\s*[:：、.]?\s*(.+)$`)
	// 无序列表项：如 "- 项目一"、"* 项目二"、"• 项目三"
	bulletPattern = regexp.MustCompile(`^([-*+•])\s+(.+)$`)
	// 复选框：如 "[ ] 项目一"、"[x] 项目二"
	checkboxPattern = regexp.MustCompile(`^\[(?: |x|X)]\s+(.+)$`)
	// 页码噪声行：如 "第 3 页"、"Page 5"、"3 / 10"
	pageNoisePattern = regexp.MustCompile(`^(?:第\s*\d+\s*页|Page\s*\d+|\d+\s*/\s*\d+)$`)
	// 版权声明噪声行：包含常见版权/保密关键词
	copyrightNoisePattern = regexp.MustCompile(`.*(?:版权所有|未经授权|内部使用|copyright|all rights reserved|保密).*`)
	// 版本号噪声行：包含常见版本号关键词
	versionFooterPattern = regexp.MustCompile(`.*(?:\bV\d+(?:\.\d+)*\b|版本|修订|Rev\.?\s*\d+).*`)
	// 内联步骤边界匹配器：如 "第一步：安装" 或 "步骤2：配置"
	inlineExplicitStepBoundaryPattern = regexp.MustCompile(`(?=(?:第\s*[0-9一二三四五六七八九十百]+\s*步|步骤\s*[0-9一二三四五六七八九十百]+)\s*[:：、.])`)
	// 表格分隔符：如 "|"
	tableSplitPattern = regexp.MustCompile(`\|`)

	specialLinePattern = regexp.MustCompile(`^[:\-\\s|]+$`)
)

type SignalExtractor struct {
	maxPlainHeadingChars int
	lineClassifier       LineClassifier
}

type extractorOption struct {
	documentTitle string
}

type lineContext struct {
	previousNonBlank *vo.DocumentStructureLogicalLine
	nextNonBlank     *vo.DocumentStructureLogicalLine
	blankBefore      bool
	blankAfter       bool
}

func WithDocumentTitle(documentTitle string) TransformerOption {
	return WrapTransformerImplSpecificOptFn(func(opts *extractorOption) {
		opts.documentTitle = documentTitle
	})
}

func (e *SignalExtractor) Transform(ctx context.Context, text string, opts ...TransformerOption) *vo.DocumentStructureSignalBatch {
	opt := GetTransformerImplSpecificOptions[extractorOption](&extractorOption{}, opts...)
	normalizedTitle := strutil.Trim(opt.documentTitle)
	logicalLines := e.buildLogicalLines(text)
	lineFrequency := e.buildLineFrequency(logicalLines)
	signals := make([]*vo.DocumentStructureSignal, 0, len(logicalLines)+1)

	if normalizedTitle != "" {
		signals = append(signals, e.signal(0, normalizedTitle, normalizedTitle, 0,
			vo.SignalKindDocumentTitle, "", normalizedTitle, intPtr(0), nil, []string{}, 1.0))
	}

	for index := 0; index < len(logicalLines); index++ {
		logicalLine := logicalLines[index]
		lineCtx := e.buildContext(logicalLines, index)
		signals = append(signals, e.classify(normalizedTitle, logicalLine, lineCtx, lineFrequency))
	}

	contextLines := make([]string, len(logicalLines))
	for i, line := range logicalLines {
		contextLines[i] = line.NormalizedText
	}

	return &vo.DocumentStructureSignalBatch{
		ContextLines: contextLines,
		Signals:      signals,
	}
}

// buildLogicalLines 将原始纯文本拆成"逻辑行"列表
/*
  这里之所以不直接把按换行切出来的物理行原样往下传，是因为一条物理行里可能内联写了多个步骤，
  例如"步骤1：下载 步骤2：安装"。如果不先拆开，后续分类器就只能把整行当成一个信号，会让步骤识别和层级解析都变得不准确。

  最终每个 DocumentStructureLogicalLine 都会携带：
  逻辑行号、原始物理行号、物理行内的片段序号、缩进级别、原始片段文本、规范化文本。
  这些信息后面会被上下文构建、列表层级恢复和兄弟关系排序复用。
*/
func (e *SignalExtractor) buildLogicalLines(parsedText string) []*vo.DocumentStructureLogicalLine {
	if parsedText == "" {
		return nil
	}

	rawLines := strings.Split(parsedText, "\n")
	logicalLines := make([]*vo.DocumentStructureLogicalLine, 0, len(rawLines))
	logicalLineNo := 1

	for index := 0; index < len(rawLines); index++ {
		rawLine := strutil.Trim(rawLines[index])
		segments := e.splitInlineSegments(rawLines[index])

		if len(segments) == 0 {
			logicalLines = append(logicalLines, &vo.DocumentStructureLogicalLine{
				LogicalLineNo:  logicalLineNo,
				PhysicalLineNo: index + 1,
				SegmentNo:      1,
				IndentLevel:    0,
				RawText:        rawLines[index],
				NormalizedText: rawLine,
			})
			logicalLineNo++
			continue
		}

		for segmentIndex := 0; segmentIndex < len(segments); segmentIndex++ {
			segment := segments[segmentIndex]
			logicalLines = append(logicalLines, &vo.DocumentStructureLogicalLine{
				LogicalLineNo:  logicalLineNo,
				PhysicalLineNo: index + 1,
				SegmentNo:      segmentIndex + 1,
				IndentLevel:    e.countIndentLevel(segment),
				RawText:        segment,
				NormalizedText: strutil.Trim(segment),
			})
			logicalLineNo++
		}
	}

	return logicalLines
}

// buildLineFrequency 统计每个单词出现的频率
/*
   统计每条规范化逻辑行在文档中出现的次数。这一步主要服务于"重复噪声识别"：
   如果某一短行在全文中频繁重复出现，它更可能是页眉、页脚、版本尾注或版权提示，而不是正常正文或标题。
*/
func (e *SignalExtractor) buildLineFrequency(logicalLines []*vo.DocumentStructureLogicalLine) map[string]int {
	frequency := make(map[string]int)
	for _, line := range logicalLines {
		if line.NormalizedText != "" {
			frequency[line.NormalizedText]++
		}
	}
	return frequency
}

// buildContext 为当前逻辑行构造局部上下文
/*
   向前找到最近一个非空行，向后找到最近一个非空行，同时记录当前行前后是否跨过空行。
   这些上下文信息会直接影响：
   1. 纯文本标题判定；
   2. 单级编号是否更像列表项；
   3. 上一行是否在引入一个列表。
*/
func (e *SignalExtractor) buildContext(logicalLines []*vo.DocumentStructureLogicalLine, currentIndex int) *lineContext {
	var previousNonBlank *vo.DocumentStructureLogicalLine
	blankBefore := false

	for index := currentIndex - 1; index >= 0; index-- {
		candidate := logicalLines[index]
		if candidate.NormalizedText == "" {
			blankBefore = true
			continue
		}
		previousNonBlank = candidate
		break
	}

	var nextNonBlank *vo.DocumentStructureLogicalLine
	blankAfter := false

	for index := currentIndex + 1; index < len(logicalLines); index++ {
		candidate := logicalLines[index]
		if candidate.NormalizedText == "" {
			blankAfter = true
			continue
		}
		nextNonBlank = candidate
		break
	}

	return &lineContext{
		previousNonBlank: previousNonBlank,
		nextNonBlank:     nextNonBlank,
		blankBefore:      blankBefore,
		blankAfter:       blankAfter,
	}
}

// splitInlineSegments 将内联步骤边界匹配器匹配到的片段拆分成多个信号
func (e *SignalExtractor) splitInlineSegments(rawLine string) []string {
	trimmed := strutil.Trim(rawLine)
	if trimmed == "" {
		return nil
	}

	if trimmed[0] == '#' || trimmed[0] == '|' || trimmed[0] == '>' || specialLinePattern.MatchString(trimmed) {
		return []string{rawLine}
	}

	matches := inlineExplicitStepBoundaryPattern.FindAllStringIndex(rawLine, -1)
	boundaries := make([]int, 1, len(matches)+1)
	for _, match := range matches {
		if match[0] > 0 {
			boundaries = append(boundaries, match[0])
		}
	}

	if len(boundaries) == 1 {
		return []string{rawLine}
	}

	boundaries = append(boundaries, len(rawLine))
	segments := make([]string, 0, len(boundaries)-1)
	for index := 0; index < len(boundaries)-1; index++ {
		start, end := boundaries[index], boundaries[index+1]
		segment := strutil.Trim(rawLine[start:end])
		if segment != "" {
			segments = append(segments, segment)
		}
	}
	return utils.Ternary(len(segments) == 0, []string{rawLine}, segments)
}

// countIndentLevel 计算文本的缩进级别
func (e *SignalExtractor) countIndentLevel(text string) int {
	if text == "" {
		return 0
	}
	indent := 0
	for _, c := range text {
		if c == ' ' {
			indent++
			continue
		}
		if c == '\t' {
			indent += 4
			continue
		}
		break
	}
	return indent
}

func (e *SignalExtractor) signal(lineNo int,
	rawText string,
	normalized string,
	indentLevel int,
	kind vo.DocumentStructureSignalKind,
	code string,
	title string,
	levelHint *int,
	itemIndex *int,
	reasons []string,
	confidence float64) *vo.DocumentStructureSignal {

	if code == "" {
		code = ""
	}
	if title == "" {
		title = normalized
	}

	return &vo.DocumentStructureSignal{
		LineNo:         lineNo,
		RawText:        rawText,
		NormalizedText: normalized,
		Kind:           kind,
		NodeCode:       code,
		Title:          title,
		LevelHint:      levelHint,
		IndentLevel:    indentLevel,
		ItemIndex:      itemIndex,
		Reasons:        reasons,
		Confidence:     confidence,
	}
}
