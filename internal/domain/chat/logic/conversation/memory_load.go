package conversation

import (
	"context"
	"math"
	"strings"

	"github.com/duke-git/lancet/v2/stream"
	"github.com/duke-git/lancet/v2/strutil"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/memory"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/aggregate"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/enum"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
	"github.com/swiftbit/know-agent/internal/domain/chat/support"
	"github.com/swiftbit/know-agent/internal/svc"
)

type MemoryLoadStage struct {
	memoryManager           memory.SessionMemoryManager
	planningHistoryMaxChars int    // 规划历史最大字符数
	noEvidenceReply         string // 无证据回复
}

func NewMemoryLoadStage(svcCtx *svc.ServiceContext, m memory.SessionMemoryManager) *MemoryLoadStage {
	noEvidenceReply := svcCtx.Config.Chat.Rag.NoEvidenceReply
	noEvidenceReply = utils.BlankToDefault(noEvidenceReply, "当前没有从已接入文档中检索到足够证据，暂时不能给出可靠结论。")
	return &MemoryLoadStage{
		memoryManager:           m,
		planningHistoryMaxChars: svcCtx.Config.Chat.Rag.PlanningHistoryMaxChars,
		noEvidenceReply:         noEvidenceReply,
	}
}

func (m *MemoryLoadStage) Name() string {
	return enum.ConversationTraceStageMemory.Name
}

func (m *MemoryLoadStage) Execute(ctx context.Context, convCtx *Context) error {
	// 规范化原始问题
	question := strutil.Trim(convCtx.Question)

	// 装载会话记忆（含长期摘要、近期转录、压缩信息）
	memoryContext, err := m.summarizeHistory(ctx, convCtx)
	if err != nil {
		return err
	}

	// 构建历史规划上下文与问题历史上下文
	//  - historyPlanningContext：聚合长期记忆，供 Agent 做规划引用
	//  - historySummary：以可读文本形式描述规划历史
	//  - questionHistoryContext：当前问题 + 近期对话转录，用于后续改写与检索
	historyPlanningContext := vo.NewHistoryPlanningContext(memoryContext.Summary)
	historySummary := m.buildPlanningHistory(memoryContext, historyPlanningContext)
	questionHistoryContext := vo.NewQuestionHistoryContext(question, strutil.Trim(memoryContext.RecentTranscript), m.planningHistoryMaxChars)

	// 判断时间敏感与实时搜索需求（关键词规则判断，无外部调用）
	requiresCurrentDateAnchoring := support.RequiresCurrentDateAnchoring(question)
	requiresRealTimeSearch := support.RequiresRealTimeSearch(question)

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
	return nil
}

// summarizeHistory 构建会话记忆
//
// 执行步骤：
//  1. 启动记忆追踪阶段
//  2. 调用 memoryManager 装载长期摘要与近期转录（含压缩状态）
//  3. 失败时记录失败追踪并返回
//  4. 成功时写入快照（压缩状态、覆盖的 exchange、摘要内容），提交追踪后返回
func (m *MemoryLoadStage) summarizeHistory(ctx context.Context, convCtx *Context) (*aggregate.Conversation, error) {
	// 启动记忆追踪阶段，使用 chatMode 作为执行模式名
	ctx = vo.OnStart(ctx, enum.ConversationTraceStageMemory, enum.ChatQueryModeName(convCtx.ChatMode), &vo.StageInput{SummaryText: "正在装载会话记忆与最近窗口。"})

	// 调用 memoryManager 装载记忆上下文（含长期摘要、近期转录、压缩状态）
	memoryContext, err := m.memoryManager.LoadMemoryContext(ctx, convCtx.ConversationId)
	if err != nil {
		ctx = vo.OnError(ctx, "会话记忆装载失败。", err)
		return nil, err
	}
	// 写入快照（压缩状态、覆盖的 exchange 信息、长期/近期摘要）
	snapshot := map[string]any{
		"compressionApplied":       memoryContext.CompressionApplied,
		"coveredExchangeId":        memoryContext.CoveredExchangeId,
		"coveredExchangeCount":     memoryContext.CoveredExchangeCount,
		"compressionCount":         memoryContext.CompressionCount,
		"longTermSummary":          strutil.Trim(memoryContext.LongTermSummary),
		"recentTranscript":         strutil.Trim(memoryContext.RecentTranscript),
		"RecentQuestionTranscript": strutil.Trim(memoryContext.RecentQuestionTranscript),
	}
	// 提交记忆追踪阶段，成功后返回记忆上下文
	ctx = vo.OnEnd(ctx, &vo.StageOutput{SummaryText: "会话记忆装载完成。", Snapshot: snapshot})
	return memoryContext, nil
}

func (m *MemoryLoadStage) buildPlanningHistory(memoryContext *aggregate.Conversation, historyPlanningContext *vo.HistoryPlanningContext) string {
	// 拼接结构化历史（会话目标 + 三类要点提示）
	var sb strings.Builder
	m.appendSection(&sb, "会话目标", historyPlanningContext.ConversationGoal)
	m.appendBulletSection(&sb, "已确认事实", historyPlanningContext.StableFacts)
	m.appendBulletSection(&sb, "待跟进问题", historyPlanningContext.PendingQuestions)
	m.appendBulletSection(&sb, "检索提示", historyPlanningContext.RetrievalHints)
	structuredHistory := strutil.Trim(sb.String())
	recentTranscript := strutil.Trim(memoryContext.RecentTranscript)

	maxChars := m.planningHistoryMaxChars
	// 近期转录为空 → 仅返回结构化文本（ClipHead 保留开头，避免尾部上下文缺失）
	if recentTranscript == "" {
		return utils.ClipHead(structuredHistory, maxChars)
	}

	// 按 65% 预算切分近期转录（ClipTail 保留末尾最新的对话），剩余预算留给结构化历史
	recentBudget := int(math.Round(float64(maxChars) * 0.65))
	recentPart := utils.ClipTail(recentTranscript, recentBudget)

	// 结构化历史预算 = 总预算 - 近期转录长度 - 分隔符长度（取 max 0 防止负数）
	structuredBudget := max(0, maxChars-utils.Len(recentPart)-2)
	structuredPart := utils.ClipHead(structuredHistory, structuredBudget)

	// 合并结构化文本与近期转录（非空项以 "\n\n" 分隔）
	return utils.JoinNonBlank("\n\n", structuredPart, recentPart)
}

// appendSection 追加章节
func (m *MemoryLoadStage) appendSection(sb *strings.Builder, title, content string) {
	if strutil.IsBlank(content) {
		return
	}
	if sb.Len() > 0 {
		sb.WriteString("\n")
	}
	sb.WriteString("【")
	sb.WriteString(title)
	sb.WriteString("】\n")
	sb.WriteString(strutil.Trim(content))
	sb.WriteString("\n")
}

// appendBulletSection 追加带项目符号的章节
func (m *MemoryLoadStage) appendBulletSection(sb *strings.Builder, title string, values []string) {
	if len(values) == 0 {
		return
	}
	if sb.Len() > 0 {
		sb.WriteString("\n")
	}
	sb.WriteString("【")
	sb.WriteString(title)
	sb.WriteString("】\n")

	stream.FromSlice(values).
		Map(func(item string) string { return strutil.Trim(item) }).
		Filter(func(item string) bool { return item != "" }).
		Limit(5).
		ForEach(func(item string) {
			sb.WriteString("- ")
			sb.WriteString(item)
			sb.WriteString("\n")
		})
}
