package signal

import (
	"regexp"
	"strings"

	"github.com/duke-git/lancet/v2/strutil"

	"github.com/swiftbit/know-agent/internal/domain/document/model/vo"
)

// decimalHeadingPattern 数字编号标题模式（如：1.1、1.1.1）
/*
  要求至少包含一个小数点
  捕获组1：编号（如：1.1.1），捕获组2：标题文本
*/
var decimalHeadingPattern = regexp.MustCompile(`^(\d+(?:\.\d+)+)\s*[、.]?\s*(.+)$`)

// DecimalHeadingDetector 数字编号标题检测器, 检测带小数点的层级编号标题（1.1、1.1.1）
type DecimalHeadingDetector struct {
	BaseDetector
}

// Detect 检测数字编号标题, 支持格式：1.1 标题、1.1.1 标题、1.1.标题
func (d *DecimalHeadingDetector) Detect(detCtx *DetectorContext, text string, opts ...DetectorOption) *vo.DocumentStructureSignal {
	if text == "" {
		return nil
	}

	// 匹配数字编号标题模式
	matches := decimalHeadingPattern.FindStringSubmatch(text)
	if len(matches) < 2 {
		return nil
	}

	nodeCode := strutil.Trim(matches[1]) // 编号（如：1.1.1）
	title := strutil.Trim(matches[2])    // 标题文本

	return &vo.DocumentStructureSignal{
		Kind:        vo.SignalKindHeading,
		NodeCode:    nodeCode,
		Title:       title,
		LevelHint:   max(len(strings.Split(nodeCode, ".")), 1), // 小数点数量+1即为层级
		NumericPath: d.extractNumericPath(nodeCode),
		Reasons:     []string{"decimal-heading"},
		Confidence:  0.95,
	}
}

func (d *DecimalHeadingDetector) Order() int {
	return 60
}
