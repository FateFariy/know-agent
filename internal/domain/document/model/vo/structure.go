package vo

// StructureNode 文档结构节点
type StructureNode struct {
	NodeNumber            int               // 节点编号
	NodeType              int               // 节点类型
	ParentNodeNumber      int               // 父节点编号
	PreviousSiblingNumber int               // 前序兄弟节点编号
	NextSiblingNumber     int               // 后继兄弟节点编号
	Depth                 int               // 节点深度
	NodeCode              string            // 节点编码
	Title                 string            // 节点标题
	AnchorText            string            // 锚点文本
	CanonicalPath         string            // 规范路径
	SectionPath           string            // 章节路径
	ContentText           string            // 节点内容文本
	ItemIndex             int               // 项目索引
	SyntaxProvenance      *SyntaxProvenance // 语法来源信息
}

// SyntaxProvenance 语法来源信息
type SyntaxProvenance struct {
	SchemaVersion  string      // 语法模式版本
	SourceHash     string      // 源文本哈希值
	SyntaxNodeID   string      // 语法节点ID
	SyntaxNodeType string      // 语法节点类型
	SourceOrigin   string      // 源文本来源
	SourceSpan     *SourceSpan // 源文本位置范围
}
