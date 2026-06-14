package vo

type ChatExchangeStatus = int

const (
	ChatExchangeStatusRunning   ChatExchangeStatus = 1 + iota // 进行中
	ChatExchangeStatusCompleted                               // 已完成
	ChatExchangeStatusFailed                                  // 失败
	ChatExchangeStatusStopped                                 // 已停止
)
