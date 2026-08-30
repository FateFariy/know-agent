package markdown

import (
	"strings"

	"github.com/yuin/goldmark/ast"
	extensionAst "github.com/yuin/goldmark/extension/ast"
)

// mapNodeType maps an AST node to a node type string
func mapNodeType(n ast.Node) string {
	switch n.Kind() {
	case ast.KindHeading:
		return NodeHeading
	case ast.KindParagraph:
		return NodeParagraph
	case ast.KindTextBlock:
		return NodeParagraph
	case ast.KindList:
		lst := n.(*ast.List)
		if lst.IsOrdered() {
			return NodeOrderedList
		}
		return NodeUnorderedList
	case ast.KindListItem:
		return NodeListItem
	case ast.KindBlockquote:
		return NodeBlockquote
	case extensionAst.KindTable:
		return NodeTable
	case extensionAst.KindTableHeader:
		return NodeTableHead
	case extensionAst.KindTableRow:
		return NodeTableRow
	case extensionAst.KindTableCell:
		return NodeTableCell
	case ast.KindFencedCodeBlock:
		return NodeCodeBlock
	case ast.KindCodeBlock:
		return NodeCodeBlock
	case ast.KindThematicBreak:
		return NodeThematicBreak
	case ast.KindHTMLBlock:
		return NodeHTMLBlock
	}
	return ""
}

// isContainerNode checks if a node type is a container (has children)
func isContainerNode(n ast.Node) bool {
	switch n.Kind() {
	case ast.KindHeading, ast.KindParagraph, ast.KindTextBlock, ast.KindList,
		ast.KindListItem, ast.KindBlockquote,
		extensionAst.KindTable, extensionAst.KindTableHeader,
		extensionAst.KindTableRow,
		extensionAst.KindTableCell:
		return true
	}
	return false
}

// isInlineNode checks if a node is an inline node (BaseInline) that panics on Lines()
func isInlineNode(n ast.Node) bool {
	switch n.Kind() {
	case ast.KindString, ast.KindText, ast.KindEmphasis,
		ast.KindCodeSpan, ast.KindRawHTML, ast.KindImage, ast.KindLink,
		ast.KindAutoLink:
		return true
	}
	return false
}

// getNodeLineMap extracts the line map [startLine, endLine] from a node
func getNodeLineMap(n ast.Node, source []byte) []int {
	if isInlineNode(n) {
		return []int{0, 0}
	}
	lines := n.Lines()
	if lines.Len() > 0 {
		firstSeg := lines.At(0)
		lastSeg := lines.At(lines.Len() - 1)

		startLine := 0
		for i, b := range source {
			if b == '\n' {
				startLine++
			}
			if i >= firstSeg.Start {
				break
			}
		}
		// 从第一个 segment 的起始到最后一个 segment 的结尾统计总行数
		lineCount := 0
		for i := firstSeg.Start; i < lastSeg.Stop && i < len(source); i++ {
			if source[i] == '\n' {
				lineCount++
			}
		}
		return []int{startLine, startLine + lineCount + 1}
	}
	return []int{0, 0}
}

// extractLeafText extracts text from a leaf node (code block, etc.)
func extractLeafText(n ast.Node, source []byte) string {
	switch n.Kind() {
	case ast.KindFencedCodeBlock:
		fcb := n.(*ast.FencedCodeBlock)
		lines := fcb.Lines()
		var result []byte
		for i := 0; i < lines.Len(); i++ {
			seg := lines.At(i)
			result = append(result, seg.Value(source)...)
		}
		return string(result)
	case ast.KindCodeBlock:
		cb := n.(*ast.CodeBlock)
		lines := cb.Lines()
		var result []byte
		for i := 0; i < lines.Len(); i++ {
			seg := lines.At(i)
			result = append(result, seg.Value(source)...)
		}
		return string(result)
	case ast.KindHTMLBlock:
		hb := n.(*ast.HTMLBlock)
		lines := hb.Lines()
		var result []byte
		for i := 0; i < lines.Len(); i++ {
			seg := lines.At(i)
			result = append(result, seg.Value(source)...)
		}
		return string(result)
	case ast.KindThematicBreak:
		return ""
	}
	return ""
}

func fillMissingSpans(drafts []*nodeDraft, draftMap map[string]*nodeDraft) {
	childrenByParent := make(map[string][]*nodeDraft)
	for _, d := range drafts {
		if d.parentNodeId != "" {
			childrenByParent[d.parentNodeId] = append(childrenByParent[d.parentNodeId], d)
		}
	}

	for i := len(drafts) - 1; i >= 0; i-- {
		d := drafts[i]
		if d.hasSpan {
			continue
		}
		children := childrenByParent[d.nodeId]
		if len(children) > 0 {
			var minStart, maxEnd int
			hasSpan := false
			for _, child := range children {
				if child.hasSpan {
					if !hasSpan {
						minStart, maxEnd = child.charSpan[0], child.charSpan[1]
						hasSpan = true
					} else {
						if child.charSpan[0] < minStart {
							minStart = child.charSpan[0]
						}
						if child.charSpan[1] > maxEnd {
							maxEnd = child.charSpan[1]
						}
					}
				}
			}
			if hasSpan {
				d.charSpan = [2]int{minStart, maxEnd}
				d.hasSpan = true
				continue
			}
		}
		parent := draftMap[d.parentNodeId]
		if parent != nil && parent.hasSpan {
			d.charSpan = parent.charSpan
			d.hasSpan = true
		}
	}
}

func fillContainerText(drafts []*nodeDraft) {
	childrenByParent := make(map[string][]*nodeDraft)
	for _, d := range drafts {
		if d.parentNodeId != "" {
			childrenByParent[d.parentNodeId] = append(childrenByParent[d.parentNodeId], d)
		}
	}

	for i := len(drafts) - 1; i >= 0; i-- {
		d := drafts[i]
		if d.text != "" || d.nodeType == NodeDocument {
			continue
		}
		children := childrenByParent[d.nodeId]
		texts := make([]string, 0, len(children))
		for _, child := range children {
			if child.text != "" {
				texts = append(texts, child.text)
			}
		}
		if d.nodeType == NodeTableRow {
			// For table rows, join cell texts with " | "
			cellTexts := make([]string, 0, len(children))
			for _, child := range children {
				if child.nodeType == NodeTableCell && child.text != "" {
					cellTexts = append(cellTexts, child.text)
				}
			}
			d.text = strings.Join(cellTexts, " | ")
		} else if len(texts) > 0 {
			d.text = strings.Join(texts, "\n")
		}
	}
}

// findAncestor 查找指定类型的祖先节点ID
func findAncestor(nodeId string, nodeType string, draftsById map[string]*nodeDraft) string {
	currentId := nodeId
	for currentId != "" {
		current, ok := draftsById[currentId]
		if !ok {
			return ""
		}
		if current.nodeType == nodeType {
			return currentId
		}
		currentId = current.parentNodeId
	}
	return ""
}

// extractInlineText 提取内联文本（文本、代码、换行等）
func extractInlineText(n ast.Node, source []byte) string {
	var res []byte
	err := ast.Walk(n, func(child ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch child.Kind() {
		case ast.KindText:
			if t, ok := child.(*ast.Text); ok {
				res = append(res, t.Text(source)...)
				if t.SoftLineBreak() || t.HardLineBreak() {
					res = append(res, '\n')
				}
			}
		case ast.KindCodeSpan, ast.KindRawHTML:
			res = append(res, child.Text(source)...)
		}
		return ast.WalkContinue, nil
	})
	if err != nil {
		return ""
	}
	return string(res)
}
