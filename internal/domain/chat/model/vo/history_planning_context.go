package vo

import "github.com/swiftbit/know-agent/internal/domain/chat/model/entity"

type HistoryPlanningContext struct {
	ConversationGoal  string   // 对话目标
	StableFacts       []string // 稳定事实
	PendingQuestions  []string // 待处理问题
	RetrievalHints    []string // 检索提示
	QueryContextHints []string // 查询上下文提示
}

func NewHistoryPlanningContext(summary *entity.ConversationSummary) *HistoryPlanningContext {
	if summary == nil {
		return &HistoryPlanningContext{}
	}
	return &HistoryPlanningContext{
		ConversationGoal:  summary.ConversationGoal,
		StableFacts:       append([]string{}, summary.StableFacts...),
		PendingQuestions:  append([]string{}, summary.PendingQuestions...),
		RetrievalHints:    append([]string{}, summary.RetrievalHints...),
		QueryContextHints: append([]string{}, summary.RetrievalHints...),
	}
}
