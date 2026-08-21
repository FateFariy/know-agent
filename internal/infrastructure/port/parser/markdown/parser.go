package markdown

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	extensionAst "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/text"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/document/model/enum"
	"github.com/swiftbit/know-agent/internal/domain/document/model/shared"
)

type GoldmarkParser struct {
	parser goldmark.Markdown
}

func NewGoldmarkParser() *GoldmarkParser {
	md := goldmark.New(goldmark.WithExtensions(extension.Table))
	return &GoldmarkParser{
		parser: md,
	}
}

type nodeDraft struct {
	order        int
	nodeId       string
	parentNodeId string
	nodeType     string
	origin       string
	charSpan     [2]int
	hasSpan      bool
	text         string
	level        int
	marker       string
	ordinal      *int
	isHeader     bool
	alignment    string
	rowIndex     *int
	columnIndex  *int
	codeInfo     string
}

type openingFields struct {
	level        int
	marker       string
	ordinal      *int
	isHeader     bool
	alignment    string
	rowIndex     *int
	columnIndex  *int
	cellSpanSet  bool
	cellCharSpan [2]int
}

const Markdown = "native_markdown"

func (p *GoldmarkParser) Name() string {
	return Markdown
}

func (p *GoldmarkParser) Parse(_ context.Context, sourceText []byte) (entity.DocumentBlocks, error) {
	reader := text.NewReader(sourceText)
	rootAst := p.parser.Parser().Parse(reader)
	si := newSourceIndex(string(sourceText))

	var drafts []*nodeDraft
	draftMap := make(map[string]*nodeDraft)
	var stack []string
	tableRowCount := make(map[string]int)
	rowColIndex := make(map[string]int)

	appendNode := func(nodeType, parentId string, span [2]int, hasSpan bool) *nodeDraft {
		oid := len(drafts)
		nid := fmt.Sprintf("md-%06d", oid)
		d := &nodeDraft{
			order:        oid,
			nodeId:       nid,
			parentNodeId: parentId,
			nodeType:     nodeType,
			origin:       "markdown-parse",
			charSpan:     span,
			hasSpan:      hasSpan,
		}
		drafts = append(drafts, d)
		draftMap[nid] = d
		return d
	}
	rootDraft := appendNode(NodeDocument, "", [2]int{0, utils.Len(string(sourceText))}, true)
	stack = append(stack, rootDraft.nodeId)

	err := ast.Walk(rootAst, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if n.Kind() == ast.KindDocument {
			return ast.WalkContinue, nil
		}
		parentId := stack[len(stack)-1]
		nodeType := mapNodeType(n)
		if nodeType == "" {
			return ast.WalkContinue, nil
		}
		if entering {
			if isContainerNode(n) {
				lm := getNodeLineMap(n, sourceText)
				cs, ce, err := si.CharacterSpan(lm)
				if err != nil {
					return ast.WalkStop, err
				}
				span := [2]int{cs, ce}
				fields := p.fillOpeningFields(n, nodeType, parentId, draftMap, tableRowCount, rowColIndex, si, span)
				draft := appendNode(nodeType, parentId, span, true)
				draft.level = fields.level
				draft.marker = fields.marker
				draft.ordinal = fields.ordinal
				draft.isHeader = fields.isHeader
				draft.alignment = fields.alignment
				draft.rowIndex = fields.rowIndex
				draft.columnIndex = fields.columnIndex
				if fields.cellSpanSet {
					draft.charSpan = fields.cellCharSpan
				}
				stack = append(stack, draft.nodeId)
			} else {
				lm := getNodeLineMap(n, sourceText)
				cs, ce, err := si.CharacterSpan(lm)
				if err != nil {
					return ast.WalkStop, err
				}
				draft := appendNode(nodeType, parentId, [2]int{cs, ce}, true)
				draft.text = extractLeafText(n, sourceText)
				if cd, ok := n.(*ast.FencedCodeBlock); ok {
					draft.codeInfo = string(cd.Language(sourceText))
					draft.marker = "```"
				}
			}
			return ast.WalkContinue, nil
		}
		if isContainerNode(n) {
			curDraft := draftMap[stack[len(stack)-1]]
			if curDraft.text == "" {
				curDraft.text = extractInlineText(n, sourceText)
			}
			stack = stack[:len(stack)-1]
		}
		return ast.WalkContinue, nil
	})
	if err != nil {
		return nil, err
	}
	fillMissingSpans(drafts, draftMap)
	fillContainerText(drafts)

	nodes := make([]*shared.MarkdownNode, 0, len(drafts))
	for _, d := range drafts {
		nodes = append(nodes, &shared.MarkdownNode{
			Order:        d.order,
			NodeId:       d.nodeId,
			ParentNodeId: d.parentNodeId,
			NodeType:     d.nodeType,
			Origin:       d.origin,
			Text:         d.text,
			Level:        d.level,
			Marker:       d.marker,
			Ordinal:      utils.PointerOrDefault(d.ordinal, 0),
			IsHeader:     d.isHeader,
			Alignment:    d.alignment,
			RowIndex:     utils.PointerOrDefault(d.rowIndex, 0),
			ColumnIndex:  utils.PointerOrDefault(d.columnIndex, 0),
			CodeInfo:     d.codeInfo,
			SourceSpan:   si.SourceSpan(d.charSpan),
		})
	}
	hash := sha256.Sum256(sourceText)

	syntax := &shared.MarkdownSyntax{
		SchemaVersion:     "1.0.0",
		SourceOrigin:      "markdown-parse",
		SourceText:        string(sourceText),
		SourceLengthBytes: len(sourceText),
		SourceSHA256:      hex.EncodeToString(hash[:]),
		Nodes:             nodes,
	}
	return p.syntaxToBlocks(syntax), nil
}

// syntaxToBlocks 将 MarkdownSyntax 文档转换为 DocumentBlock 列表
func (p *GoldmarkParser) syntaxToBlocks(syntax *shared.MarkdownSyntax) entity.DocumentBlocks {
	if syntax == nil || len(syntax.Nodes) == 0 {
		return nil
	}

	root := syntax.Nodes[0]
	topLevel := make([]*shared.MarkdownNode, 0)
	for _, node := range syntax.Nodes {
		if node.ParentNodeId == root.NodeId {
			topLevel = append(topLevel, node)
		}
	}

	childrenByParent := buildChildrenMap(syntax.Nodes)

	var blocks []*entity.DocumentBlock
	for _, node := range topLevel {
		rawText := nodeSourceText(syntax, node)
		blockType, txt := legacyBlockContent(node, rawText)
		if txt == "" && blockType != enum.BlockTypeTable {
			continue
		}
		rows := tableRows(node, childrenByParent)
		if blockType == enum.BlockTypeTable && txt == "" {
			var rowStrs []string
			for _, row := range rows {
				rowStrs = append(rowStrs, strings.Join(row, " | "))
			}
			txt = strings.Join(rowStrs, "\n")
		}
		tableHTML := ""
		if blockType == enum.BlockTypeTable {
			tableHTML = renderMarkdownTable(rawText)
		}

		metadata := map[string]any{
			"parser":              Markdown,
			"syntaxSchemaVersion": syntax.SchemaVersion,
			"syntaxNodeId":        node.NodeId,
			"syntaxNodeType":      node.NodeType,
			"sourceOrigin":        node.Origin,
			"sourceSpan":          node.SourceSpan,
		}
		if node.Level > 0 {
			metadata["headingLevel"] = node.Level
		}
		if node.Marker != "" {
			metadata["originalMarker"] = node.Marker
		}

		block := &entity.DocumentBlock{
			BlockNo:   len(blocks) + 1,
			BlockType: blockType,
			Text:      txt,
			TableHTML: tableHTML,
			TableRows: rows,
			Metadata:  metadata,
		}
		blocks = append(blocks, block)
	}
	return blocks
}

func (p *GoldmarkParser) fillOpeningFields(n ast.Node, nodeType string, parentId string,
	draftsById map[string]*nodeDraft, rowCountByTable map[string]int, colCountByRow map[string]int,
	sourceIndex *sourceIndex, parentCharSpan [2]int) openingFields {
	fields := openingFields{}
	switch nodeType {
	case NodeHeading:
		h := n.(*ast.Heading)
		lvl := h.Level
		fields.level = lvl
		fields.marker = ""
		for i := 0; i < lvl; i++ {
			fields.marker += "#"
		}
	case NodeOrderedList:
		lst := n.(*ast.List)
		start := lst.Start
		fields.ordinal = &start
	case NodeListItem:
		lst := n.Parent().(*ast.List)
		if lst.IsOrdered() {
			idx := 0
			for c := lst.FirstChild(); c != nil; c = c.NextSibling() {
				if c == n {
					break
				}
				idx++
			}
			ord := lst.Start + idx
			fields.ordinal = &ord
			fields.marker = strconv.Itoa(ord) + "."
		} else {
			fields.marker = string(lst.Marker)
		}
	case NodeTableRow:
		tid := findAncestor(parentId, NodeTable, draftsById)
		ridx := rowCountByTable[tid]
		rowCountByTable[tid] = ridx + 1
		fields.rowIndex = &ridx
	case NodeTableCell:
		rid := findAncestor(parentId, NodeTableRow, draftsById)
		cidx := colCountByRow[rid]
		colCountByRow[rid] = cidx + 1
		fields.columnIndex = &cidx
		tc := n.(*extensionAst.TableCell)
		fields.isHeader = n.Parent().Kind() == extensionAst.KindTableHeader
		switch tc.Alignment {
		case extensionAst.AlignLeft:
			fields.alignment = "LEFT"
		case extensionAst.AlignCenter:
			fields.alignment = "CENTER"
		case extensionAst.AlignRight:
			fields.alignment = "RIGHT"
		default:
			fields.alignment = "NONE"
		}
		s, e := sourceIndex.TableCellSpan(parentCharSpan, cidx)
		fields.cellCharSpan = [2]int{s, e}
		fields.cellSpanSet = true
	}
	return fields
}
