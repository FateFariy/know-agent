package vo

// TableCandidate 表格候选
type TableCandidate struct {
	SourceBlockNo   int                    // 源块编号
	SectionPath     string                 // 章节路径
	PageNo          int                    // 页码
	PageRange       string                 // 页码范围
	BoundingBoxJson string                 // 边界框JSON
	TableHTML       string                 // 表格HTML内容
	TitleHint       string                 // 标题提示
	ProjectionOwner string                 // 投影所有者
	SchemaVersion   string                 // 模式版本
	SourceOrigin    string                 // 源文本来源
	SourceHash      string                 // 源文本哈希
	SyntaxNodeId    string                 // 语法节点ID
	SourceSpan      *SourceSpan            // 源文本位置范围
	Rows            []*TableRow            // 表格行列表
	SourceMetadata  map[string]interface{} // 源元数据
}

// TableRow 表格行
type TableRow struct {
	RowIndex     int          // 行索引
	IsHeader     bool         // 是否为表头行
	SourceRowNo  int          // 源行号
	SyntaxNodeId string       // 语法节点ID
	SourceSpan   *SourceSpan  // 源文本位置范围
	Cells        []*TableCell // 单元格列表
}

// TableCell 表格单元格
type TableCell struct {
	RowIndex        int            // 行索引
	ColumnIndex     int            // 列索引
	IsHeader        bool           // 是否为表头单元格
	Alignment       string         // 对齐方式
	Text            string         // 单元格文本
	SourceRowNo     int            // 源行号
	SourceColumnNo  int            // 源列号
	SourceCellRef   string         // 源单元格引用
	BoundingBoxJson string         // 边界框JSON
	SyntaxNodeId    string         // 语法节点ID
	SourceSpan      *SourceSpan    // 源文本位置范围
	SourceMetadata  map[string]any // 源元数据
}
