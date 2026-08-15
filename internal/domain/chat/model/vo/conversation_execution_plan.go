package vo

import (
	"errors"
	"time"

	"github.com/swiftbit/know-agent/internal/domain/chat/model/enum"
)

// ConversationExecutionPlan 对话执行计划（聚合根的一部分）
// 用于存储一次对话执行前的完整计划信息，包括查询改写、检索策略、历史压缩等。
type ConversationExecutionPlan struct {
	Mode                         enum.ExecutionMode          // 执行模式
	ChatMode                     enum.ChatQueryMode          // 对话模式/查询模式
	OriginalQuestion             string                      // 原始问题
	AgentQuestion                string                      // 代理问题
	RewriteQuestion              string                      // 问题改写结果
	RewriteSubQuestions          []string                    // 问题改写子问题列表
	RetrievalQuestion            string                      // 检索问题(Deprecated）
	RetrievalSubQuestions        []string                    // 检索子问题列表(Deprecated）
	HistorySummary               string                      // 历史摘要
	LongTermSummary              string                      // 长期摘要
	HistoryPlanningContext       *HistoryPlanningContext     // 历史规划上下文
	RecentHistoryTranscript      string                      // 最近历史记录转录
	RecentQuestionTranscript     string                      // 最近问题转录
	QuestionHistoryContext       *QuestionHistoryContext     // 问题历史上下文
	NavigationDecision           *DocumentNavigationDecision // 导航决策
	RecognitionResult            *IntentRecognitionResult    // 意图识别结果
	RetrievalPlan                *RetrievalPlan              // 检索执行计划
	HistoryCompressionApplied    bool                        // 是否应用历史压缩
	HistoryCoveredExchangeId     int64                       // 覆盖的历史记录交换ID
	HistoryCoveredExchangeCount  int                         // 覆盖的历史记录交换计数
	HistoryCompressionCount      int                         // 历史压缩计数
	CurrentDate                  time.Time                   // 当前日期
	CurrentDateText              string                      // 当前日期文本表示
	RequiresRealTimeSearch       bool                        // 是否需要实时搜索
	RequiresCurrentDateAnchoring bool                        // 是否需要当前日期锚定
	SelectedDocumentId           int64                       // 选中的文档ID(Deprecated）
	SelectedDocumentName         string                      // 选中的文档名称(Deprecated）
	SelectedTaskId               int64                       // 选中的任务ID(Deprecated）
	RetrievalDocumentIds         []int64                     // 检索文档ID列表(Deprecated）
	RetrievalTaskIds             []int64                     // 检索任务ID列表(Deprecated）
	ClarificationReply           string                      // 澄清回复
	ClarificationOptions         []string                    // 澄清选项列表
	ClarificationReason          string                      // 澄清原因文本
	NoEvidenceReply              string                      // 无证据回复文本
	ScopedEvidenceAnchors        []*EvidenceAnchor           // 作用域内的证据锚点
}

func (p *ConversationExecutionPlan) Validate() error {
	if p == nil {
		return errors.New("conversation execution plan is nil")
	}
	if p.RetrievalPlan == nil {
		return errors.New("retrieval plan is nil")
	}
	return p.RetrievalPlan.ValidateForExecution()
}

// ExecutionModeName 获取执行模式名称
func (p *ConversationExecutionPlan) ExecutionModeName() string {
	if p.Mode == nil {
		return ""
	}
	return p.Mode.Name()
}

func (p *ConversationExecutionPlan) BuildRetrievalPlanSnapshot() map[string]any {
	if p == nil {
		return nil
	}
	return map[string]any{
		"mode":              p.ExecutionModeName(),
		"retrievalQuestion": p.RetrievalQuestion,
		"subQuestions":      p.RetrievalSubQuestions,
		"selectedDocument":  p.SelectedDocumentName,
	}
}
