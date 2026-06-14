package vo

type ChatSessionStatus = int

const (
	ChatSessionStatusIdle ChatSessionStatus = 1 + iota
	ChatSessionStatusRunning
)
