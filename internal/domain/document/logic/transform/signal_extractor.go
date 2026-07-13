package transform

import (
	"context"
	"regexp"
	"strings"

	"github.com/duke-git/lancet/v2/slice"
	"github.com/duke-git/lancet/v2/strutil"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/document/logic/transform/signal"
	"github.com/swiftbit/know-agent/internal/domain/document/model/vo"
	"github.com/swiftbit/know-agent/internal/domain/document/support"
)

var (
	// inlineExplicitStepBoundaryPattern 显式步骤边界（支持中文数字与阿拉伯数字）
	inlineExplicitStepBoundaryPattern = regexp.MustCompile(`(?:第\s*[0-9一二三四五六七八九十百]+\s*步|步骤\s*[0-9一二三四五六七八九十百]+)\s*[:：、.]`)

	// specialLinePattern 整行仅由分隔符构成，视为特殊结构（表格/列表装饰）
	specialLinePattern = regexp.MustCompile(`^[:\-\\s|]+$`)
)

// SignalExtractor 文档结构信号抽取器：按行对文本进行结构识别，产出标题/正文/候选标题等信号
type SignalExtractor struct {
	lineClassifier       *support.DocumentLineClassifier // 行分类器，判断行的类型，如标题、列表、表格等
	detectorsManager     *signal.DetectorsManager        // 检测器管理器，包含所有检测器，识别显式标题结构
	maxPlainHeadingChars int
}

// extractorOption SignalExtractor 的可选参数（通过 WithXXX 函数注入）
type extractorOption struct {
	documentTitle        string // 文档标题，用于生成 DocumentTitle 信号
	maxPlainHeadingChars int    // 朴素标题判断的字符长度上限
}

// WithDocumentTitle 注入文档标题
func WithDocumentTitle(documentTitle string) TransformerOption {
	return WrapTransformerImplSpecificOptFn(func(opts *extractorOption) {
		opts.documentTitle = documentTitle
	})
}

// WithMaxPlainHeadingChars 注入朴素标题的最大字符数阈值
func WithMaxPlainHeadingChars(maxPlainHeadingChars int) TransformerOption {
	return WrapTransformerImplSpecificOptFn(func(opts *extractorOption) {
		opts.maxPlainHeadingChars = maxPlainHeadingChars
	})
}

func NewSignalExtractor() *SignalExtractor {
	return &SignalExtractor{
		detectorsManager: signal.NewDefaultDetectorsManager(),
		lineClassifier:   &support.DocumentLineClassifier{},
	}
}

// Transform 从原始文本中抽取结构信号
//
// 整体流程：
//  1. 解析可选参数，规范化文档标题
//  2. 将 text 切分为逻辑行（处理显式步骤内联切分等）
//  3. 统计行频率，用于启发式判断「反复出现的模板行」
//  4. 若存在文档标题，则以最高置信度输出一个 DocumentTitle 信号
//  5. 逐行调用检测器链 classify，生成标题/正文/候选标题信号
//  6. 汇总规范化文本作为上下文并返回批量信号
//
// 返回：批量结构信号，供后续聚合使用
func (e *SignalExtractor) Transform(ctx context.Context, text string, opts ...TransformerOption) *vo.DocumentStructureSignalBatch {
	// 聚合可选参数
	opt := GetTransformerImplSpecificOptions[extractorOption](&extractorOption{}, opts...)
	e.maxPlainHeadingChars = opt.maxPlainHeadingChars
	normalizedTitle := strutil.Trim(opt.documentTitle)

	// 构建逻辑行：物理行 → 按显式步骤边界拆分为多个逻辑段
	logicalLines := e.buildLogicalLines(text)
	// 行频率表：用于后续判断是否为常见模板/重复页眉等
	lineFrequency := e.buildLineFrequency(logicalLines)
	signals := make([]*vo.DocumentStructureSignal, 0, len(logicalLines)+1)

	// 文档标题作为首条信号（置信度 1.0）
	if normalizedTitle != "" {
		signals = append(signals, &vo.DocumentStructureSignal{
			RawText:        normalizedTitle,
			NormalizedText: normalizedTitle,
			Kind:           vo.SignalKindDocumentTitle,
			Title:          normalizedTitle,
			Confidence:     1.0,
		})
	}

	// 构造检测器上下文：共享文档标题 + 行频率
	detectorCtx := signal.NewDetectorContext(opt.documentTitle, lineFrequency)

	// 逐行分类：为每个逻辑行产出一个结构信号
	for index := 0; index < len(logicalLines); index++ {
		signals = append(signals, e.classify(detectorCtx, logicalLines, index))
	}

	// 汇总规范化文本作为上下文，便于下游做语义与格式回查
	contextLines := slice.Map(logicalLines, func(index int, line *vo.DocumentStructureLogicalLine) string { return line.NormalizedText })

	return &vo.DocumentStructureSignalBatch{
		ContextLines: contextLines,
		Signals:      signals,
	}
}

// classify 对单个逻辑行进行结构分类：先运行检测器链，再做朴素标题兜底，最后回归正文
//
// 判定顺序：
//  1. 尝试由 detectorsManager 的显式检测器识别（编号标题/列表/引用等）
//  2. 若检测器未命中，且行分类器也未判定为标题，则按「上下文 + 字符特征」判断是否为朴素标题候选
//  3. 其余情况统一标记为正文 Body
//
// 返回：单个结构信号（已携带行号/原始文本/缩进级别）
func (e *SignalExtractor) classify(detCtx *signal.DetectorContext, logicalLines []*vo.DocumentStructureLogicalLine, index int) *vo.DocumentStructureSignal {
	logicalLine := logicalLines[index]
	lineNo := logicalLine.LogicalLineNo
	rawText := logicalLine.RawText
	normalized := logicalLine.NormalizedText

	// 构建该行的前后文信息（前后非空行、是否空行间隔），用于后续启发式判断
	lineContext := e.buildContext(logicalLines, index)
	detCtx.LineContext = lineContext

	// 调用显式检测器链：命中则直接返回信号（置信度由检测器给出）
	result := e.detectorsManager.Detect(detCtx, normalized, signal.WithMaxPlainHeadingChars(e.maxPlainHeadingChars))
	if result != nil {
		// 回填检测器未设置的行级元数据
		result.LineNo = lineNo
		result.RawText = rawText
		result.NormalizedText = normalized
		result.IndentLevel = logicalLine.IndentLevel
		return result
	}

	// 构造 fallback 信号的基础字段
	fallbackSignal := &vo.DocumentStructureSignal{
		LineNo:         lineNo,
		RawText:        rawText,
		NormalizedText: normalized,
		Title:          normalized,
		IndentLevel:    logicalLine.IndentLevel,
	}

	// 基础分类器与朴素标题判断：
	// - 当分类器未识别为标题，但 looksLikePlainHeading 为真时，标记为 HeadingCandidate（0.58 置信度）
	// - LevelHint：有空行在前 → 认为更像一级标题；否则更像二级小标题
	fallback := e.lineClassifier.Classify(normalized)
	if !fallback.IsHeading() && support.LooksLikePlainHeading(lineContext, normalized, e.maxPlainHeadingChars) {
		fallbackSignal.Kind = vo.SignalKindHeadingCandidate
		fallbackSignal.Reasons = []string{"plain-heading-candidate"}
		// 有空行间隔的标题通常层级更高，无空行则视为二级候选
		fallbackSignal.LevelHint = utils.Ternary(lineContext.BlankBefore, 1, 2)
		fallbackSignal.Confidence = 0.58
		return fallbackSignal
	}

	// 默认降级为正文（最安全的 fallback）
	fallbackSignal.Kind = vo.SignalKindBody
	fallbackSignal.Reasons = []string{"body"}
	fallbackSignal.Confidence = 1.0
	return fallbackSignal
}

// buildLogicalLines 将原始文本切分为逻辑行

// 策略：
//  1. 按 \n 切出物理行，保留首尾空格的原始行用于 indent 计算
//  2. 对每个物理行尝试按「显式步骤边界」进一步拆分，使「第1步:...第2步:...」拆成多个逻辑段
//  3. 每个段单独编号为逻辑行，便于后续按段生成结构信号
//
// 返回：按出现顺序排列的逻辑行切片
func (e *SignalExtractor) buildLogicalLines(parsedText string) []*vo.DocumentStructureLogicalLine {
	if parsedText == "" {
		return nil
	}

	rawLines := strings.Split(parsedText, "\n")
	logicalLines := make([]*vo.DocumentStructureLogicalLine, 0, len(rawLines))
	logicalLineNo := 1

	// 遍历每个物理行，可能拆出多个逻辑段
	for index := 0; index < len(rawLines); index++ {
		rawLine := strutil.Trim(rawLines[index])
		segments := e.splitInlineSegments(rawLines[index])
		// 无段（空行或特殊单行）：直接作为一个逻辑行
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

		// 显式步骤切分后：每个段为独立逻辑行，记录段序号与缩进
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

// buildLineFrequency 构建行文本出现频率表, 检测器判断「重复出现的模板行/页眉/页脚」等，降低对其的标题置信度
func (e *SignalExtractor) buildLineFrequency(logicalLines []*vo.DocumentStructureLogicalLine) map[string]int {
	frequency := make(map[string]int)
	for _, line := range logicalLines {
		// 空行不计入频率，避免大量空行影响分布判断
		if line.NormalizedText != "" {
			frequency[line.NormalizedText]++
		}
	}
	return frequency
}

// buildContext 构造行级前后文信息：向前/向后查找首个非空行，并记录遍历途中是否存在空行间隔
func (e *SignalExtractor) buildContext(logicalLines []*vo.DocumentStructureLogicalLine, currentIndex int) *vo.LineContext {
	// 向前扫描：遇到空行标记 BlankBefore，遇到首个非空行作为 PreviousNonBlank 并停止
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

	// 向后扫描：对称逻辑
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

	return &vo.LineContext{
		PreviousNonBlank: previousNonBlank,
		NextNonBlank:     nextNonBlank,
		BlankBefore:      blankBefore,
		BlankAfter:       blankAfter,
	}
}

// splitInlineSegments 对单个物理行进行内联切分
//
// 规则：
//  1. 空行 → 不切分（返回 nil 表示作为原始空行处理）
//  2. 以 # | > 开头，或为纯分隔行 → 不切分，整体保留（视为 Markdown 结构）
//  3. 其他 → 按「第N步 / 步骤N」等显式步骤边界切分，每个片段返回一个段
func (e *SignalExtractor) splitInlineSegments(rawLine string) []string {
	trimmed := strutil.Trim(rawLine)
	if trimmed == "" {
		return nil
	}

	// Markdown 特殊结构保留完整原始行
	if trimmed[0] == '#' || trimmed[0] == '|' || trimmed[0] == '>' || specialLinePattern.MatchString(trimmed) {
		return []string{rawLine}
	}

	// 收集显式步骤边界的起点（零宽断言），从每个边界处作为新段起点
	matches := inlineExplicitStepBoundaryPattern.FindAllStringIndex(rawLine, -1)
	// boundaries[0]=0 确保首段内容从行首开始
	boundaries := make([]int, 1, len(matches)+1)
	for _, match := range matches {
		// 跳过位于行首的边界（避免产生空段）
		if match[0] > 0 {
			boundaries = append(boundaries, match[0])
		}
	}

	// 无有效边界 → 保持原行
	if len(boundaries) == 1 {
		return []string{rawLine}
	}

	// 末尾追加行末作为最后一个分段边界
	boundaries = append(boundaries, len(rawLine))
	segments := make([]string, 0, len(boundaries)-1)
	for index := 0; index < len(boundaries)-1; index++ {
		start, end := boundaries[index], boundaries[index+1]
		segment := strutil.Trim(rawLine[start:end])
		if segment != "" {
			segments = append(segments, segment)
		}
	}
	// 全部为空段时回退为整行（防御性兜底）
	return utils.Ternary(len(segments) == 0, []string{rawLine}, segments)
}

// countIndentLevel 计算行的缩进级别（空格数 + tab×4）
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
		// 一个制表符按 4 个空格折算
		if c == '\t' {
			indent += 4
			continue
		}
		// 遇到首个非空白字符停止计数
		break
	}
	return indent
}
