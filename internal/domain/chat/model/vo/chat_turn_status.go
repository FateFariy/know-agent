package vo

type ChatTurnStatus = int

const (
	ChatTurnStatusRunning   ChatTurnStatus = 1 + iota // 进行中
	ChatTurnStatusCompleted                           // 已完成
	ChatTurnStatusFailed                              // 失败
	ChatTurnStatusStopped                             // 已停止
)
