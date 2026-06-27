package signal

import (
	"strings"

	"github.com/duke-git/lancet/v2/convertor"
	"github.com/duke-git/lancet/v2/strutil"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/document/model/vo"
	"github.com/swiftbit/know-agent/internal/domain/document/support"
)

// 数字序列类型枚举
const (
	arabicSingle   = iota + 1 // 阿拉伯数字序列（1. 2. 3.）
	chineseOutline            // 中文大纲序列（一、二、三、）
)

// BaseDetector 基础检测器，提供共享辅助方法
type BaseDetector struct{}

// sameDocumentTitle 判断候选文本是否与文档标题相同（用于识别重复标题噪声）
func (d *BaseDetector) sameDocumentTitle(documentTitle, candidate string) bool {
	if documentTitle == "" || candidate == "" {
		return false
	}
	left := support.NormalizeComparableTitle(documentTitle)
	right := support.NormalizeComparableTitle(candidate)
	return left == right
}

// parseLooseNumber 解析松散格式的数字（支持阿拉伯数字和中文数字）
func (d *BaseDetector) parseLooseNumber(text string) int {
	normalized := strutil.Trim(text)
	if normalized == "" {
		return 0
	}

	// 优先尝试解析阿拉伯数字
	if toInt, err := convertor.ToInt(normalized); err == nil {
		return int(toInt)
	}

	// 中文数字映射表
	digitMap := map[rune]int{
		'一': 1, '二': 2, '三': 3, '四': 4, '五': 5,
		'六': 6, '七': 7, '八': 8, '九': 9, '十': 10,
	}

	runeStr := []rune(normalized)
	// 处理中文数字：十、十五、二十、二十五
	if len(runeStr) == 2 && strings.HasPrefix(normalized, "十") {
		return 10 + digitMap[runeStr[1]]
	}
	if len(runeStr) == 2 && strings.HasSuffix(normalized, "十") {
		return digitMap[runeStr[0]] * 10
	}
	if len(runeStr) == 3 && strings.Contains(normalized, "十") {
		return digitMap[runeStr[0]]*10 + digitMap[runeStr[2]]
	}

	// 单个中文数字
	return digitMap[runeStr[0]]
}

// extractNumericPath 从编号中提取数字路径（用于构建层级结构），支持格式：1.1.1 → [1,1,1]；第一章 → [1]
func (d *BaseDetector) extractNumericPath(code string) []int {
	normalized := strutil.Trim(code)
	if normalized == "" {
		return nil
	}

	// 优先处理小数点分隔的编号
	var path []int
	for _, segment := range strings.Split(normalized, ".") {
		if toInt, err := convertor.ToInt(segment); err == nil {
			path = append(path, int(toInt))
		} else {
			return nil
		}
	}

	// 处理章节格式（如：第一章）
	if matches := chapterPattern.FindStringSubmatch(normalized + " 标题"); len(matches) >= 3 {
		chapterNo := d.parseLooseNumber(matches[2])
		if chapterNo > 0 {
			return []int{chapterNo}
		}
	}
	return path
}

// extractCode 从标题中提取编号代码, 数字编号（1.1）→ 章节编号（第一章）→ 附录编号（附录A）
func (d *BaseDetector) extractCode(title string) string {
	// 匹配数字编号标题（如：1.1 标题）
	if matches := decimalHeadingPattern.FindStringSubmatch(title); len(matches) > 1 {
		return strutil.Trim(matches[1])
	}

	// 匹配章节编号（如：第一章）
	if matches := chapterPattern.FindStringSubmatch(title); len(matches) > 1 {
		return strutil.Trim(matches[1])
	}

	// 匹配附录编号（如：附录A）
	if matches := appendixPattern.FindStringSubmatch(title); len(matches) > 1 {
		return strutil.Trim(matches[1])
	}

	return ""
}

// isNeighborSequence 判断当前编号是否与相邻行形成连续序列, 通过检查前一行或后一行的编号是否与当前编号相差1来判断
func (d *BaseDetector) isNeighborSequence(lineContext *vo.LineContext, itemIndex, family int) bool {
	if itemIndex == 0 || family == 0 {
		return false
	}
	return d.isSequenceNeighbor(lineContext.PreviousNonBlank, itemIndex, family, -1) ||
		d.isSequenceNeighbor(lineContext.NextNonBlank, itemIndex, family, 1)
}

// isSequenceNeighbor 判断候选行是否与当前行形成指定偏移的序列
func (d *BaseDetector) isSequenceNeighbor(candidate *vo.DocumentStructureLogicalLine, itemIndex, family, offset int) bool {
	if candidate == nil {
		return false
	}

	normalized := strutil.Trim(candidate.NormalizedText)
	if normalized == "" {
		return false
	}

	var candidateIndex *int
	switch family {
	case arabicSingle:
		// 阿拉伯数字序列（如：1. 2. 3.）
		if matches := singleLevelDigitPattern.FindStringSubmatch(normalized); len(matches) >= 2 {
			candidateIndex = utils.Pointer(d.parseLooseNumber(matches[1]))
		}
	case chineseOutline:
		// 中文大纲序列（如：一、二、三、）
		if matches := chineseOutlinePattern.FindStringSubmatch(normalized); len(matches) >= 2 {
			candidateIndex = utils.Pointer(d.parseLooseNumber(matches[1]))
		}
	}

	return candidateIndex != nil && *candidateIndex == itemIndex+offset
}

// previousIntroducesList 判断前一行是否为列表引导语（以冒号结尾）, 例如："以下是注意事项：" 后面通常紧跟列表项
func (d *BaseDetector) previousIntroducesList(previousNonBlank *vo.DocumentStructureLogicalLine) bool {
	if previousNonBlank == nil {
		return false
	}
	previous := strutil.Trim(previousNonBlank.NormalizedText)
	return strings.HasSuffix(previous, "：") || strings.HasSuffix(previous, ":")
}
