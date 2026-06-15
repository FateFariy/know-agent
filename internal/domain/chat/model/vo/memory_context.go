package vo

// MemoryContext 存储会话的摘要、近期对话及压缩状态信息
type MemoryContext struct {
	AssembledHistory         string              // 长期摘要 + 最近窗口
	LongTermSummary          string              // 长期摘要，总结整个会话的主题和关键信息
	RecentTranscript         string              // 最近窗口（含问题和回答）
	QuestionRecentTranscript string              // 最近窗口（只含问题）
	Summary                  ConversationSummary // 摘要
	IsCompressed             bool                // 是否已应用历史压缩
	CoveredExchangeId        int64               // 压缩覆盖到的最后一轮对话ID
	CoveredExchangeCount     int                 // 压缩覆盖的对话轮次数
	CompressionCount         int                 // 历史压缩的总次数
}
