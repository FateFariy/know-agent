package vo

// ============================================================
// ChatSessionStatus 会话状态
// ============================================================

type ChatSessionStatus = int

const (
	ChatSessionStatusIdle ChatSessionStatus = 1 + iota
	ChatSessionStatusRunning
)

// ============================================================
// ChatStage 聊天阶段
// ============================================================

type ChatStage = string

const (
	ChatStageRewrite   ChatStage = "rewrite"
	ChatStageRagAnswer ChatStage = "rag_answer"
	ChatStageSummary   ChatStage = "summary"
	ChatStageRecommend ChatStage = "recommend"
)

// ============================================================
// ChatTurnStatus 对话轮次状态
// ============================================================

type ChatTurnStatus = int

const (
	ChatTurnStatusRunning   ChatTurnStatus = 1 + iota // 进行中
	ChatTurnStatusCompleted                           // 已完成
	ChatTurnStatusFailed                              // 失败
	ChatTurnStatusStopped                             // 已停止
)

func ChatTurnStatusName(code int) string {
	switch code {
	case ChatTurnStatusRunning:
		return "进行中"
	case ChatTurnStatusCompleted:
		return "已完成"
	case ChatTurnStatusFailed:
		return "失败"
	case ChatTurnStatusStopped:
		return "已停止"
	}
	return ""
}

func ToChatTurnStatus(name string) ChatTurnStatus {
	switch name {
	case "进行中":
		return ChatTurnStatusRunning
	case "已完成":
		return ChatTurnStatusCompleted
	case "失败":
		return ChatTurnStatusFailed
	case "已停止":
		return ChatTurnStatusStopped
	}
	return 0
}

// ============================================================
// ConversationTraceStageState 追踪阶段状态
// ============================================================

type ConversationTraceStageState = int

const (
	ConversationTraceStageStateRunning   ConversationTraceStageState = 1 + iota // 进行中
	ConversationTraceStageStateCompleted                                        // 已完成
	ConversationTraceStageStateFailed                                           // 失败
	ConversationTraceStageStateSkipped                                          // 跳过
)

func ConversationTraceStageStateName(code int) string {
	switch code {
	case ConversationTraceStageStateRunning:
		return "进行中"
	case ConversationTraceStageStateCompleted:
		return "已完成"
	case ConversationTraceStageStateFailed:
		return "失败"
	case ConversationTraceStageStateSkipped:
		return "跳过"
	}
	return ""
}
