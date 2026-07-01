package vo

import "time"

// ConversationExecutionPlan 对话执行计划（聚合根的一部分）
// 用于存储一次对话执行前的完整计划信息，包括查询改写、检索策略、历史压缩等。
type ConversationExecutionPlan struct {
	Mode                         ExecutionMode
	ChatMode                     ChatQueryMode
	OriginalQuestion             string
	AgentQuestion                string
	RewriteQuestion              string
	RewriteSubQuestions          []string
	RetrievalQuestion            string
	RetrievalSubQuestions        []string
	HistorySummary               string
	LongTermSummary              string
	HistoryPlanningContext       *HistoryPlanningContext
	RecentHistoryTranscript      string
	RecentQuestionTranscript     string
	QuestionHistoryContext       *QuestionHistoryContext
	NavigationDecision           *DocumentNavigationDecision
	HistoryCompressionApplied    bool
	HistoryCoveredExchangeId     *int64
	HistoryCoveredExchangeCount  *int
	HistoryCompressionCount      *int
	CurrentDate                  time.Time
	CurrentDateText              string
	RequiresFreshSearch          bool
	RequiresCurrentDateAnchoring bool
	SelectedDocumentId           int64
	SelectedDocumentName         string
	SelectedTaskId               int64
	RetrievalDocumentIds         []int64
	RetrievalTaskIds             []int64
	ClarificationReply           string
	ClarificationOptions         []string
	ClarificationReason          string
	NoEvidenceReply              string
}
