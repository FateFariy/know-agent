package transform

import (
	"context"
	"regexp"
	"strings"

	"github.com/duke-git/lancet/v2/strutil"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/document/logic/transform/signal"
	"github.com/swiftbit/know-agent/internal/domain/document/model/vo"
)

var (
	inlineExplicitStepBoundaryPattern = regexp.MustCompile(`(?=(?:第\s*[0-9一二三四五六七八九十百]+\s*步|步骤\s*[0-9一二三四五六七八九十百]+)\s*[:：、.])`)
	specialLinePattern                = regexp.MustCompile(`^[:\-\\s|]+$`)
)

type SignalExtractor struct {
	maxPlainHeadingChars int
	lineClassifier       signal.LineClassifier
	detectorsManager     signal.DetectorsManager
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

func NewSignalExtractor() *SignalExtractor {
	return &SignalExtractor{
		maxPlainHeadingChars: 80,
		detectorsManager:     signal.NewDefaultDetectorsManager(),
	}
}

func (e *SignalExtractor) Transform(ctx context.Context, text string, opts ...TransformerOption) *vo.DocumentStructureSignalBatch {
	opt := GetTransformerImplSpecificOptions[extractorOption](&extractorOption{}, opts...)
	normalizedTitle := strutil.Trim(opt.documentTitle)
	logicalLines := e.buildLogicalLines(text)
	lineFrequency := e.buildLineFrequency(logicalLines)
	signals := make([]*vo.DocumentStructureSignal, 0, len(logicalLines)+1)

	if normalizedTitle != "" {
		signals = append(signals, &vo.DocumentStructureSignal{
			RawText:        normalizedTitle,
			NormalizedText: normalizedTitle,
			Kind:           vo.SignalKindDocumentTitle,
			Title:          normalizedTitle,
			Confidence:     1.0,
		})
	}

	detectorCtx := signal.NewDetectorContext(opt.documentTitle, lineFrequency)

	for index := 0; index < len(logicalLines); index++ {
		signals = append(signals, e.classify(logicalLines, index, detectorCtx))
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

func (e *SignalExtractor) buildLineFrequency(logicalLines []*vo.DocumentStructureLogicalLine) map[string]int {
	frequency := make(map[string]int)
	for _, line := range logicalLines {
		if line.NormalizedText != "" {
			frequency[line.NormalizedText]++
		}
	}
	return frequency
}

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

func (e *SignalExtractor) classify(logicalLines []*vo.DocumentStructureLogicalLine, index int, ctx *signal.DetectorContext) *vo.DocumentStructureSignal {
	logicalLine := logicalLines[index]
	lineNo := logicalLine.LogicalLineNo
	rawText := logicalLine.RawText
	normalized := logicalLine.NormalizedText

	baseSignal := &vo.DocumentStructureSignal{
		LineNo:         lineNo,
		RawText:        rawText,
		NormalizedText: normalized,
		IndentLevel:    logicalLine.IndentLevel,
		Kind:           vo.SignalKindBlank,
		Confidence:     1.0,
	}

	result := e.detectorsManager.Detect(ctx, normalized)
	if result != nil {
		result.LineNo = lineNo
		result.RawText = rawText
		result.NormalizedText = normalized
		result.IndentLevel = logicalLine.IndentLevel
		return result
	}

	if e.lineClassifier != nil {
		fallback := e.lineClassifier.Classify(normalized)
		if !fallback.IsHeading && e.looksLikePlainHeading(normalized, e.buildContext(logicalLines, index)) {
			baseSignal.Kind = vo.SignalKindHeadingCandidate
			baseSignal.Reasons = []string{"plain-heading-candidate"}
			baseSignal.Confidence = 0.58
			return baseSignal
		}
	}

	baseSignal.Kind = vo.SignalKindBody
	baseSignal.Reasons = []string{"body"}
	baseSignal.Confidence = 1.0
	return baseSignal
}

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
