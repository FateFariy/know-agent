package vo

import "github.com/duke-git/lancet/v2/enum"

// ============================================================
// ChatQueryMode 提问模式枚举
// ============================================================

type ChatQueryMode = *enum.Item[int]

const (
	DocumentMode     = iota + 1 // 当前文档问答
	OpenChatMode                // 开放式提问
	AutoDocumentMode            // 自动知识问答
)

var (
	ChatQueryModeDocument     = enum.NewItem(DocumentMode, "document")
	ChatQueryModeOpenChat     = enum.NewItem(OpenChatMode, "open_chat")
	ChatQueryModeAutoDocument = enum.NewItem(AutoDocumentMode, "auto_document")
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
		return enum.NewItem(-1, "unknown")
	}
}

func ChatQueryModeValue(mode ChatQueryMode) int {
	if mode == nil {
		return 0
	}
	return mode.Value()
}
