package route

import (
	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/enum"
)

// 相邻章节意图关键词
var adjacentSectionHints = []string{
	"上一节", "下一节", "前一节", "后一节", "上一章", "下一章",
	"相邻章节", "同一一级章节",
}

// 章节位置意图关键词
var sectionLocationHints = []string{
	"属于哪个章节", "哪个章节", "哪个小节", "哪一节", "哪一章", "章节位置",
}

// 目录/大纲意图关键词
var outlineSectionHints = []string{
	"包含哪些章节", "都包含哪些章节", "有哪些章节", "有哪些小节",
	"包含哪些小节", "章节列表", "小节列表", "子章节", "子小节",
	"下级章节", "展开目录", "列出目录",
}

// navigationTextSignals 由问题文本规则兜底判定的结构导航信号
type navigationTextSignals struct {
	hasStructureNav bool     // 文本命中结构导航语义（相邻/大纲/位置+锚点）
	action          string   // 推导的结构导航动作（语法正则解析不到时的语义兜底）
	sectionAnchors  []string // 提取的显式章节锚点（编号 / 第N章节 / 引号标题）
}

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

// detectNavigationTextSignals 依据问题文本做确定性结构导航判定
//
// 判定语义迁移自意图识别下线前的 DeterministicFallbackRecognizer.navigationQuestion：
//   - 目录/大纲询问（包含哪些章节…）→ 大纲导航 → CHILD_SECTION_DESCEND；
//   - 相邻章节（上一节/下一节…）→ 相邻导航 → SECTION_ADJACENCY_LOOKUP；
//   - 「位置询问 + 显式锚点」→ 父章节定位 → ANCESTOR_SECTION_RETURN。
func (n *navigationExtractor) detectNavigationTextSignals() *navigationTextSignals {
	signals := &navigationTextSignals{}
	normalized := utils.Trim(n.text)
	if normalized == "" {
		return signals
	}
	hasExplicitAnchor := n.hasExplicitSectionAnchor()
	adjacent := utils.ContainsAnyString(normalized, adjacentSectionHints)
	location := utils.ContainsAnyString(normalized, sectionLocationHints)
	outline := utils.ContainsAnyString(normalized, outlineSectionHints)

	switch {
	case outline:
		signals.hasStructureNav = true
		signals.action = enum.DocumentNavigationActionChildSectionDescend
	case adjacent:
		signals.hasStructureNav = true
		signals.action = enum.DocumentNavigationActionSectionAdjacencyLookup
	case location && hasExplicitAnchor:
		signals.hasStructureNav = true
		signals.action = enum.DocumentNavigationActionAncestorSectionReturn
	}
	if signals.hasStructureNav {
		signals.sectionAnchors = n.extractSectionAnchors()
	}
	return signals
}

// extractSectionAnchors 从问题文本提取显式章节锚点（编号 / 第N章节 / 引号标题），去重限 8。
func (n *navigationExtractor) extractSectionAnchors() []string {
	text := n.text
	candidates := append([]string{}, sectionCodePattern.FindAllString(text, -1)...)
	candidates = append(candidates, chineseSectionReferencePattern.FindAllString(text, -1)...)
	candidates = append(candidates, n.collectQuotedPhrases()...)
	return utils.FilterMapUniqueLimit(candidates, 8, func(anchor string) (string, string, bool) {
		trimmed := utils.Trim(anchor)
		return trimmed, trimmed, trimmed != ""
	})
}
