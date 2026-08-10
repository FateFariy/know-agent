package markdown

import (
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"

	"github.com/swiftbit/know-agent/internal/domain/document/model/shared"
)

// buildChildrenMap 构建子节点映射
func buildChildrenMap(nodes []*shared.MarkdownNode) map[string][]*shared.MarkdownNode {
	childrenByParent := make(map[string][]*shared.MarkdownNode)
	for _, node := range nodes {
		if node.ParentNodeId != "" {
			childrenByParent[node.ParentNodeId] = append(childrenByParent[node.ParentNodeId], node)
		}
	}
	return childrenByParent
}

// nodeSourceText 从 MarkdownSyntax 中提取节点的源文本
func nodeSourceText(syntax *shared.MarkdownSyntax, node *shared.MarkdownNode) string {
	if node.SourceSpan == nil {
		return ""
	}
	sourceBytes := []byte(syntax.SourceText)
	if node.SourceSpan.StartByte >= len(sourceBytes) || node.SourceSpan.EndByte > len(sourceBytes) {
		return ""
	}
	return string(sourceBytes[node.SourceSpan.StartByte:node.SourceSpan.EndByte])
}

// legacyBlockContent 根据节点类型生成块类型和文本内容
func legacyBlockContent(node *shared.MarkdownNode, rawText string) (string, string) {
	rawText = strings.TrimSpace(rawText)
	switch node.NodeType {
	case NodeHeading:
		return BlockTypeTitle, strings.TrimSpace(node.Text)
	case NodeOrderedList, NodeUnorderedList:
		return BlockTypeList, rawText
	case NodeTable:
		return BlockTypeTable, strings.TrimSpace(node.Text)
	case NodeCodeBlock:
		return BlockTypeCode, rawText
	case NodeBlockquote:
		return BlockTypeBlockquote, rawText
	case NodeThematicBreak:
		return BlockTypeThematicBreak, rawText
	case NodeHTMLBlock:
		return BlockTypeHTML, rawText
	default:
		return BlockTypeText, rawText
	}
}

// tableRows 提取表格行数据
func tableRows(table *shared.MarkdownNode, childrenByParent map[string][]*shared.MarkdownNode) [][]string {
	descendants := make([]*shared.MarkdownNode, 0)
	pending := make([]*shared.MarkdownNode, len(childrenByParent[table.NodeId]))
	copy(pending, childrenByParent[table.NodeId])
	for len(pending) > 0 {
		node := pending[0]
		pending = pending[1:]
		descendants = append(descendants, node)
		children := childrenByParent[node.NodeId]
		pending = append(pending, children...)
	}

	// 收集所有 TABLE_CELL 节点
	var cells []*shared.MarkdownNode
	for _, node := range descendants {
		if node.NodeType == NodeTableCell {
			cells = append(cells, node)
		}
	}
	if len(cells) == 0 {
		return nil
	}

	// 计算行数和列数
	rowCount := 0
	colCount := 0
	for _, cell := range cells {
		if cell.RowIndex+1 > rowCount {
			rowCount = cell.RowIndex + 1
		}
		if cell.ColumnIndex+1 > colCount {
			colCount = cell.ColumnIndex + 1
		}
	}
	if rowCount == 0 || colCount == 0 {
		return nil
	}

	rows := make([][]string, rowCount)
	for i := range rows {
		rows[i] = make([]string, colCount)
	}
	for _, cell := range cells {
		r, c := cell.RowIndex, cell.ColumnIndex
		if r < rowCount && c < colCount {
			rows[r][c] = cell.Text
		}
	}
	return rows
}

// renderMarkdownTable 使用 goldmark 渲染表格为 HTML
func renderMarkdownTable(mdText string) string {
	md := goldmark.New(
		goldmark.WithExtensions(extension.Table),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
	)
	reader := text.NewReader([]byte(mdText))
	doc := md.Parser().Parse(reader)
	var buf strings.Builder
	if err := md.Renderer().Render(&buf, []byte(mdText), doc); err != nil {
		return ""
	}
	return buf.String()
}
