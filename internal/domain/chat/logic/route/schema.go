package route

import (
	"github.com/swiftbit/know-agent/common/utils"
)

type navigationExtractor struct {
	text string
}

func newNavigationExtractor(text string) *navigationExtractor {
	return &navigationExtractor{text: utils.Trim(text)}
}

// collectQuotedPhrases 收集引号包裹的短语
func (n *navigationExtractor) collectQuotedPhrases() []string {
	matches := quotedTextPattern.FindAllStringSubmatch(n.text, -1)
	phrases := utils.FilterMapUniqueLimit(matches, -1, func(m []string) (string, string, bool) {
		if len(m) < 1 {
			return "", "", false
		}
		phrase := utils.Trim(m[1])
		return phrase, phrase, phrase != ""
	})

	return phrases
}

// hasExplicitSectionAnchor 判断是否有显式章节锚点
func (n *navigationExtractor) hasExplicitSectionAnchor() bool {
	return sectionCodePattern.MatchString(n.text) ||
		chineseSectionReferencePattern.MatchString(n.text) ||
		quotedTextPattern.MatchString(n.text)
}

// resolveExplicitItemIndex 解析显式条目索引
func (n *navigationExtractor) resolveExplicitItemIndex() *int {
	// 先匹配"第N步"
	stepMatches := stepReferencePattern.FindAllStringSubmatch(n.text, -1)
	if len(stepMatches) > 0 && len(stepMatches[0]) >= 2 {
		parsed := utils.ParseChineseNumber(stepMatches[0][1])
		if parsed > 0 {
			return &parsed
		}
	}

	// 再匹配"第N条/点/项/个"
	ordinalMatches := ordinalReferencePattern.FindAllStringSubmatch(n.text, -1)
	if len(ordinalMatches) > 0 && len(ordinalMatches[0]) >= 2 {
		parsed := utils.ParseChineseNumber(ordinalMatches[0][1])
		if parsed > 0 {
			return &parsed
		}
	}

	return nil
}

// detectFacet 检测问题维度
func (n *navigationExtractor) detectFacet() string {
	if sectionCodePattern.MatchString(n.text) ||
		chineseSectionReferencePattern.MatchString(n.text) ||
		quotedTextPattern.MatchString(n.text) {
		return "章节"
	}

	if stepReferencePattern.MatchString(n.text) ||
		ordinalReferencePattern.MatchString(n.text) {
		return "步骤"
	}

	return ""
}
