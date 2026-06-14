package vo

type ConversationStop struct {
	ConversationId string // 对话ID
	Stopped        bool   // 是否停止
	Message        string // 提示信息
}
