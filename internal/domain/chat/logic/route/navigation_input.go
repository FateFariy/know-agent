package route

import (
	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/enum"
)

// NavigationInput 文档内结构路由输入
type NavigationInput struct {
	DocumentId       int64                         // 文档 ID
	QueryType        string                        // 查询类型
	Question         string                        // 原始问题
	RewriteQuestion  string                        // 改写问题
	SubQuestions     []string                      // 子问题列表
	SectionAnchors   []string                      // 显式章节锚点
	HasStructureNav  bool                          // 是否高置信结构导航
	NavigationAction enum.DocumentNavigationAction // 导航动作
	ItemIndex        *int                          //条目索引（第N步/条/点/项）
}

// Normalize 执行归一化：默认值填充、文本信号探测、冲突合并、最终动作解析
func (n *NavigationInput) Normalize() {
	if n == nil {
		return
	}

	// 基础字段兜底
	n.RewriteQuestion = utils.BlankToDefault(n.RewriteQuestion, n.Question)
	n.QueryType = utils.BlankToDefault(n.QueryType, enum.QueryTypeDocumentQA)
	if len(n.SubQuestions) == 0 {
		n.SubQuestions = []string{n.RewriteQuestion}
	}

	// 文本信号探测（兜底来源）
	routeText := n.RouteText()
	extractor := newNavigationExtractor(routeText)
	signals := extractor.detectNavigationTextSignals()
	textItemIndex := extractor.resolveExplicitItemIndex()

	// 冲突合并：自身显式字段优先，缺省项由文本信号补全
	n.mergeNavigationSignals(signals, textItemIndex)

	// 最终动作解析
	n.NavigationAction = n.resolveFinalAction(signals)
}

// mergeNavigationSignals 合并导航信号，处理动作冲突。
//
// 信号优先级：Agent 显式提供的 NavigationAction 最高，其次自身结构导航声明，
// 文本信号仅作缺失字段的兜底来源。冲突时（动作与 HasStructureNav/QueryType 矛盾），以显式动作为准校正其余字段。
func (n *NavigationInput) mergeNavigationSignals(signals *NavigationSignals, textItemIndex *int) {
	// Agent 显式提供导航动作：动作即最高级信号，直接按其校正结构导航声明
	if n.NavigationAction != "" {
		n.applyActionConsistency(signals, textItemIndex)
		return
	}
	// 自身已声明结构导航：补齐缺失的 QueryType / 章节锚点
	if n.HasStructureNav {
		if n.QueryType == enum.QueryTypeDocumentQA {
			n.QueryType = enum.QueryTypeStructureNavigation
		}
		if len(n.SectionAnchors) == 0 && signals.HasStructureNav {
			n.SectionAnchors = append([]string{}, signals.SectionAnchors...)
		}
		return
	}

	// 自身未声明结构导航：仅当文本命中结构导航语义时升级
	if signals.HasStructureNav {
		n.HasStructureNav = true
		if n.QueryType == enum.QueryTypeDocumentQA {
			n.QueryType = enum.QueryTypeStructureNavigation
		}
		if len(n.SectionAnchors) == 0 {
			n.SectionAnchors = append([]string{}, signals.SectionAnchors...)
		}
		return
	}

	// 普通问题：文本显式条目索引兜底
	if n.ItemIndex == nil && len(n.SubQuestions) == 1 {
		n.ItemIndex = textItemIndex
	}
}

// applyActionConsistency 依据 Agent 显式动作校正结构导航声明、QueryType、锚点与条目索引
func (n *NavigationInput) applyActionConsistency(signals *NavigationSignals, textItemIndex *int) {
	// 结构导航类动作：强制结构导航声明与查询类型
	if n.IsStructureNavigation() {
		n.HasStructureNav = true
		n.QueryType = enum.QueryTypeStructureNavigation
		if len(n.SectionAnchors) == 0 && signals.HasStructureNav {
			n.SectionAnchors = append([]string{}, signals.SectionAnchors...)
		}
		return
	}

	// 非结构导航类动作（FRESH_TOPIC / ITEM_REFERENCE）：降级为普通文档问题
	n.HasStructureNav = false
	if n.QueryType == enum.QueryTypeStructureNavigation {
		n.QueryType = enum.QueryTypeDocumentQA
	}
	// ITEM_REFERENCE 缺条目索引时用文本索引兜底
	if n.NavigationAction == enum.DocumentNavigationActionItemReference && n.ItemIndex == nil {
		n.ItemIndex = textItemIndex
	}
}

// resolveFinalAction 解析最终导航动作
func (n *NavigationInput) resolveFinalAction(signals *NavigationSignals) enum.DocumentNavigationAction {
	// 显式动作优先
	if n.NavigationAction != "" {
		return n.NavigationAction
	}
	// 结构导航：文本语义动作优先，缺省用兜底动作
	if n.HasStructureNav {
		if signals.Action != "" {
			return signals.Action
		}
		return enum.DocumentNavigationActionChildSectionDescend
	}

	// 普通问题：条目引用或新主题
	if n.ItemIndex != nil {
		return enum.DocumentNavigationActionItemReference
	}
	return enum.DocumentNavigationActionFreshTopic
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

// RetrievalIntent 获取主要检索意图
func (n *NavigationInput) RetrievalIntent() enum.RetrievalIntent {
	if n == nil {
		return enum.RetrievalIntentGeneral
	}
	switch n.QueryType {
	case enum.QueryTypeStructureNavigation:
		return enum.RetrievalIntentStructure
	case enum.QueryTypeTableQuery:
		return enum.RetrievalIntentTable
	case enum.QueryTypeGraphRelation:
		return enum.RetrievalIntentGraphRAG
	case enum.QueryTypeGlobalSummary:
		return enum.RetrievalIntentRaptor
	}
	return enum.RetrievalIntentGeneral
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
