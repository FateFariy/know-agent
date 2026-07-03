package signal

import (
	"regexp"

	"github.com/duke-git/lancet/v2/strutil"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/document/model/vo"
	"github.com/swiftbit/know-agent/internal/domain/document/support"
)

// singleLevelDigitPattern 单层数字编号模式（如：1. 标题、1、标题）
/*
  注意：带小数点的多层编号（1.1、1.1.1）由 DecimalHeadingDetector 处理
  捕获组1：数字，捕获组2：内容
*/
var singleLevelDigitPattern = regexp.MustCompile(`^(\d+)\s*[、.]\s*(.+)$`)

// SingleLevelDigitDetector 单层数字编号检测器, 检测单层数字编号（1.、1、），通过上下文分析判断是标题还是列表项，优先级 Order=100
type SingleLevelDigitDetector struct {
	BaseDetector
}

// Detect 检测单层数字编号
/*
 通过上下文分析判断是标题还是列表项：
 1. 相邻行有连续编号 → 列表项
 2. 前一行以冒号结尾（列表引导语）→ 列表项
 3. 满足标题特征（孤立、名词性、长度适中）→ 标题（置信度较低）
 4. 其他情况 → 列表项
*/
func (d *SingleLevelDigitDetector) Detect(detCtx *DetectorContext, text string, opts ...DetectorOption) *vo.DocumentStructureSignal {
	if text == "" {
		return nil
	}

	// 匹配单层数字编号模式
	matches := singleLevelDigitPattern.FindStringSubmatch(text)
	if len(matches) < 2 {
		return nil
	}

	// 获取检测器选项
	opt := GetDetectorOptions(nil, opts...)

	// 提取标题内容
	title := strutil.Trim(matches[2])
	itemIndex := utils.ParseChineseNumber(matches[1])

	// 判断是否与相邻行形成连续序列（如：1. 2. 3. 连续）
	sequential := d.isNeighborSequence(detCtx.LineContext, itemIndex, arabicSingle)

	// 判断前一行是否为列表引导语（以冒号结尾，如："以下是注意事项："）
	introducedByLeadIn := d.previousIntroducesList(detCtx.LineContext.PreviousNonBlank)

	// 判断是否更像标题：非序列、非引导、且满足标题特征
	headingLike := !sequential && !introducedByLeadIn && support.LooksLikePlainHeading(detCtx.LineContext, title, opt.maxPlainHeadingChars)

	// 根据判断结果确定类型、原因和置信度
	reason := utils.Ternary(headingLike, "single-digit-ambiguous-heading",
		utils.Ternary(sequential, "single-digit-sequence-list", "single-digit-list"))
	confidence := utils.Ternary(headingLike, 0.62,
		utils.Ternary(sequential || introducedByLeadIn, 0.93, 0.88))

	return &vo.DocumentStructureSignal{
		Kind:        utils.Ternary(headingLike, vo.SignalKindHeading, vo.SignalKindListItem),
		NodeCode:    strutil.Trim(matches[1]),
		Title:       title,
		ItemIndex:   itemIndex,
		LevelHint:   utils.Ternary(headingLike, 1, 0),
		NumericPath: utils.Ternary(headingLike && itemIndex > 0, []int{itemIndex}, nil),
		Reasons:     []string{reason},
		Confidence:  confidence,
	}
}

// Order 返回检测器优先级（100）
func (d *SingleLevelDigitDetector) Order() int {
	return 100
}
