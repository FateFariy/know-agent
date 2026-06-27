package signal

import (
	"regexp"

	"github.com/duke-git/lancet/v2/strutil"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/document/model/vo"
)

// chapterPattern 章节标题模式（如：第一章、第一节、第二条）
/*
  支持后缀：章、节、条、部、分
  捕获组1：完整编号（如：第一章），捕获组2：数字（如：一），捕获组3：标题文本
*/
var chapterPattern = regexp.MustCompile(`^(第([一二三四五六七八九十百\d]+)[章节条部分])\s*(.+)$`)

// ChapterHeadingDetector 章节标题检测器, 检测章节标题（如：第一章、第一节、第二条、第三部分）
type ChapterHeadingDetector struct {
	BaseDetector
}

// Detect 检测章节标题, 支持格式：第一章、第一节、第二条、第三部分
func (d *ChapterHeadingDetector) Detect(detCtx *DetectorContext, text string, opts ...DetectorOption) *vo.DocumentStructureSignal {
	if text == "" {
		return nil
	}

	matches := chapterPattern.FindStringSubmatch(text)
	if len(matches) < 3 {
		return nil
	}

	nodeCode := strutil.Trim(matches[1]) // 完整编号（如：第一章）
	title := strutil.Trim(matches[3])    // 标题文本

	// 如果标题与文档标题相同，标记为噪声
	if d.sameDocumentTitle(detCtx.DocumentTitle, title) {
		return &vo.DocumentStructureSignal{
			Kind:       vo.SignalKindNoise,
			NodeCode:   nodeCode,
			Title:      title,
			Reasons:    []string{"duplicate-document-title"},
			Confidence: 0.99,
		}
	}

	// 解析章节编号（支持中文数字）
	chapterNo := d.parseLooseNumber(matches[2])

	return &vo.DocumentStructureSignal{
		Kind:        vo.SignalKindHeading,
		NodeCode:    nodeCode,
		Title:       title,
		LevelHint:   1, // 章节标题级别为1
		NumericPath: utils.Ternary(chapterNo > 0, []int{chapterNo}, nil),
		Reasons:     []string{"chapter-heading"},
		Confidence:  0.96,
	}
}

func (d *ChapterHeadingDetector) Order() int {
	return 40
}
