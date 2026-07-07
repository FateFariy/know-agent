package vo

// ============================================================
// ChatQueryMode 提问模式枚举
// ============================================================

type ChatQueryMode = int

const (
	ChatQueryModeDocument     = iota + 1 // 当前文档问答
	ChatQueryModeOpenChat                // 开放式提问
	ChatQueryModeAutoDocument            // 自动知识问答
)

func ToChatQueryMode(name string) ChatQueryMode {
	switch name {
	case "document":
		return ChatQueryModeDocument
	case "open_chat":
		return ChatQueryModeOpenChat
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
	case ChatQueryModeOpenChat:
		return "open_chat"
	case ChatQueryModeAutoDocument:
		return "auto_document"
	}
	return ""
}
