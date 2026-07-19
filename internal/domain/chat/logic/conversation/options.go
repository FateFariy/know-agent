package conversation

type options struct {
	historyPreviewTurns    int // 预览历史轮次
	maxModelCallsPerRun    int // 每次最大模型调用次数
	maxModelCallsPerThread int // 每次最大模型调用线程数
	maxToolCallsPerRun     int // 每次最大工具调用次数
	maxToolCallsPerThread  int // 每次最大工具调用线程数
}

type Option func(*options)

// WithHistoryPreviewTurns 设置历史预览轮数
func WithHistoryPreviewTurns(turns int) Option {
	return func(o *options) {
		o.historyPreviewTurns = max(turns, 1)
	}
}
