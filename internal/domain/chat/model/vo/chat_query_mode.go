package vo

import (
	"fmt"
)

// ChatQueryMode 提问模式枚举
type ChatQueryMode = int

const (
	DocumentMode     ChatQueryMode = iota + 1 // 当前文档问答
	OpenChatMode                              // 开放式提问
	AutoDocumentMode                          // 自动知识问答
)

func ChatQueryModeName(code int) string {
	switch code {
	case DocumentMode:
		return "document"
	case OpenChatMode:
		return "open_chat"
	case AutoDocumentMode:
		return "auto_document"
	default:
		return fmt.Sprintf("未知模式(%d)", code)
	}
}
