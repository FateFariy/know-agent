package enum

// ============================================================
// ChatQueryMode 提问模式枚举
// ============================================================

// 枚举值以 int 形式持久化在 chat_dialogue / chat_cache_entry，并作为语义缓存的隔离维度，
// 因此取值固定不复用：2 为已下线的开放式提问（open_chat），保留空档避免历史数据错位。
type ChatQueryMode = int

const (
	ChatQueryModeDocument     = 1 // 当前文档问答
	ChatQueryModeAutoDocument = 3 // 自动知识问答
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
