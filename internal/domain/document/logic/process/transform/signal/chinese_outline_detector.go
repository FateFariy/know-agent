package signal

import (
	"regexp"

	"github.com/duke-git/lancet/v2/strutil"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/document/model/vo"
	"github.com/swiftbit/know-agent/internal/domain/document/support"
)

// chineseOutlinePattern 中文大纲编号模式（如：一、标题、二. 标题）
//
// 支持一~百的中文数字，后缀为顿号或点号
// 捕获组1：中文数字，捕获组2：内容
var chineseOutlinePattern = regexp.MustCompile(`^([一二三四五六七八九十百]+)[、.]\s*(.+)$`)

// ChineseOutlineDetector 中文大纲编号检测器, 检测中文数字编号（一、二、三、），通过上下文分析判断是标题还是列表项，优先级 Order=110
type ChineseOutlineDetector struct {
	BaseDetector
}

// Detect 检测中文大纲编号
//
// 通过上下文分析判断是标题还是列表项：
//  1. 相邻行有连续编号 → 列表项
//  2. 前一行以冒号结尾（列表引导语）→ 列表项
//  3. 满足标题特征（孤立、名词性、长度适中）→ 标题（置信度较低）
//  4. 其他情况 → 列表项
func (d *ChineseOutlineDetector) Detect(detCtx *DetectorContext, text string, opts ...DetectorOption) *vo.DocumentStructureSignal {
	if text == "" {
		return nil
	}

	// 匹配中文大纲编号模式
	matches := chineseOutlinePattern.FindStringSubmatch(text)
	if len(matches) < 2 {
		return nil
	}

	// 获取检测器选项
	opt := GetDetectorOptions(nil, opts...)

	// 提取标题内容
	title := strutil.Trim(matches[2])
	itemIndex := utils.ParseChineseNumber(matches[1])

	// 判断是否与相邻行形成连续序列（如：一、二、三、连续）
	sequential := d.isNeighborSequence(detCtx.LineContext, itemIndex, chineseOutline)

	// 判断前一行是否为列表引导语（以冒号结尾，如："以下是注意事项："）
	introducedByLeadIn := d.previousIntroducesList(detCtx.LineContext.PreviousNonBlank)

	// 判断是否更像标题：非序列、非引导、且满足标题特征
	headingLike := !sequential && !introducedByLeadIn && support.LooksLikePlainHeading(detCtx.LineContext, title, opt.maxPlainHeadingChars)

	// 根据判断结果确定类型、原因和置信度
	reason := utils.Ternary(headingLike, "chinese-outline-ambiguous-heading",
		utils.Ternary(sequential, "chinese-outline-sequence-list", "chinese-outline-list"))
	confidence := utils.Ternary(headingLike, 0.60,
		utils.Ternary(sequential || introducedByLeadIn, 0.92, 0.86))

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

func (d *ChineseOutlineDetector) Order() int {
	return 110
}
