package vo

// AnalysisResult 文档分析结果
type AnalysisResult struct {
	ParsedText            string                 // 解析后的文本内容
	CharCount             int                    // 字符数量
	TokenCount            int                    // Token数量
	StructureLevel        int                    // 结构层级深度
	ContentQualityLevel   int                    // 内容质量等级
	HeadingCount          int                    // 标题数量
	ParagraphCount        int                    // 段落数量
	MaxParagraphLength    int                    // 最长段落长度
	ParserProviderName    string                 // 解析器提供者名称
	ParserProviderVersion string                 // 解析器版本号
	ParserCapabilities    []string               // 解析器支持的能力列表
	ParserElapsedMs       int                    // 解析耗时(毫秒)
	ParserWarnings        []string               // 解析警告信息
	ParserFailedReason    string                 // 解析失败原因
	ParserTraceMetadata   map[string]interface{} // 解析追踪元数据
	MarkdownSyntax        *MarkdownSyntax        // Markdown语法分析结果
	StructureNodes        []StructureNode        // 文档结构节点列表
	TableCandidates       []TableCandidate       // 表格候选列表
	ParseArtifacts        []ParseArtifact        // 解析产物列表
	Blocks                []ContentBlock         // 内容块列表
}
