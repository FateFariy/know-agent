package vo

// DocumentAnalysisResult 文档分析结果
type DocumentAnalysisResult struct {
	ParsedText          string // 解析后的文本
	CharCount           int    // 字符数
	TokenCount          int    // Token数量
	StructureLevel      int    // 结构化等级
	ContentQualityLevel int    // 内容质量等级
	HeadingCount        int    // 标题数量
	ParagraphCount      int    // 段落数量
	MaxParagraphLength  int    // 最长段落长度
}
