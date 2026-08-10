package markdown

import (
	"errors"
	"regexp"
	"sort"
	"strings"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/document/model/shared"
)

var re = regexp.MustCompile(`\r\n|\n|\r`)

type sourceIndex struct {
	sourceText  string
	lineStarts  []int
	byteOffsets []int
}

func newSourceIndex(sourceText string) *sourceIndex {
	si := &sourceIndex{sourceText: sourceText}
	si.buildLineStarts()
	si.buildByteOffsets()
	return si
}

func (s *sourceIndex) buildLineStarts() {
	matches := re.FindAllStringIndex(s.sourceText, -1)
	s.lineStarts = []int{0}
	for _, m := range matches {
		s.lineStarts = append(s.lineStarts, m[1])
	}
}

func (s *sourceIndex) buildByteOffsets() {
	s.byteOffsets = []int{0}
	total := 0
	for _, r := range s.sourceText {
		total += len(string(r))
		s.byteOffsets = append(s.byteOffsets, total)
	}
}

func (s *sourceIndex) CharacterSpan(lineMap []int) (int, int, error) {
	if len(lineMap) != 2 {
		return 0, 0, errors.New("invalid line map: must have 2 elements")
	}
	startLine, endLine := lineMap[0], lineMap[1]
	if startLine < 0 || endLine < startLine {
		return 0, 0, errors.New("invalid markdown token line map")
	}
	start := s.lineStarts[startLine]
	if startLine >= len(s.lineStarts) {
		start = len([]rune(s.sourceText))
	}
	end := len([]rune(s.sourceText))
	if endLine < len(s.lineStarts) {
		end = s.lineStarts[endLine]
	}
	return start, end, nil
}

func (s *sourceIndex) SourceSpan(charSpan [2]int) *shared.SourceSpan {
	start, end := charSpan[0], charSpan[1]
	startLine, startCol := s.LineColumn(start)
	endLine, endCol := s.LineColumn(end)
	return &shared.SourceSpan{
		StartByte:   s.byteOffsets[start],
		EndByte:     s.byteOffsets[end],
		StartLine:   startLine,
		StartColumn: startCol,
		EndLine:     endLine,
		EndColumn:   endCol,
	}
}

func (s *sourceIndex) LineColumn(charOffset int) (int, int) {
	idx := sort.Search(len(s.lineStarts), func(i int) bool {
		return s.lineStarts[i] > charOffset
	})
	lineIndex := idx - 1
	if lineIndex < 0 {
		lineIndex = 0
	}
	return lineIndex + 1, charOffset - s.lineStarts[lineIndex] + 1
}

func (s *sourceIndex) TableCellSpan(rowCharSpan [2]int, columnIndex int) (int, int) {
	runes := []rune(s.sourceText)
	rowStart, rowEnd := rowCharSpan[0], rowCharSpan[1]
	rowText := string(runes[rowStart:rowEnd])

	rowText = strings.TrimSuffix(rowText, "\r\n")
	rowText = strings.TrimSuffix(rowText, "\n")
	rowText = strings.TrimSuffix(rowText, "\r")

	leftTrimmed := strings.TrimLeft(rowText, " ")
	contentStart := rowStart + utils.Len(rowText) - utils.Len(leftTrimmed)
	content := strings.TrimRight(leftTrimmed, " ")

	var segments [][2]int
	segStart := 0
	escaped := false
	contentRunes := []rune(content)
	for i, r := range contentRunes {
		if r == '|' && !escaped {
			segments = append(segments, [2]int{segStart, i})
			segStart = i + 1
		}
		escaped = r == '\\'
	}
	segments = append(segments, [2]int{segStart, len(contentRunes)})

	if len(segments) > 0 && segments[0][0] == 0 && segments[0][1] == 0 {
		segments = segments[1:]
	}
	if len(segments) > 0 && len(segments)-1 < len(segments) && segments[len(segments)-1][0] == len(contentRunes) {
		segments = segments[:len(segments)-1]
	}

	if columnIndex >= len(segments) {
		return contentStart, contentStart
	}
	start, end := segments[columnIndex][0], segments[columnIndex][1]
	cellRunes := contentRunes[start:end]
	trimL := len(cellRunes) - utils.Len(strings.TrimLeft(string(cellRunes), " "))
	start += trimL
	end = start + utils.Len(strings.TrimRight(string(cellRunes[trimL:]), " "))
	return contentStart + start, contentStart + end
}
