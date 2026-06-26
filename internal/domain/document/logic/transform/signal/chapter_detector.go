package signal

import (
	"regexp"
	"strings"

	"github.com/duke-git/lancet/v2/convertor"
	"github.com/duke-git/lancet/v2/strutil"

	"github.com/swiftbit/know-agent/internal/domain/document/model/vo"
)

var chapterPattern = regexp.MustCompile(`^(第([一二三四五六七八九十百\d]+)[章节条部分])\s*(.+)$`)

type ChapterHeadingDetector struct{}

func (d *ChapterHeadingDetector) Name() string {
	return "chapter-heading"
}

func (d *ChapterHeadingDetector) Order() int {
	return 40
}

func (d *ChapterHeadingDetector) Detect(detCtx *DetectorContext, text string) *vo.DocumentStructureSignal {
	if text == "" {
		return nil
	}

	matches := chapterPattern.FindStringSubmatch(text)
	if len(matches) != 4 {
		return nil
	}

	nodeCode := strutil.Trim(matches[1])
	title := strutil.Trim(matches[3])

	if d.sameDocumentTitle(detCtx.DocumentTitle, title) {
		return &vo.DocumentStructureSignal{
			Kind:       vo.SignalKindNoise,
			NodeCode:   nodeCode,
			Title:      title,
			Reasons:    []string{"duplicate-document-title"},
			Confidence: 0.99,
		}
	}

	chapterNo := d.parseLooseNumber(matches[2])
	var numericPath []int
	if chapterNo > 0 {
		numericPath = []int{chapterNo}
	}

	return &vo.DocumentStructureSignal{
		Kind:        vo.SignalKindHeading,
		NodeCode:    nodeCode,
		Title:       title,
		LevelHint:   1,
		NumericPath: numericPath,
		Reasons:     []string{"chapter-heading"},
		Confidence:  0.96,
	}
}

func (d *ChapterHeadingDetector) sameDocumentTitle(documentTitle, candidate string) bool {
	if documentTitle == "" || candidate == "" {
		return false
	}
	left := d.normalizeComparableTitle(documentTitle)
	right := d.normalizeComparableTitle(candidate)
	return left == right
}

func (d *ChapterHeadingDetector) normalizeComparableTitle(text string) string {
	normalized := strutil.Trim(text)
	if normalized == "" {
		return ""
	}
	normalized = regexp.MustCompile(`^#+\s*`).ReplaceAllString(normalized, "")
	normalized = regexp.MustCompile(`\.[A-Za-z0-9]{1,6}$`).ReplaceAllString(normalized, "")
	normalized = regexp.MustCompile(`\s+`).ReplaceAllString(normalized, "")
	return strings.ToLower(normalized)
}

func (d *ChapterHeadingDetector) parseLooseNumber(text string) int {
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
