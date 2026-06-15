package vo

type ConversationSummary struct {
	Summary          string   // 摘要
	ConversationGoal string   // 会话目标
	StableFacts      []string // 稳定事实
	UserPreferences  []string // 用户偏好
	ResolvedPoints   []string // 解决的点
	PendingQuestions []string // 待解决的问题
	RetrievalHints   []string // 检索提示
}
