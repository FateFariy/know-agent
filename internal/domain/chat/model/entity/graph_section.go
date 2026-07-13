package entity

import (
	"github.com/duke-git/lancet/v2/strutil"

	"github.com/swiftbit/know-agent/common/utils"
)

// GraphSection 结构图节点基本信息
type GraphSection struct {
	NodeId            int64  `json:"nodeId"`
	DocumentId        int64  `json:"documentId"`
	ParseTaskId       int64  `json:"parseTaskId"`
	NodeNo            int    `json:"nodeNo"`
	Depth             int    `json:"depth"`
	ParentNodeId      int64  `json:"parentId"`
	PrevSiblingNodeId int64  `json:"prevSiblingId"`
	NextSiblingNodeId int64  `json:"nextSiblingId"`
	NodeCode          string `json:"nodeCode"`
	Title             string `json:"title"`
	AnchorText        string `json:"anchorText"`
	SectionPath       string `json:"sectionPath"`
	CanonicalPath     string `json:"canonicalPath"`
	ContentText       string `json:"contentText"`
}

// DisplayTitle 返回节点的展示标题
func (s *GraphSection) DisplayTitle() string {
	if strutil.IsNotBlank(s.CanonicalPath) {
		return strutil.Trim(s.SectionPath)
	}
	if strutil.IsNotBlank(s.NodeCode) && strutil.IsNotBlank(s.Title) {
		return strutil.Trim(s.NodeCode + " " + s.Title)
	}
	return s.Title
}

// GraphItem 结构图编号项
type GraphItem struct {
	NodeId            int64  // 节点ID
	DocumentId        int64  // 文档ID
	ParseTaskId       int64  // 解析任务ID
	NodeNo            int    // 节点编号
	NodeType          string // 节点类型
	SectionNodeId     int64  // 节点所属章节ID
	PrevSiblingNodeId int64  // 前一个兄弟节点ID
	NextSiblingNodeId int64  // 后一个兄弟节点ID
	Title             string // 节点标题
	AnchorText        string // 节点锚点文本
	SectionPath       string // 节点所属章节路径
	CanonicalPath     string // 节点所属章节路径（规范路径）
	ContentText       string // 节点内容文本
	ItemIndex         int    // 节点编号索引
}

// DisplayText 返回节点的展示文本
func (i *GraphItem) DisplayText() string {
	return utils.BlankToDefault(i.ContentText, utils.BlankToDefault(i.AnchorText, i.Title))
}

// GraphSectionWithChildren 包含目标节点及其子节点的查询结果
type GraphSectionWithChildren struct {
	Section  *GraphSection   `json:"section"`
	Children []*GraphSection `json:"children"`
}

// GraphSectionWithSiblings 包含目标节点及其父节点和前后兄弟节点的查询结果
type GraphSectionWithSiblings struct {
	Section         *GraphSection `json:"section"`
	Parent          *GraphSection `json:"parent"`
	PreviousSibling *GraphSection `json:"previousSibling"`
	NextSibling     *GraphSection `json:"nextSibling"`
}

// GraphQueryResult 结构图综合查询结果
type GraphQueryResult struct {
	TargetSection   *GraphSection   `json:"targetSection"`
	ParentSection   *GraphSection   `json:"parentSection"`
	Children        []*GraphSection `json:"children"`
	PreviousSibling *GraphSection   `json:"previousSibling"`
	NextSibling     *GraphSection   `json:"nextSibling"`
	TargetItem      *GraphItem      `json:"targetItem"`
	MatchedItems    []*GraphItem    `json:"matchedItems"`
	AllItems        []*GraphItem    `json:"allItems"`
}
