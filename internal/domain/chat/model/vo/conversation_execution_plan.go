package vo

import (
	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/enum"
)

// ConversationExecutionPlan 对话执行计划（聚合根的一部分）
// 用于存储一次对话执行前的完整计划信息，包括查询改写、检索策略、历史压缩等。
type ConversationExecutionPlan struct {
	ChatMode                     enum.ChatQueryMode          // 对话模式/查询模式
	OriginalQuestion             string                      // 原始问题
	AgentQuestion                string                      // 代理问题
	RewriteQuestion              string                      // 问题改写结果
	RewriteSubQuestions          []string                    // 问题改写子问题列表
	HistorySummary               string                      // 历史摘要
	LongTermSummary              string                      // 长期摘要
	HistoryPlanningContext       *HistoryPlanningContext     // 历史规划上下文
	RecentHistoryTranscript      string                      // 最近历史记录转录
	RecentQuestionTranscript     string                      // 最近问题转录
	QuestionHistoryContext       *QuestionHistoryContext     // 问题历史上下文
	NavigationDecision           *DocumentNavigationDecision // 导航决策
	RecognitionResult            *IntentRecognitionResult    // 意图识别结果
	RetrievalPlan                *RetrievalPlan              // 检索执行计划
	RetrievalResult              *RetrievalResult            // 检索结果
	PromptAssemblyResult         *RagPromptAssemblyResult    // 提示组装结果
	HistoryCompressionApplied    bool                        // 是否应用历史压缩
	HistoryCoveredExchangeId     int64                       // 覆盖的历史记录交换ID
	HistoryCoveredExchangeCount  int                         // 覆盖的历史记录交换计数
	HistoryCompressionCount      int                         // 历史压缩计数
	RecentEvidenceAnchors        EvidenceAnchors             // 最近证据锚点
	CurrentDateText              string                      // 当前日期文本表示
	RequiresRealTimeSearch       bool                        // 是否需要实时搜索
	RequiresCurrentDateAnchoring bool                        // 是否需要当前日期锚定
	ClarificationReply           string                      // 澄清回复
	ClarificationOptions         []string                    // 澄清选项列表
	ClarificationReason          string                      // 澄清原因文本
}

// HasRetrievalQuestion 是否有检索问题
func (p *ConversationExecutionPlan) HasRetrievalQuestion() bool {
	if p == nil || p.RetrievalPlan == nil {
		return false
	}
	questionPlan := p.RetrievalPlan.QuestionPlan
	return questionPlan != nil && utils.IsNotBlank(questionPlan.RetrievalQuestion) && questionPlan.RetrievalQuestion != p.OriginalQuestion
}

// HasHistoryContext 是否有历史上下文
func (p *ConversationExecutionPlan) HasHistoryContext() bool {
	if p == nil || p.QuestionHistoryContext == nil {
		return false
	}
	return utils.IsNotBlank(p.QuestionHistoryContext.RenderedText)
}

func (p *ConversationExecutionPlan) SubQuestions() []string {
	if p == nil || p.RetrievalPlan == nil || p.RetrievalPlan.QuestionPlan == nil {
		return nil
	}
	return p.RetrievalPlan.QuestionPlan.SubQuestions
}
