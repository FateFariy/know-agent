package vo

type DocumentStructureSignalKind int

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
	LineNo         int                         `json:"lineNo"`
	RawText        string                      `json:"rawText"`
	NormalizedText string                      `json:"normalizedText"`
	Kind           DocumentStructureSignalKind `json:"kind"`
	NodeCode       string                      `json:"nodeCode"`
	Title          string                      `json:"title"`
	LevelHint      *int                        `json:"levelHint"`
	IndentLevel    int                         `json:"indentLevel"`
	ItemIndex      *int                        `json:"itemIndex"`
	NumericPath    []int                       `json:"numericPath"`
	Reasons        []string                    `json:"reasons"`
	Confidence     float64                     `json:"confidence"`
}

func (s *DocumentStructureSignal) IsAmbiguous() bool {
	return (s.Kind == SignalKindHeadingCandidate) || (s.Confidence > 0.5 && s.Confidence < 0.7)
}

type DocumentStructureLogicalLine struct {
	LogicalLineNo  int    `json:"logicalLineNo"`
	PhysicalLineNo int    `json:"physicalLineNo"`
	SegmentNo      int    `json:"segmentNo"`
	IndentLevel    int    `json:"indentLevel"`
	RawText        string `json:"rawText"`
	NormalizedText string `json:"normalizedText"`
}

type DocumentStructureSignalBatch struct {
	ContextLines []string                   `json:"contextLines"`
	Signals      []*DocumentStructureSignal `json:"signals"`
}

type DocumentStructureNodeDraft struct {
	NodeNo            int     `json:"nodeNo"`
	LineNo            int     `json:"lineNo"`
	NodeType          int     `json:"nodeType"`
	ParentNodeNo      *int    `json:"parentNodeNo"`
	PrevSiblingNodeNo *int    `json:"prevSiblingNodeNo"`
	NextSiblingNodeNo *int    `json:"nextSiblingNodeNo"`
	Depth             int     `json:"depth"`
	NodeCode          string  `json:"nodeCode"`
	Title             string  `json:"title"`
	AnchorText        string  `json:"anchorText"`
	CanonicalPath     string  `json:"canonicalPath"`
	SectionPath       string  `json:"sectionPath"`
	ContentText       string  `json:"contentText"`
	ItemIndex         *int    `json:"itemIndex"`
	NumericPath       []int   `json:"numericPath"`
	SourceFamily      string  `json:"sourceFamily"`
	Confidence        float64 `json:"confidence"`
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
	NodeNo            int    `json:"nodeNo"`
	NodeType          int    `json:"nodeType"`
	ParentNodeNo      *int   `json:"parentNodeNo"`
	PrevSiblingNodeNo *int   `json:"prevSiblingNodeNo"`
	NextSiblingNodeNo *int   `json:"nextSiblingNodeNo"`
	Depth             int    `json:"depth"`
	NodeCode          string `json:"nodeCode"`
	Title             string `json:"title"`
	AnchorText        string `json:"anchorText"`
	CanonicalPath     string `json:"canonicalPath"`
	SectionPath       string `json:"sectionPath"`
	ContentText       string `json:"contentText"`
	ItemIndex         *int   `json:"itemIndex"`
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
