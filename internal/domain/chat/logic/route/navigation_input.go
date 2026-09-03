package route

import (
	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/enum"
)

// NavigationInput 文档内结构路由的轻量输入
type NavigationInput struct {
	DocumentId      int64                               // 文档 ID
	Question        string                              // 原始问题
	RewriteQuestion string                              // 改写问题
	SubQuestions    []string                            // 子问题列表
	QueryType       string                              // 查询类型
	Channels        []enum.RetrievalChannel             // 检索通道
	Operations      []enum.StructureNavigationOperation // 结构导航操作
	SectionAnchors  []string                            // 显式章节锚点
	HasStructureNav bool                                // 是否高置信结构导航
}

// RouteText 拼接路由文本（原始 + 改写），供正则锚点提取使用
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

// PrimaryRetrievalIntent 获取主要检索意图
func (n *NavigationInput) PrimaryRetrievalIntent() enum.RetrievalIntent {
	if n == nil {
		return enum.RetrievalIntentGeneral
	}
	queryType := utils.BlankToDefault(n.QueryType, enum.QueryTypeDocumentQA)
	switch queryType {
	case enum.QueryTypeStructureNavigation:
		return enum.RetrievalIntentStructure
	case enum.QueryTypeTableQuery:
		return enum.RetrievalIntentTable
	case enum.QueryTypeGraphRelation:
		return enum.RetrievalIntentGraphRAG
	case enum.QueryTypeGlobalSummary:
		return enum.RetrievalIntentRaptor
	}
	// 从通道列表获取
	for _, ch := range n.Channels {
		switch ch {
		case enum.RetrievalIntentTable:
			return enum.RetrievalIntentTable
		case enum.RetrievalIntentGraphRAG:
			return enum.RetrievalIntentGraphRAG
		case enum.RetrievalIntentRaptor:
			return enum.RetrievalIntentRaptor
		case enum.RetrievalIntentStructure:
			return enum.RetrievalIntentStructure
		}
	}
	return enum.RetrievalIntentGeneral
}

// ResolveAction 解析高置信结构导航动作。
// 仅当 HasStructureNav 标志或意图为 STRUCTURE 时，按问题文本正则推导结构语法动作；
// 否则返回空串，说明不命中结构导航分支，走普通混合检索。
func (n *NavigationInput) ResolveAction() string {
	if n == nil || !n.HasStructureNav || n.QueryType != enum.QueryTypeStructureNavigation {
		return ""
	}
	contains := func(ops ...enum.StructureNavigationOperation) bool {
		return utils.ContainsAny(n.Operations, ops...)
	}
	// 目录展开
	if contains(enum.SectionWithChildren, enum.DirectChildren) {
		return enum.DocumentNavigationActionChildSectionDescend
	}
	// 相邻章节
	if contains(enum.SectionWithSiblings, enum.PreviousSibling, enum.NextSibling) {
		return enum.DocumentNavigationActionSectionAdjacencyLookup
	}
	// 父章节
	if contains(enum.ParentSection) {
		return enum.DocumentNavigationActionAncestorSectionReturn
	}
	// 当前章节
	if contains(enum.CurrentSection) {
		return enum.DocumentNavigationActionFreshTopic
	}
	return ""
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
