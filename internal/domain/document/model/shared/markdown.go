package shared

// MarkdownSyntax Markdown语法分析结果
type MarkdownSyntax struct {
	SchemaVersion     string          // 语法模式版本
	SourceOrigin      string          // 源文本来源标识
	SourceText        string          // 原始源文本
	SourceLengthBytes int             // 源文本字节长度
	SourceSHA256      string          // 源文本SHA256哈希值
	Nodes             []*MarkdownNode // 语法节点列表
}

// MarkdownNode Markdown语法节点
type MarkdownNode struct {
	Order        int         // 节点序号
	NodeId       string      // 节点唯一标识
	ParentNodeId string      // 父节点标识
	NodeType     string      // 节点类型
	Origin       string      // 节点来源
	SourceSpan   *SourceSpan // 源文本位置范围
	Text         string      // 节点文本内容
	Level        int         // 节点层级
	Marker       string      // 标记符号(如列表符号)
	Ordinal      int         // 序号值(有序列表)
	IsHeader     bool        // 是否为标题
	Alignment    string      // 对齐方式
	RowIndex     int         // 表格行索引
	ColumnIndex  int         // 表格列索引
	CodeInfo     string      // 附加信息
}

// SourceSpan 源文本位置范围
type SourceSpan struct {
	StartByte   int // 起始字节位置
	EndByte     int // 结束字节位置
	StartLine   int // 起始行号
	StartColumn int // 起始列号
	EndLine     int // 结束行号
	EndColumn   int // 结束列号
}

// SyntaxProvenance 语法来源信息
type SyntaxProvenance struct {
	SchemaVersion  string      // 语法模式版本
	SourceHash     string      // 源文本哈希值
	SyntaxNodeId   string      // 语法节点ID
	SyntaxNodeType string      // 语法节点类型
	SourceOrigin   string      // 源文本来源
	SourceSpan     *SourceSpan // 源文本位置范围
}
