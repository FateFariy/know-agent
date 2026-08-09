package vo

import (
	"slices"

	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/document/model/enum"
)

// AnalysisResult 文档分析结果
type AnalysisResult struct {
	ParsedText            string                  // 解析后的文本内容
	CharCount             int                     // 字符数量
	TokenCount            int                     // Token数量
	StructureLevel        int                     // 结构层级深度
	ContentQualityLevel   int                     // 内容质量等级
	HeadingCount          int                     // 标题数量
	ParagraphCount        int                     // 段落数量
	MaxParagraphLength    int                     // 最长段落长度
	ParserProviderName    string                  // 解析器提供者名称
	ParserProviderVersion string                  // 解析器版本号
	ParserCapabilities    []string                // 解析器支持的能力列表
	ParserElapsedMs       int                     // 解析耗时(毫秒)
	ParserWarnings        []string                // 解析警告信息
	ParserFailedReason    string                  // 解析失败原因
	ParserTraceMetadata   map[string]any          // 解析追踪元数据
	MarkdownSyntax        *MarkdownSyntax         // Markdown语法分析结果
	StructureNodes        []*entity.StructureNode // 文档结构节点列表
	TableCandidates       []*TableCandidate       // 表格候选列表
	ParseArtifacts        []*entity.ParseArtifact // 解析产物列表
	Blocks                []*entity.DocumentBlock // 内容块列表
}

// ShouldUseStructure 是否启用结构切块，启用条件：文件类型被识别 +（结构等级达到中等或标题数≥2）
func (a *AnalysisResult) ShouldUseStructure(fileType enum.FileType) bool {
	if fileType == enum.FileTypeXLSX {
		return true
	}
	types := []enum.FileType{enum.FileTypePDF, enum.FileTypeDOCX, enum.FileTypeDOC, enum.FileTypeMD, enum.FileTypeHTML, enum.FileTypeTXT}
	return slices.Contains(types, fileType) && (a.StructureLevel >= enum.StructureLevelMedium || a.HeadingCount >= 2)
}

// ShouldUseRecursive 是否启用递归切块，启用条件：文本总长度或最长段落长度 ≥ 递归窗口上限（需要控制单次块大小）
func (a *AnalysisResult) ShouldUseRecursive(maxChars int) bool {
	return a.CharCount >= maxChars || a.MaxParagraphLength > maxChars
}

// ShouldUseSemantic 是否启用语义切块，启用条件：文本总长度 ≥ 语义切块最小字符数 + 段落数量 ≥ 3 + 内容质量等级 ≥ 中等
func (a *AnalysisResult) ShouldUseSemantic(minChars int) bool {
	return a.CharCount >= minChars && a.ParagraphCount >= 3 && a.ContentQualityLevel >= enum.ContentQualityLevelMedium
}

// ShouldUseLlm 是否启用LLM切块，启用条件：文本总长度 ≥ LLM切块最小字符数 + 内容质量等级 = 低
func (a *AnalysisResult) ShouldUseLlm(minChars int) bool {
	return a.CharCount >= minChars && a.ContentQualityLevel == enum.ContentQualityLevelLow
}
