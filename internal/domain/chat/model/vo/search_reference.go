package vo

// SearchReference 检索引用来源
type SearchReference struct {
	ReferenceId       string  `json:"referenceId"`       // 引用ID
	SourceType        string  `json:"sourceType"`        // 来源类型
	Title             string  `json:"title"`             // 标题
	Url               string  `json:"url"`               // URL
	Snippet           string  `json:"snippet"`           // 片段
	DocumentId        int64   `json:"documentId"`        // 文档ID
	DocumentName      string  `json:"documentName"`      // 文档名称
	ChunkId           int64   `json:"chunkId"`           // 块ID
	ParentBlockId     int64   `json:"parentBlockId"`     // 父块ID
	ParentBlockNo     int     `json:"parentBlockNo"`     // 父块序号
	ChunkNo           int     `json:"chunkNo"`           // 块序号
	SectionPath       string  `json:"sectionPath"`       // 章节路径
	StructureNodeId   int64   `json:"structureNodeId"`   // 结构节点ID
	StructureNodeType int     `json:"structureNodeType"` // 结构节点类型
	CanonicalPath     string  `json:"canonicalPath"`     // 规范路径
	ItemIndex         int     `json:"itemIndex"`         // 项索引
	Score             float64 `json:"score"`             // 相似度分数
	SubQuestionIndex  int     `json:"subQuestionIndex"`  // 子问题索引
}
