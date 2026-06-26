package signal

import (
	"regexp"

	"github.com/duke-git/lancet/v2/strutil"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/document/model/vo"
)

var chapterPattern = regexp.MustCompile(`^(第([一二三四五六七八九十百\d]+)[章节条部分])\s*(.+)$`)

type ChapterHeadingDetector struct {
	BaseDetector
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
	return &vo.DocumentStructureSignal{
		Kind:        vo.SignalKindHeading,
		NodeCode:    nodeCode,
		Title:       title,
		LevelHint:   1,
		NumericPath: utils.Ternary(chapterNo > 0, []int{chapterNo}, nil),
		Reasons:     []string{"chapter-heading"},
		Confidence:  0.96,
	}
}

func (d *ChapterHeadingDetector) Order() int {
	return 40
}
