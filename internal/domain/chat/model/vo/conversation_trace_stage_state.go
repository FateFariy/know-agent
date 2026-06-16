package vo

type ConversationTraceStageState = int

const (
	ConversationTraceStageStateRunning   ConversationTraceStageState = 1 + iota // 进行中
	ConversationTraceStageStateCompleted                                        // 已完成
	ConversationTraceStageStateFailed                                           // 失败
	ConversationTraceStageStateSkipped                                          // 跳过
)

var conversationTraceStageStateMap = map[int]string{
	ConversationTraceStageStateRunning:   "进行中",
	ConversationTraceStageStateCompleted: "已完成",
	ConversationTraceStageStateFailed:    "失败",
	ConversationTraceStageStateSkipped:   "跳过",
}

func ConversationTraceStageStateFromCode(code int) ConversationTraceStageState {
	if code == 0 {
		return ConversationTraceStageStateRunning
	}
	if _, ok := conversationTraceStageStateMap[code]; ok {
		return code
	}
	return ConversationTraceStageStateRunning
}
