package vo

import (
	"path/filepath"
)

// FileType 文件类型
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
		return "未知"
	}
}
