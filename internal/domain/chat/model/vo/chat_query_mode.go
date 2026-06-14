package vo

// ChatQueryMode 提问模式枚举
type ChatQueryMode = int

const (
	DocumentMode     ChatQueryMode = iota + 1 // 当前文档问答
	OpenChatMode                              // 开放式提问
	AutoDocumentMode                          // 自动知识问答
)

var ChatQueryModeMap = map[string]int{
	"document":      DocumentMode,
	"open_chat":     OpenChatMode,
	"auto_document": AutoDocumentMode,
}
