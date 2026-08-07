package entity

// StructureNode 文档结构节点实体
type StructureNode struct {
	ID                int64  `gorm:"column:id"`                   // 节点ID
	DocumentId        int64  `gorm:"column:document_id"`          // 文档ID
	ParseTaskId       int64  `gorm:"column:parse_task_id"`        // 解析任务ID
	NodeNo            int    `gorm:"column:node_no"`              // 节点序号
	NodeType          int    `gorm:"column:node_type"`            // 节点类型
	ParentNodeId      int64  `gorm:"column:parent_node_id"`       // 父节点ID
	PrevSiblingNodeId int64  `gorm:"column:prev_sibling_node_id"` // 前一个兄弟节点ID
	NextSiblingNodeId int64  `gorm:"column:next_sibling_node_id"` // 后一个兄弟节点ID
	Depth             int    `gorm:"column:depth"`                // 深度
	NodeCode          string `gorm:"column:node_code"`            // 节点编码
	Title             string `gorm:"column:title"`                // 标题
	AnchorText        string `gorm:"column:anchor_text"`          // 锚文本
	CanonicalPath     string `gorm:"column:canonical_path"`       // 规范路径
	SectionPath       string `gorm:"column:section_path"`         // 章节路径
	ContentText       string `gorm:"column:content_text"`         // 内容文本
	ItemIndex         int    `gorm:"column:item_index"`           // 条目索引
	// todo 注意解析文本时要回填以下字段
	SyntaxSchemaVersion string `gorm:"column:syntax_schema_version"` // 语法模式版本
	SyntaxSourceSha256  string `gorm:"column:syntax_source_sha256"`  // 语法源SHA256
	SyntaxNodeId        string `gorm:"column:syntax_node_id"`        // 语法节点ID
	SyntaxNodeType      string `gorm:"column:syntax_node_type"`      // 语法节点类型
	SyntaxSourceOrigin  string `gorm:"column:syntax_source_origin"`  // 语法源位置
	SourceStartByte     int    `gorm:"column:source_start_byte"`     // 源开始字节
	SourceEndByte       int    `gorm:"column:source_end_byte"`       // 源结束字节
	SourceStartLine     int    `gorm:"column:source_start_line"`     // 源开始行
	SourceStartColumn   int    `gorm:"column:source_start_column"`   // 源开始列
	SourceEndLine       int    `gorm:"column:source_end_line"`       // 源结束行
	SourceEndColumn     int    `gorm:"column:source_end_column"`     // 源结束列
	ParentNodeNo        int    `gorm:"-"`                            // 父节点序号
	PrevSiblingNodeNo   int    `gorm:"-"`                            // 前序兄弟节点序号
	NextSiblingNodeNo   int    `gorm:"-"`                            // 后继兄弟节点序号
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
