package vo

// ContentBlock 内容块
type ContentBlock struct {
	BlockNumber        int        // 块编号
	BlockType          string     // 块类型
	ParentBlockNumber  int        // 父块编号
	SectionPath        string     // 章节路径
	CanonicalPath      string     // 规范路径
	PageNumber         int        // 页码
	PageRange          string     // 页码范围
	BoundingBoxJSON    string     // 边界框JSON
	Text               string     // 块文本内容
	ContentWithWeight  string     // 带权重内容
	TableHTML          string     // 表格HTML
	TableRows          [][]string // 表格行数据
	ImageFileName      string     // 图片文件名
	ImageContentBase64 string     // 图片Base64内容
	ImageCaption       string     // 图片标题
	MetadataJSON       any        // 元数据JSON
}
