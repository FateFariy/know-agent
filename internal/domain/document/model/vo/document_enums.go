package vo

import (
	"path/filepath"

	"github.com/duke-git/lancet/v2/strutil"
)

// ============================================================
// FileType 文件类型
// ============================================================

type FileType = int

const (
	FileTypeUnknown FileType = iota
	FileTypePDF
	FileTypeDOC
	FileTypeDOCX
	FileTypeTXT
	FileTypeMD
	FileTypeHTML
)

func DetectFileType(fileName string) FileType {
	switch filepath.Ext(fileName)[1:] {
	case "pdf":
		return FileTypePDF
	case "docx":
		return FileTypeDOCX
	case "doc":
		return FileTypeDOC
	case "txt":
		return FileTypeTXT
	case "md":
		return FileTypeMD
	case "html", "htm":
		return FileTypeHTML
	default:
		return FileTypeUnknown
	}
}

func FileTypeName(fileType FileType) string {
	switch fileType {
	case FileTypePDF:
		return "PDF"
	case FileTypeDOCX:
		return "DOCX"
	case FileTypeDOC:
		return "DOC"
	case FileTypeTXT:
		return "TXT"
	case FileTypeMD:
		return "MD"
	case FileTypeHTML:
		return "HTML"
	default:
		return ""
	}
}

// ============================================================
// ParseStatus 解析状态
// ============================================================

type ParseStatus = int

const (
	ParseStatusParsing      ParseStatus = iota + 1 // 解析中
	ParseStatusParseSuccess                        // 解析成功
	ParseStatusParseFailed                         // 解析失败
)

func ParseStatusName(statusName ParseStatus) string {
	switch statusName {
	case ParseStatusParsing:
		return "解析中"
	case ParseStatusParseSuccess:
		return "解析成功"
	case ParseStatusParseFailed:
		return "解析失败"
	default:
		return ""
	}
}

// ============================================================
// IndexStatus 索引状态
// ============================================================

type IndexStatus = int

const (
	IndexStatusWaitBuild    IndexStatus = iota + 1 // 待构建
	IndexStatusBuilding                            // 构建中
	IndexStatusBuildSuccess                        // 构建成功
	IndexStatusBuildFailed                         // 构建失败
)

func IndexStatusName(status IndexStatus) string {
	switch status {
	case IndexStatusWaitBuild:
		return "待构建"
	case IndexStatusBuilding:
		return "构建中"
	case IndexStatusBuildSuccess:
		return "构建成功"
	case IndexStatusBuildFailed:
		return "构建失败"
	default:
		return ""
	}
}

// ============================================================
// ContentQualityLevel 文档内容质量等级
// ============================================================

type ContentQualityLevel = int

const (
	ContentQualityLevelLow    ContentQualityLevel = iota + 1 // 低质量
	ContentQualityLevelMedium                                // 中质量
	ContentQualityLevelHigh                                  // 高质量
)

func ContentQualityLevelName(level ContentQualityLevel) string {
	switch level {
	case ContentQualityLevelLow:
		return "低质量"
	case ContentQualityLevelMedium:
		return "中质量"
	case ContentQualityLevelHigh:
		return "高质量"
	default:
		return ""
	}
}

// ============================================================
// StructureLevel 文档结构等级
// ============================================================

type StructureLevel = int

const (
	StructureLevelLow    StructureLevel = iota + 1 // 低结构化
	StructureLevelMedium                           // 中结构化
	StructureLevelHigh                             // 高结构化
)

func StructureLevelName(level StructureLevel) string {
	switch level {
	case StructureLevelLow:
		return "低结构化"
	case StructureLevelMedium:
		return "中结构化"
	case StructureLevelHigh:
		return "高结构化"
	default:
		return ""
	}
}

// ============================================================
// DocumentChunkSourceType 文档切块来源类型
// ============================================================

type DocumentChunkSourceType = int

const (
	ChunkSourceTypeOriginal DocumentChunkSourceType = iota + 1 // 原文切块
	ChunkSourceTypeEnriched                                    // 后处理补全文本
)

func DocumentChunkSourceTypeName(sourceType DocumentChunkSourceType) string {
	switch sourceType {
	case ChunkSourceTypeOriginal:
		return "原文切块"
	case ChunkSourceTypeEnriched:
		return "后处理补全文本"
	default:
		return "未知"
	}
}

type DocumentType = string

// 文档类型
const (
	DocTypeFAQ          DocumentType = "faq"             // 常见问题
	DocTypeTroubleshoot DocumentType = "troubleshooting" // 故障排除
	DocTypeRule         DocumentType = "rule"            // 规则
	DocTypeSpec         DocumentType = "spec"            // 规范
	DocTypeManual       DocumentType = "manual"          // 手册
	DocTypeIntro        DocumentType = "intro"           // 介绍
)

// InferDocumentType 推断文档类型
func InferDocumentType(combinedText string, supportsItemLookup bool) string {
	if strutil.ContainsAny(combinedText, []string{"faq", "常见问题"}) {
		return DocTypeFAQ
	}
	if strutil.ContainsAny(combinedText, []string{"故障", "排查", "检查顺序"}) {
		return DocTypeTroubleshoot
	}
	if strutil.ContainsAny(combinedText, []string{"规则", "制度"}) {
		return DocTypeRule
	}
	if strutil.ContainsAny(combinedText, []string{"规格", "参数"}) {
		return DocTypeSpec
	}
	if supportsItemLookup || strutil.ContainsAny(combinedText, []string{"手册", "指南", "部署"}) {
		return DocTypeManual
	}
	return DocTypeIntro
}

func ExampleQuestion(docType, topic string) string {
	switch docType {
	case DocTypeTroubleshoot:
		return topic + "的可能原因有哪些？"
	case DocTypeManual:
		return topic + "的步骤是什么？"
	case DocTypeRule:
		return topic + "有哪些规则？"
	default:
		return topic + "是什么意思？"
	}
}

// InferBusinessCategory 推断业务分类
func InferBusinessCategory(docType DocumentType, parsedText string) string {
	switch docType {
	case DocTypeTroubleshoot:
		return "故障排查"
	case DocTypeRule:
		return "规则"
	case DocTypeSpec:
		return "规格说明"
	case DocTypeManual:
		if strutil.ContainsAny(parsedText, []string{"步骤", "操作", "部署"}) {
			return "操作手册"
		}
		return "手册"
	default:
		return "介绍"
	}
}
