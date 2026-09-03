package middleware

import (
	"context"
	"strings"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/conversation"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/aggregate"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/enum"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
	"github.com/swiftbit/know-agent/internal/svc"
)

// MemoryLoadMiddleware 装载会话记忆并组装初始执行计划，同时把
// 记忆摘要与时效/实时检索规则判断注入 agent 系统指令
type MemoryLoadMiddleware struct {
	conversation.BaseAgentMiddleware
	memoryManager           conversation.SessionMemoryManager
	planningHistoryMaxChars int // 规划历史最大字符数
}

// NewMemoryLoadMiddleware 创建记忆装载中间件
func NewMemoryLoadMiddleware(svcCtx *svc.ServiceContext, memoryManager conversation.SessionMemoryManager) *MemoryLoadMiddleware {
	return &MemoryLoadMiddleware{
		memoryManager:           memoryManager,
		planningHistoryMaxChars: svcCtx.Config.Chat.Rag.PlanningHistoryMaxChars,
	}
}

// Name 中间件名称
func (m *MemoryLoadMiddleware) Name() string { return "memory-load" }

// BeforeAgent 装载记忆 → 组装执行计划 → 追加指令提示
func (m *MemoryLoadMiddleware) BeforeAgent(ctx context.Context, convCtx *conversation.Context, input *conversation.BeforeAgentInput) (*conversation.BeforeAgentOutput, error) {
	if convCtx == nil {
		return &conversation.BeforeAgentOutput{Instruction: input.Instruction}, nil
	}

	ctx = vo.OnStart(ctx, enum.ConversationTraceStageMemory, &vo.StageInput{SummaryText: "正在装载会话记忆与最近窗口。"})

	history, err := m.load(ctx, convCtx)
	if err != nil {
		ctx = vo.OnError(ctx, "会话记忆装载失败。", err)
		return &conversation.BeforeAgentOutput{Instruction: input.Instruction}, err
	}
	ctx = vo.OnEnd(ctx, &vo.StageOutput{SummaryText: "会话记忆装载完成。", Snapshot: history.BuildSnapshot()})

	instruction := input.Instruction
	if hints := m.buildInstructionHints(convCtx); len(hints) > 0 {
		if instruction != "" && !strings.HasSuffix(instruction, "\n") {
			instruction += "\n"
		}
		instruction += strings.Join(hints, "\n")
	}
	return &conversation.BeforeAgentOutput{Instruction: instruction}, nil
}

// load 装载会话记忆（长期摘要、近期转录、压缩信息），并组装初始执行计划
func (m *MemoryLoadMiddleware) load(ctx context.Context, convCtx *conversation.Context) (*aggregate.Conversation, error) {
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

	// 组装初始执行计划（所有检索/改写字段先以原始问题为兜底值，防止空值扩散）todo 这部分后续可能删除
	execPlan := &vo.ConversationExecutionPlan{
		ChatMode:                    convCtx.ChatMode,
		OriginalQuestion:            convCtx.Question,
		RewriteQuestion:             convCtx.Question,
		RewriteSubQuestions:         []string{convCtx.Question},
		HistorySummary:              historySummary,
		LongTermSummary:             memoryContext.LongTermSummary,
		HistoryPlanningContext:      historyPlanningContext,
		RecentHistoryTranscript:     memoryContext.RecentTranscript,
		RecentQuestionTranscript:    memoryContext.RecentQuestionTranscript,
		QuestionHistoryContext:      questionHistoryContext,
		HistoryCompressionApplied:   memoryContext.CompressionApplied,
		HistoryCoveredExchangeId:    memoryContext.CoveredExchangeId,
		HistoryCoveredExchangeCount: memoryContext.CoveredExchangeCount,
		HistoryCompressionCount:     memoryContext.CompressionCount,
		CurrentDateText:             convCtx.CurrentDateText,
	}
	convCtx.SetExecutePlan(execPlan)
	return memoryContext, nil
}

// buildInstructionHints 组装待注入指令的动态提示
func (m *MemoryLoadMiddleware) buildInstructionHints(convCtx *conversation.Context) []string {
	execPlan := convCtx.ExecutionPlan.Load()
	if execPlan == nil {
		return nil
	}
	question := utils.Trim(convCtx.Question)
	if question == "" {
		return nil
	}

	analyzer := conversation.NewQueryAnalyzer(question)
	requiresCurrentDateAnchoring := analyzer.RequiresCurrentDateAnchoring()
	requiresRealTimeSearch := analyzer.RequiresRealTimeSearch()
	execPlan.RequiresRealTimeSearch = requiresRealTimeSearch
	execPlan.RequiresCurrentDateAnchoring = requiresCurrentDateAnchoring

	var hints []string
	if utils.IsNotBlank(convCtx.CurrentDateText) {
		hints = append(hints, "当前日期："+convCtx.CurrentDateText+"。")
	}
	if requiresCurrentDateAnchoring {
		hints = append(hints, "当前问题包含相对时间或强时效表达（如“今天、明天、现在、最新、本周、本月”等），涉及日期必须以此为准，不要用检索结果中的旧日期替代今天。")
	}
	if requiresRealTimeSearch {
		hints = append(hints, "当前问题需要核实最新信息，检索证据不足或日期滞后时请明确说明不确定性，不要编造。")
	}
	if utils.IsNotBlank(execPlan.HistorySummary) {
		hints = append(hints, "相关会话背景：\n"+execPlan.HistorySummary)
	}
	return hints
}
