package signal

import (
	"regexp"
	"strings"

	"github.com/duke-git/lancet/v2/convertor"
	"github.com/duke-git/lancet/v2/strutil"

	"github.com/swiftbit/know-agent/internal/domain/document/model/vo"
)

var (
	titleHashPrefixRegex = regexp.MustCompile(`^#+\s*`)
	titleExtRegex        = regexp.MustCompile(`\.[A-Za-z0-9]{1,6}$`)
	titleSpaceRegex      = regexp.MustCompile(`\s+`)
	decimalPattern       = regexp.MustCompile(`^(\d+(?:\.\d+)+)\s*[、.]?\s*(.+)$`)
)

const (
	arabicSingle = iota
	chineseOutline
)

type BaseDetector struct{}

func (d *BaseDetector) sameDocumentTitle(documentTitle, candidate string) bool {
	if documentTitle == "" || candidate == "" {
		return false
	}
	left := d.normalizeComparableTitle(documentTitle)
	right := d.normalizeComparableTitle(candidate)
	return left == right
}

func (d *BaseDetector) normalizeComparableTitle(text string) string {
	normalized := strutil.Trim(text)
	if normalized == "" {
		return ""
	}
	normalized = titleHashPrefixRegex.ReplaceAllString(normalized, "")
	normalized = titleExtRegex.ReplaceAllString(normalized, "")
	normalized = titleSpaceRegex.ReplaceAllString(normalized, "")
	return strings.ToLower(normalized)
}

func (d *BaseDetector) parseLooseNumber(text string) int {
	normalized := strutil.Trim(text)
	if normalized == "" {
		return 0
	}

	if toInt, err := convertor.ToInt(normalized); err == nil {
		return int(toInt)
	}

	digitMap := map[rune]int{
		'一': 1, '二': 2, '三': 3, '四': 4, '五': 5,
		'六': 6, '七': 7, '八': 8, '九': 9, '十': 10,
	}

	if val, ok := digitMap[rune(normalized[0])]; ok {
		return val
	}
	if len(normalized) == 2 && strings.HasPrefix(normalized, "十") {
		return 10 + digitMap[rune(normalized[1])]
	}
	if len(normalized) == 2 && strings.HasSuffix(normalized, "十") {
		return digitMap[rune(normalized[0])] * 10
	}
	if len(normalized) == 3 && strings.Contains(normalized, "十") {
		return digitMap[rune(normalized[0])]*10 + digitMap[rune(normalized[2])]
	}

	return 0
}

func (d *BaseDetector) extractNumericPath(code string) []int {
	normalized := strutil.Trim(code)
	if normalized == "" {
		return nil
	}

	var path []int
	for _, segment := range strings.Split(normalized, ".") {
		if toInt, err := convertor.ToInt(segment); err == nil {
			path = append(path, int(toInt))
		} else {
			return nil
		}
	}
	if matches := chapterPattern.FindStringSubmatch(normalized + " 标题"); len(matches) >= 3 {
		chapterNo := d.parseLooseNumber(matches[2])
		if chapterNo > 0 {
			return []int{chapterNo}
		}
	}
	return path
}

func (d *BaseDetector) extractCode(title string) string {
	if matches := decimalPattern.FindStringSubmatch(title); len(matches) == 3 {
		return strutil.Trim(matches[1])
	}
	return ""
}

func (d *BaseDetector) isNeighborSequence(itemIndex *int, family int, context *lineContext) bool {
	if itemIndex == nil {
		return false
	}
	return d.isSequenceNeighbor(context.previousNonBlank, *itemIndex, family, -1) ||
		d.isSequenceNeighbor(context.nextNonBlank, *itemIndex, family, 1)
}

func (d *BaseDetector) isSequenceNeighbor(candidate *vo.DocumentStructureLogicalLine, itemIndex, family, offset int) bool {
	if candidate == nil {
		return false
	}
	candidateIndex := d.resolveOrderedIndex(candidate.NormalizedText, family)
	return candidateIndex != nil && *candidateIndex == itemIndex+offset
}

func (d *BaseDetector) resolveOrderedIndex(text string, family int) *int {
	normalized, := strutil.Trim(text)
	if normalized == "" {
		return nil
	}

	switch family {
	case arabicSingle:
		if matches := singleLevelDigitPattern.FindStringSubmatch(normalized); len(matches) == 3 {
			return d.parseLooseNumber(matches[1])
		}
	case chineseOutline:
		if matches := chineseOutlinePattern.FindStringSubmatch(normalized); len(matches) == 3 {
			return d.parseLooseNumber(matches[1])
		}
	}
	return nil
}
