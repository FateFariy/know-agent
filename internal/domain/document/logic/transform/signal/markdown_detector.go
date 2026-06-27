package signal

import (
	"regexp"
	"unicode/utf8"

	"github.com/duke-git/lancet/v2/strutil"

	"github.com/swiftbit/know-agent/internal/domain/document/model/vo"
)

// markdownHeadingPattern Markdown 标题模式（如：### 标题）
// 捕获组1：# 符号（用于确定标题级别），捕获组2：标题文本
var markdownHeadingPattern = regexp.MustCompile(`^(#+)\s+(.+)$`)

// MarkdownHeadingDetector Markdown 标题检测器, 检测以 # 开头的 Markdown 标题
type MarkdownHeadingDetector struct {
	BaseDetector
}

// Detect 检测 Markdown 标题, 支持格式：# 一级标题、## 二级标题、### 三级标题等
func (d *MarkdownHeadingDetector) Detect(detCtx *DetectorContext, text string, opts ...DetectorOption) *vo.DocumentStructureSignal {
	if text == "" {
		return nil
	}

	// 匹配 Markdown 标题模式
	matches := markdownHeadingPattern.FindStringSubmatch(text)
	if len(matches) < 2 {
		return nil
	}

	title := strutil.Trim(matches[2])

	// 如果标题与文档标题相同，标记为噪声（重复标题）
	if d.sameDocumentTitle(detCtx.DocumentTitle, title) {
		return &vo.DocumentStructureSignal{
			Kind:       vo.SignalKindNoise,
			Title:      title,
			Reasons:    []string{"duplicate-document-title"},
			Confidence: 0.99,
		}
	}

	// 从标题中提取编号代码（如：1.1、第一章）
	nodeCode := d.extractCode(title)

	return &vo.DocumentStructureSignal{
		Kind:        vo.SignalKindHeading,
		NodeCode:    nodeCode,
		Title:       title,
		LevelHint:   utf8.RuneCountInString(matches[1]), // # 的数量即为标题级别
		NumericPath: d.extractNumericPath(nodeCode),
		Reasons:     []string{"markdown-heading"},
		Confidence:  0.98,
	}
}

func (d *MarkdownHeadingDetector) Order() int {
	return 20
}
