package conversation

import (
	"context"

	"github.com/duke-git/lancet/v2/strutil"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/chat/adapter"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/intent"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/memory"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/aggregate"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/enum"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
	"github.com/swiftbit/know-agent/internal/svc"
)

const (
	maxSnippetChars    = 300
	maxRecentExchanges = 3
)

type MemoryLoadStage struct {
	repo                    adapter.ChatRepository
	memoryManager           memory.SessionMemoryManager
	planningHistoryMaxChars int    // 规划历史最大字符数
	noEvidenceReply         string // 无证据回复
}

func NewMemoryLoadStage(svcCtx *svc.ServiceContext, repo adapter.ChatRepository, m memory.SessionMemoryManager) *MemoryLoadStage {
	noEvidenceReply := svcCtx.Config.Chat.Rag.NoEvidenceReply
	noEvidenceReply = utils.BlankToDefault(noEvidenceReply, "当前没有从已接入文档中检索到足够证据，暂时不能给出可靠结论。")
	return &MemoryLoadStage{
		repo:                    repo,
		memoryManager:           m,
		planningHistoryMaxChars: svcCtx.Config.Chat.Rag.PlanningHistoryMaxChars,
		noEvidenceReply:         noEvidenceReply,
	}
}

func (m *MemoryLoadStage) Name() string {
	return enum.ConversationTraceStageMemory.Name
}

func (m *MemoryLoadStage) Execute(ctx context.Context, convCtx *Context) error {
	ctx = vo.OnStart(ctx, enum.ConversationTraceStageMemory, enum.ChatQueryModeName(convCtx.ChatMode), &vo.StageInput{SummaryText: "正在装载会话记忆与最近窗口。"})

	history, err := m.summarizeHistory(ctx, convCtx)
	if err != nil {
		ctx = vo.OnError(ctx, "会话记忆装载失败。", err)
		return err
	}
	ctx = vo.OnEnd(ctx, &vo.StageOutput{SummaryText: "会话记忆装载完成。", Snapshot: history.BuildSnapshot()})

	return nil
}

// summarizeHistory 构建会话记忆，装载长期摘要与近期转录（含压缩状态）
func (m *MemoryLoadStage) summarizeHistory(ctx context.Context, convCtx *Context) (*aggregate.Conversation, error) {
	question := strutil.Trim(convCtx.Question)

	// 装载会话记忆（含长期摘要、近期转录、压缩信息）
	memoryContext, err := m.memoryManager.LoadMemoryContext(ctx, convCtx.ConversationId)
	if err != nil {
		return nil, err
	}

	// 聚合长期记忆，供 Agent 做规划引用
	historyPlanningContext := vo.NewHistoryPlanningContext(memoryContext.Summary)
	// 以可读文本形式描述规划历史
	historySummary := historyPlanningContext.BuildPlanningText(memoryContext.RecentTranscript, m.planningHistoryMaxChars)
	// 当前问题 + 近期对话转录，用于后续改写与检索
	recentQuestions := utils.Trim(memoryContext.RecentQuestionTranscript)
	questionHistoryContext := vo.NewQuestionHistoryContext(recentQuestions, m.planningHistoryMaxChars)

	// 判断时间敏感与实时搜索需求（关键词规则判断）
	analyzer := intent.NewQueryAnalyzer(question)
	requiresCurrentDateAnchoring := analyzer.RequiresCurrentDateAnchoring()
	requiresRealTimeSearch := analyzer.RequiresRealTimeSearch()

	// 组装初始执行计划（所有检索/改写字段先以原始问题为兜底值，防止空值扩散）
	execPlan := &vo.ConversationExecutionPlan{
		ChatMode:                     convCtx.ChatMode,
		OriginalQuestion:             convCtx.Question,
		AgentQuestion:                convCtx.Question,
		RewriteQuestion:              convCtx.Question,
		RewriteSubQuestions:          []string{convCtx.Question},
		RetrievalQuestion:            convCtx.Question,
		RetrievalSubQuestions:        []string{convCtx.Question},
		HistorySummary:               historySummary,
		LongTermSummary:              memoryContext.LongTermSummary,
		HistoryPlanningContext:       historyPlanningContext,
		RecentHistoryTranscript:      memoryContext.RecentTranscript,
		RecentQuestionTranscript:     memoryContext.RecentQuestionTranscript,
		QuestionHistoryContext:       questionHistoryContext,
		HistoryCompressionApplied:    memoryContext.CompressionApplied,
		HistoryCoveredExchangeId:     memoryContext.CoveredExchangeId,
		HistoryCoveredExchangeCount:  memoryContext.CoveredExchangeCount,
		HistoryCompressionCount:      memoryContext.CompressionCount,
		CurrentDate:                  convCtx.CurrentDate,
		CurrentDateText:              convCtx.CurrentDateText,
		RequiresRealTimeSearch:       requiresRealTimeSearch,
		RequiresCurrentDateAnchoring: requiresCurrentDateAnchoring,
		NoEvidenceReply:              m.noEvidenceReply,
	}
	convCtx.SetExecutePlan(execPlan)

	return memoryContext, nil
}

// loadRecentEvidenceAnchors 加载最近的证据锚点，从对话历史中抽取追问可继承的结构锚点
func (m *MemoryLoadStage) loadRecentEvidenceAnchors(ctx context.Context, conversationId string, limit int) ([]*vo.EvidenceAnchor, error) {
	if conversationId == "" || limit <= 0 {
		return nil, nil
	}

	exchanges, err := m.repo.ListRecentExchanges(ctx, conversationId, maxRecentExchanges)
	if err != nil || len(exchanges) == 0 {
		return nil, err
	}

	var anchors []*vo.EvidenceAnchor
	for _, exchange := range exchanges {
		if exchange == nil || !exchange.IsCompleted() || len(exchange.References) == 0 {
			continue
		}
		for _, ref := range exchange.ParseReferences() {
			anchor := ref.ToEvidenceAnchor(maxSnippetChars)
			if anchor == nil {
				continue
			}
			anchors = append(anchors, anchor)
			if len(anchors) >= limit {
				return anchors, nil
			}
		}
	}
	return anchors, nil
}
