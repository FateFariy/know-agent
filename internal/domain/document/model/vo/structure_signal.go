package vo

type DocumentStructureSignalKind = int

const (
	SignalKindBlank DocumentStructureSignalKind = iota
	SignalKindNoise
	SignalKindDocumentTitle
	SignalKindHeading
	SignalKindHeadingCandidate
	SignalKindBody
	SignalKindListItem
	SignalKindStepItem
	SignalKindTableRow
	SignalKindQuote
)

type DocumentStructureSignal struct {
	LineNo         int                         // 逻辑行号
	RawText        string                      // 原始文本
	NormalizedText string                      // 规范化文本
	Kind           DocumentStructureSignalKind // 信号类型
	NodeCode       string                      // 节点代码
	Title          string                      // 标题
	LevelHint      int                         // 级别提示
	IndentLevel    int                         // 缩进级别
	ItemIndex      int                         // 项目索引
	NumericPath    []int                       // 数值路径
	Reasons        []string                    // 原因
	Confidence     float64                     // 置信度
}

func (s *DocumentStructureSignal) IsAmbiguous() bool {
	return (s.Kind == SignalKindHeadingCandidate) || (s.Confidence > 0.5 && s.Confidence < 0.7)
}

type DocumentStructureLogicalLine struct {
	LogicalLineNo  int    // 逻辑行号
	PhysicalLineNo int    // 物理行号
	SegmentNo      int    // 段落编号
	IndentLevel    int    // 缩进级别
	RawText        string // 原始文本
	NormalizedText string // 规范化文本
}

type LineContext struct {
	PreviousNonBlank *DocumentStructureLogicalLine // 前一个非空行
	NextNonBlank     *DocumentStructureLogicalLine // 后一行非空行
	BlankBefore      bool                          // 前行是否为空白
	BlankAfter       bool                          // 后行是否为空白
}

type DocumentStructureSignalBatch struct {
	ContextLines []string                   `json:"contextLines"` // 上下文行
	Signals      []*DocumentStructureSignal `json:"signals"`      // 信号
}

type DocumentStructureNodeDraft struct {
	NodeNo            int                       `json:"nodeNo"`            // 节点编号
	LineNo            int                       `json:"lineNo"`            // 行号
	NodeType          DocumentStructureNodeType `json:"nodeType"`          // 节点类型
	ParentNodeNo      *int                      `json:"parentNodeNo"`      // 父节点编号
	PrevSiblingNodeNo *int                      `json:"prevSiblingNodeNo"` // 前兄弟节点编号
	NextSiblingNodeNo *int                      `json:"nextSiblingNodeNo"` // 后兄弟节点编号
	Depth             int                       `json:"depth"`             // 深度
	NodeCode          string                    `json:"nodeCode"`          // 节点代码
	Title             string                    `json:"title"`             // 标题
	AnchorText        string                    `json:"anchorText"`        // 锚文本
	CanonicalPath     string                    `json:"canonicalPath"`     // 规范路径
	SectionPath       string                    `json:"sectionPath"`       // 段落路径
	ContentText       string                    `json:"contentText"`       // 内容文本
	ItemIndex         *int                      `json:"itemIndex"`         // 项目索引
	NumericPath       []int                     `json:"numericPath"`       // 数值路径
	SourceFamily      string                    `json:"sourceFamily"`      // 源家族
	Confidence        float64                   `json:"confidence"`        // 置信度
}

func (d *DocumentStructureNodeDraft) IsSection() bool {
	return d.NodeType == NodeTypeSection
}

func (d *DocumentStructureNodeDraft) IsListLike() bool {
	return d.NodeType == NodeTypeListItem || d.NodeType == NodeTypeStep
}

func (d *DocumentStructureNodeDraft) AppendLine(line string) {
	if d.ContentText == "" {
		d.ContentText = line
	} else {
		d.ContentText += "\n" + line
	}
}

type DocumentStructureNodeCandidate struct {
	NodeNo            int                       `json:"nodeNo"`
	NodeType          DocumentStructureNodeType `json:"nodeType"`
	ParentNodeNo      *int                      `json:"parentNodeNo"`
	PrevSiblingNodeNo *int                      `json:"prevSiblingNodeNo"`
	NextSiblingNodeNo *int                      `json:"nextSiblingNodeNo"`
	Depth             int                       `json:"depth"`
	NodeCode          string                    `json:"nodeCode"`
	Title             string                    `json:"title"`
	AnchorText        string                    `json:"anchorText"`
	CanonicalPath     string                    `json:"canonicalPath"`
	SectionPath       string                    `json:"sectionPath"`
	ContentText       string                    `json:"contentText"`
	ItemIndex         *int                      `json:"itemIndex"`
}

type DocumentStructureNodeType int

const (
	NodeTypeDocument DocumentStructureNodeType = iota + 1
	NodeTypeSection
	NodeTypeListItem
	NodeTypeStep
)

type DisambiguationResult struct {
	LineNo       int    `json:"lineNo"`
	ResolvedKind string `json:"resolvedKind"`
	LevelHint    *int   `json:"levelHint"`
}
