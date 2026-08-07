package markdown

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	extensionAst "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/text"

	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/document/model/vo"
)

// MarkdownParser 定义 Markdown 解析器接口
type MarkdownParser interface {
	Parse(sourceText string) (*vo.MarkdownSyntax, error)
	ToBlocks(syntax *vo.MarkdownSyntax, parserName string) []*entity.DocumentBlock
}

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
	level        *int
	marker       string
	ordinal      *int
	isHeader     bool
	alignment    string
	rowIndex     *int
	columnIndex  *int
	codeInfo     string
}

type openingFields struct {
	level        *int
	marker       string
	ordinal      *int
	isHeader     bool
	alignment    string
	rowIndex     *int
	columnIndex  *int
	cellSpanSet  bool
	cellCharSpan [2]int
}

func (p *GoldmarkParser) Parse(sourceText string) (*vo.MarkdownSyntax, error) {
	sourceBytes := []byte(sourceText)
	reader := text.NewReader(sourceBytes)
	rootAst := p.parser.Parser().Parse(reader)
	si := newSourceIndex(sourceText)

	var drafts []*nodeDraft
	draftMap := make(map[string]*nodeDraft)
	var stack []string
	tableRowCount := make(map[string]int)
	rowColIndex := make(map[string]int)

	appendNode := func(ntype, parentId string, span [2]int, hasSpan bool) *nodeDraft {
		oid := len(drafts)
		nid := fmt.Sprintf("md-%06d", oid)
		d := &nodeDraft{
			order:        oid,
			nodeId:       nid,
			parentNodeId: parentId,
			nodeType:     ntype,
			origin:       "markdown-parse",
			charSpan:     span,
			hasSpan:      hasSpan,
		}
		drafts = append(drafts, d)
		draftMap[nid] = d
		return d
	}
	rootDraft := appendNode(NodeDocument, "", [2]int{0, len([]rune(sourceText))}, true)
	stack = append(stack, rootDraft.nodeId)

	err := ast.Walk(rootAst, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if n.Kind() == ast.KindDocument {
			return ast.WalkContinue, nil
		}
		parentId := stack[len(stack)-1]
		ntype := mapNodeType(n)
		if entering {
			if isContainerNode(n) {
				lm := getNodeLineMap(n, sourceBytes)
				cs, ce, err := si.CharacterSpan(lm)
				if err != nil {
					return ast.WalkStop, err
				}
				span := [2]int{cs, ce}
				fields := p.fillOpeningFields(n, ntype, parentId, draftMap, tableRowCount, rowColIndex, si, span)
				draft := appendNode(ntype, parentId, span, true)
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
				lm := getNodeLineMap(n, sourceBytes)
				cs, ce, err := si.CharacterSpan(lm)
				if err != nil {
					return ast.WalkStop, err
				}
				draft := appendNode(ntype, parentId, [2]int{cs, ce}, true)
				draft.text = extractLeafText(n, sourceBytes)
				if cd, ok := n.(*ast.FencedCodeBlock); ok {
					draft.codeInfo = string(cd.Language(sourceBytes))
					draft.marker = "```"
				}
			}
			return ast.WalkContinue, nil
		}
		if isContainerNode(n) {
			curDraft := draftMap[stack[len(stack)-1]]
			if curDraft.text == "" {
				curDraft.text = extractInlineText(n, sourceBytes)
			}
			stack = stack[:len(stack)-1]
		}
		return ast.WalkContinue, nil
	})
	if err != nil {
		return nil, err
	}
	fillMissingSpans(drafts, draftMap)
	fillContainerText(drafts, draftMap)

	nodes := make([]*vo.MarkdownNode, 0, len(drafts))
	for _, d := range drafts {
		level := 0
		if d.level != nil {
			level = *d.level
		}
		ordinal := 0
		if d.ordinal != nil {
			ordinal = *d.ordinal
		}
		rowIndex := 0
		if d.rowIndex != nil {
			rowIndex = *d.rowIndex
		}
		columnIndex := 0
		if d.columnIndex != nil {
			columnIndex = *d.columnIndex
		}
		nodes = append(nodes, &vo.MarkdownNode{
			Order:        d.order,
			NodeId:       d.nodeId,
			ParentNodeId: d.parentNodeId,
			NodeType:     d.nodeType,
			Origin:       d.origin,
			Text:         d.text,
			Level:        level,
			Marker:       d.marker,
			Ordinal:      ordinal,
			IsHeader:     d.isHeader,
			Alignment:    d.alignment,
			RowIndex:     rowIndex,
			ColumnIndex:  columnIndex,
			CodeInfo:     d.codeInfo,
			SourceSpan:   si.SourceSpan(d.charSpan),
		})
	}
	hash := sha256.Sum256(sourceBytes)
	return &vo.MarkdownSyntax{
		SchemaVersion:     "1.0.0",
		SourceOrigin:      "markdown-parse",
		SourceText:        sourceText,
		SourceLengthBytes: len(sourceBytes),
		SourceSHA256:      hex.EncodeToString(hash[:]),
		Nodes:             nodes,
	}, nil
}

func (p *GoldmarkParser) fillOpeningFields(n ast.Node, nodeType string, parentId string,
	draftsById map[string]*nodeDraft, rowCountByTable map[string]int, colCountByRow map[string]int,
	sourceIndex sourceIndexer, parentCharSpan [2]int) openingFields {
	fields := openingFields{}
	switch nodeType {
	case NodeHeading:
		h := n.(*ast.Heading)
		lvl := h.Level
		fields.level = &lvl
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

func (p *GoldmarkParser) ToBlocks(syntax *vo.MarkdownSyntax, parserName string) []*entity.DocumentBlock {
	return SyntaxToBlocks(syntax, parserName)
}
