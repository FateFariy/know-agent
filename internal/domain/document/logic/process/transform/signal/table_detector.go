package signal

import (
	"regexp"
	"strings"

	"github.com/swiftbit/know-agent/internal/domain/document/model/vo"
)

// 表格行检测相关正则表达式
var (
	tableSplitPattern  = regexp.MustCompile(`\|`)           // 竖线分隔符（用于分割表格单元格）
	specialLinePattern = regexp.MustCompile(`^[:\-\\s|]+$`) // 表格分割线（如：|---|----|）
)

// TableRowDetector 表格行检测器, 检测表格行内容，包括 Markdown 表格和制表符分隔的表格
type TableRowDetector struct{}

// Detect 检测表格行, 通过多种特征判断：竖线包围、制表符、多列分割、表格分割线
func (d *TableRowDetector) Detect(detCtx *DetectorContext, text string, opts ...DetectorOption) *vo.DocumentStructureSignal {
	if text == "" {
		return nil
	}

	if d.isTableRow(text) {
		return &vo.DocumentStructureSignal{
			Kind:       vo.SignalKindTableRow,
			Title:      text,
			Reasons:    []string{"table-row"},
			Confidence: 0.90,
		}
	}

	return nil
}

// isTableRow 判断是否为表格行
//
// 判断条件（满足任一）：
//  1. 以 | 开头且以 | 结尾（Markdown 表格行）
//  2. 包含制表符 \t（制表符分隔表格）
//  3. 用 | 分割后至少有3列（多列表格）
//  4. 符合表格分割线模式（如：|---|----|）
func (d *TableRowDetector) isTableRow(text string) bool {
	if strings.HasPrefix(text, "|") && strings.HasSuffix(text, "|") {
		return true
	}
	if strings.Contains(text, "\t") {
		return true
	}
	if len(tableSplitPattern.Split(text, -1)) >= 3 && strings.Contains(text, "|") {
		return true
	}
	return specialLinePattern.MatchString(text)
}

func (d *TableRowDetector) Order() int {
	return 70
}
