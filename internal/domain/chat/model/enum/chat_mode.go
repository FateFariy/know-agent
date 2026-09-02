package enum

// ============================================================
// ChatQueryMode 提问模式枚举
// ============================================================

type ChatQueryMode = int

const (
	ChatQueryModeDocument     = 1 // 当前文档问答
	ChatQueryModeAutoDocument = 2 // 自动知识问答
)

func ToChatQueryMode(name string) ChatQueryMode {
	switch name {
	case "document":
		return ChatQueryModeDocument
	case "auto_document":
		return ChatQueryModeAutoDocument
	default:
		return 0
	}
}

func ChatQueryModeName(code int) string {
	switch code {
	case ChatQueryModeDocument:
		return "document"
	case ChatQueryModeAutoDocument:
		return "auto_document"
	}
	return ""
}
