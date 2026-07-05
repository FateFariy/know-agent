package vo

type ConversationStop struct {
	ConversationId string // 对话ID
	Stopped        bool   // 是否停止
	Message        string // 提示信息
}

// ConversationReset 会话重置
type ConversationReset struct {
	ConversationId         string `json:"conversationId"`         // 会话ID
	StoppedRunningTask     bool   `json:"stoppedRunningTask"`     // 是否停止正在运行的任务
	RemovedDialogueCount   int    `json:"removedDialogueCount"`   // 移除对话轮次数量
	RemovedExchangeCount   int    `json:"removedExchangeCount"`   // 移除交互记录数量
	RemovedCheckpointCount int    `json:"removedCheckpointCount"` // 移除检查点数量
	Message                string `json:"message"`                // 提示信息
}
