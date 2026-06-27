package signal

import (
	"regexp"

	"github.com/duke-git/lancet/v2/strutil"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/document/model/vo"
)

// appendixPattern 附录标题模式（如：附录A、附录一）
/*
  支持字母和中文数字编号
  捕获组1：完整编号（如：附录A），捕获组2：编号（如：A），捕获组3：标题文本（可选）
*/
var appendixPattern = regexp.MustCompile(`^(附录\s*([A-Za-z一二三四五六七八九十百\d]+))(?:\s+(.+))?$`)

// AppendixHeadingDetector 附录标题检测器, 检测以"附录"开头的附录标题
type AppendixHeadingDetector struct {
	BaseDetector
}

// Detect 检测附录标题, 支持格式：附录A、附录一、附录A 附录内容、附录一 附录内容
func (d *AppendixHeadingDetector) Detect(detCtx *DetectorContext, text string, opts ...DetectorOption) *vo.DocumentStructureSignal {
	if text == "" {
		return nil
	}

	matches := appendixPattern.FindStringSubmatch(text)
	if len(matches) < 3 {
		return nil
	}

	// 完整编号（如：附录A）
	nodeCode := strutil.Trim(matches[1])

	// 标题文本可选，若为空则使用编号作为标题
	title := utils.BlankToDefault(strutil.Trim(matches[3]), nodeCode)

	return &vo.DocumentStructureSignal{
		Kind:       vo.SignalKindHeading,
		NodeCode:   nodeCode,
		Title:      title,
		LevelHint:  1, // 附录标题级别为1
		Reasons:    []string{"appendix-heading"},
		Confidence: 0.92,
	}
}

func (d *AppendixHeadingDetector) Order() int {
	return 50
}
