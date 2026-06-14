package vo

type HistoryPlanningContext struct {
	ConversationGoal  string
	StableFacts       []string
	PendingQuestions  []string
	RetrievalHints    []string
	QueryContextHints []string
}
