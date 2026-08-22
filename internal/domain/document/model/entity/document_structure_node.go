package entity

import (
	"github.com/duke-git/lancet/v2/strutil"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/document/model/enum"
)

// StructureNode 文档结构节点实体
type StructureNode struct {
	ID                int64  `gorm:"column:id"`                   // 节点ID
	DocumentId        int64  `gorm:"column:document_id"`          // 文档ID
	ParseTaskId       int64  `gorm:"column:parse_task_id"`        // 解析任务ID
	NodeNo            int    `gorm:"column:node_no"`              // 节点序号
	NodeType          int    `gorm:"column:node_type"`            // 节点类型
	ParentNodeId      int64  `gorm:"column:parent_node_id"`       // 父节点ID
	PrevSiblingNodeId int64  `gorm:"column:prev_sibling_node_id"` // 前一个兄弟节点ID
	NextSiblingNodeId int64  `gorm:"column:next_sibling_node_id"` // 后一个兄弟节点ID
	Depth             int    `gorm:"column:depth"`                // 深度
	NodeCode          string `gorm:"column:node_code"`            // 节点编码
	Title             string `gorm:"column:title"`                // 标题
	AnchorText        string `gorm:"column:anchor_text"`          // 锚文本
	CanonicalPath     string `gorm:"column:canonical_path"`       // 规范路径
	SectionPath       string `gorm:"column:section_path"`         // 章节路径
	ContentText       string `gorm:"column:content_text"`         // 内容文本
	ItemIndex         int    `gorm:"column:item_index"`           // 条目索引
	// todo 注意解析文本时要回填以下字段
	SyntaxSchemaVersion string `gorm:"column:syntax_schema_version"` // 语法模式版本
	SyntaxSourceSha256  string `gorm:"column:syntax_source_sha256"`  // 语法源SHA256
	SyntaxNodeId        string `gorm:"column:syntax_node_id"`        // 语法节点ID
	SyntaxNodeType      string `gorm:"column:syntax_node_type"`      // 语法节点类型
	SyntaxSourceOrigin  string `gorm:"column:syntax_source_origin"`  // 语法源位置
	SourceStartByte     int    `gorm:"column:source_start_byte"`     // 源开始字节
	SourceEndByte       int    `gorm:"column:source_end_byte"`       // 源结束字节
	SourceStartLine     int    `gorm:"column:source_start_line"`     // 源开始行
	SourceStartColumn   int    `gorm:"column:source_start_column"`   // 源开始列
	SourceEndLine       int    `gorm:"column:source_end_line"`       // 源结束行
	SourceEndColumn     int    `gorm:"column:source_end_column"`     // 源结束列
	ParentNodeNo        int    `gorm:"-"`                            // 父节点序号
	PrevSiblingNodeNo   int    `gorm:"-"`                            // 前序兄弟节点序号
	NextSiblingNodeNo   int    `gorm:"-"`                            // 后继兄弟节点序号
}

type StructureNodes []*StructureNode

// ExtractSectionTitles 提取章节标题（去重、取前 8 条）
func (n StructureNodes) ExtractSectionTitles() []string {
	if len(n) == 0 {
		return nil
	}
	return utils.FilterMapUniqueLimit(n, 8, func(node *StructureNode) (string, string, bool) {
		if node == nil || node.NodeType != enum.NodeTypeSection {
			return "", "", false
		}
		title := utils.Trim(node.Title)
		return title, title, title != ""
	})
}

func (n StructureNodes) FindNodeByPath(sectionPath, canonicalPath string) *StructureNode {
	if len(n) == 0 {
		return nil
	}
	sectionPath = strutil.Trim(sectionPath)
	canonicalPath = strutil.Trim(canonicalPath)
	if sectionPath == "" && canonicalPath == "" {
		return nil
	}
	for _, node := range n {
		if node == nil {
			continue
		}
		if canonicalPath != "" && strutil.Trim(node.CanonicalPath) == canonicalPath {
			return node
		}
		if sectionPath != "" && (strutil.Trim(node.SectionPath) == sectionPath || strutil.Trim(node.Title) == sectionPath) {
			return node
		}
	}
	return nil
}

// FindNodeByNo 根据 NodeNo 查找节点
func (n StructureNodes) FindNodeByNo(nodeNo int) *StructureNode {
	for _, node := range n {
		if node != nil && node.NodeNo == nodeNo {
			return node
		}
	}
	return nil
}

// FindChildrenByNo 根据父节点序号查找所有子节点序号
func (n StructureNodes) FindChildrenByNo(parentNodeNo int) []int {
	var children []int
	for _, node := range n {
		if node != nil && node.ParentNodeNo == parentNodeNo && node.NodeNo != parentNodeNo {
			children = append(children, node.NodeNo)
		}
	}
	return children
}
