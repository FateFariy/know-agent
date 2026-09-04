package route

import (
	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/enum"
)

// NavigationInput 文档内结构路由输入
type NavigationInput struct {
	DocumentId       int64                         // 文档 ID
	Question         string                        // 原始问题
	RewriteQuestion  string                        // 改写问题
	SectionAnchors   []string                      // 显式章节锚点
	NavigationAction enum.DocumentNavigationAction // 导航动作
	ItemIndex        *int                          //条目索引（第N步/条/点/项）
}

// Normalize 执行归一化：默认值填充、文本信号探测、冲突合并
func (n *NavigationInput) Normalize() {
	if n == nil {
		return
	}

	// 基础字段兜底
	n.RewriteQuestion = utils.BlankToDefault(n.RewriteQuestion, n.Question)

	// 文本信号探测（兜底来源）
	routeText := n.RouteText()
	extractor := newNavigationExtractor(routeText)
	signals := extractor.detectNavigationTextSignals()
	textItemIndex := extractor.resolveExplicitItemIndex()

	// 冲突合并：自身显式字段优先，缺省项由文本信号补全
	n.mergeNavigationSignals(signals, textItemIndex)
}

// mergeNavigationSignals 合并导航信号，处理动作冲突。
//
// 信号优先级：Agent 显式提供的 NavigationAction 最高，其次自身结构导航声明
func (n *NavigationInput) mergeNavigationSignals(signals *NavigationSignals, textItemIndex *int) {
	// Agent 显式提供导航动作：动作即最高级信号，直接按其校正结构导航声明
	if n.NavigationAction != "" {
		// 结构导航类动作：强制结构导航声明与查询类型
		if n.IsStructureNavigation() {
			if len(n.SectionAnchors) == 0 {
				n.SectionAnchors = append([]string{}, signals.SectionAnchors...)
			}
			return
		}

		if n.NavigationAction == enum.DocumentNavigationActionItemReference {
			n.ItemIndex = textItemIndex
		}
		return
	}
	n.SectionAnchors = append([]string{}, signals.SectionAnchors...)

	// 结构导航：文本语义动作优先，缺省用兜底动作
	n.NavigationAction = utils.BlankToDefault(signals.Action, enum.DocumentNavigationActionFreshTopic)

	// 普通问题：文本显式条目索引兜底
	n.ItemIndex = textItemIndex
	n.NavigationAction = enum.DocumentNavigationActionItemReference
}

// RouteText 拼接路由文本（原始 + 改写），供章节定位匹配使用
func (n *NavigationInput) RouteText() string {
	if n == nil {
		return ""
	}
	text := n.Question
	if n.RewriteQuestion != "" && n.RewriteQuestion != n.Question {
		text += " " + n.RewriteQuestion
	}
	return text
}

// HasSectionAnchor 判断是否携带非空显式章节锚点
func (n *NavigationInput) HasSectionAnchor() bool {
	if n == nil {
		return false
	}
	for _, anchor := range n.SectionAnchors {
		if utils.IsNotBlank(anchor) {
			return true
		}
	}
	return false
}

// IsStructureNavigation 是否结构导航类动作（走结构树确定性查询）
func (n *NavigationInput) IsStructureNavigation() bool {
	if n == nil {
		return false
	}
	switch n.NavigationAction {
	case enum.DocumentNavigationActionChildSectionDescend,
		enum.DocumentNavigationActionSectionAdjacencyLookup,
		enum.DocumentNavigationActionAncestorSectionReturn,
		enum.DocumentNavigationActionSiblingSectionSwitch,
		enum.DocumentNavigationActionTopicSwitch,
		enum.DocumentNavigationActionTopicContinue:
		return true
	}
	return false
}
