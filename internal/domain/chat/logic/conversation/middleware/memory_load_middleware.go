package middleware

import (
	"context"
	"strings"

	"github.com/swiftbit/know-agent/common/logx"
	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/chat/adapter"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/conversation"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/aggregate"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/enum"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
	"github.com/swiftbit/know-agent/internal/svc"
)

const (
	maxEvidenceAnchorSnippetChars = 300 // 证据锚点内容片段最大字符数
	maxRecentExchanges            = 3   // 追问承接最多回溯的对话轮次
)

// MemoryLoadMiddleware 装载会话记忆并组装初始执行计划，同时把记忆摘要与时效/实时检索规则判断、上一轮证据锚点（追问承接）作为Agent指令注入
type MemoryLoadMiddleware struct {
	BaseAgentMiddleware
	memoryManager           conversation.SessionMemoryManager
	repo                    adapter.ChatRepository // 用于加载最近证据锚点
	planningHistoryMaxChars int                    // 规划历史最大字符数
}

// NewMemoryLoadMiddleware 创建记忆装载中间件
func NewMemoryLoadMiddleware(svcCtx *svc.ServiceContext, memoryManager conversation.SessionMemoryManager, repo adapter.ChatRepository) *MemoryLoadMiddleware {
	return &MemoryLoadMiddleware{
		memoryManager:           memoryManager,
		repo:                    repo,
		planningHistoryMaxChars: svcCtx.Config.Chat.Rag.PlanningHistoryMaxChars,
	}
}

// Name 中间件名称
func (m *MemoryLoadMiddleware) Name() string { return "memory-load" }

// BeforeAgent 装载记忆 → 组装执行计划 → 追加指令提示
func (m *MemoryLoadMiddleware) BeforeAgent(ctx context.Context, convCtx *conversation.Context, input *BeforeAgentInput) (*BeforeAgentOutput, error) {
	if convCtx == nil {
		return &BeforeAgentOutput{Instruction: input.Instruction}, nil
	}

	ctx = vo.OnStart(ctx, enum.ConversationTraceStageMemory, &vo.StageInput{SummaryText: "正在装载会话记忆与最近窗口。"})

	history, err := m.load(ctx, convCtx)
	if err != nil {
		ctx = vo.OnError(ctx, "会话记忆装载失败。", err)
		return &BeforeAgentOutput{Instruction: input.Instruction}, err
	}
	ctx = vo.OnEnd(ctx, &vo.StageOutput{SummaryText: "会话记忆装载完成。", Snapshot: history.BuildSnapshot()})

	instruction := input.Instruction
	if hints := m.buildInstructionHints(convCtx); len(hints) > 0 {
		if instruction != "" && !strings.HasSuffix(instruction, "\n") {
			instruction += "\n"
		}
		instruction += strings.Join(hints, "\n")
	}
	return &BeforeAgentOutput{Instruction: instruction}, nil
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
	// 追问承接：装配上一轮证据锚点
	anchors := m.loadFollowUpAnchors(ctx, convCtx)

	// 组装初始执行计划
	question := utils.CompactWhitespace(convCtx.Question)
	execPlan := &vo.ConversationExecutionPlan{
		ChatMode:                    convCtx.ChatMode,
		OriginalQuestion:            question,
		RewriteQuestion:             question,
		RewriteSubQuestions:         []string{question},
		HistorySummary:              historySummary,
		LongTermSummary:             memoryContext.LongTermSummary,
		HistoryPlanningContext:      historyPlanningContext,
		RecentHistoryTranscript:     memoryContext.RecentTranscript,
		RecentQuestionTranscript:    memoryContext.RecentQuestionTranscript,
		QuestionHistoryContext:      questionHistoryContext,
		NavigationDecision:          &vo.DocumentNavigationDecision{NavigationAction: enum.DocumentNavigationActionFreshTopic},
		HistoryCompressionApplied:   memoryContext.CompressionApplied,
		HistoryCoveredExchangeId:    memoryContext.CoveredExchangeId,
		HistoryCoveredExchangeCount: memoryContext.CoveredExchangeCount,
		HistoryCompressionCount:     memoryContext.CompressionCount,
		RecentEvidenceAnchors:       anchors,
		CurrentDateText:             convCtx.CurrentDateText,
	}
	convCtx.SetExecutePlan(execPlan)

	// 检索计划前置构建：在 agent 启动前按对话范围组装基础计划，
	// 路由/检索等工具后续在计划基础上回填补充（导航决策、子问题、topK 等）
	execPlan.RetrievalPlan = buildRetrievalPlan(convCtx, execPlan)

	return memoryContext, nil
}

// loadFollowUpAnchors 加载并过滤上一轮证据锚点（失败仅告警，不阻断 agent）
func (m *MemoryLoadMiddleware) loadFollowUpAnchors(ctx context.Context, convCtx *conversation.Context) vo.EvidenceAnchors {
	anchors, err := m.loadRecentEvidenceAnchors(ctx, convCtx.ConversationId)
	if err != nil {
		logx.Warnf("加载最近证据锚点失败, conversationId=%s, err=%v", convCtx.ConversationId, err)
		return nil
	}
	docIds := []int64{convCtx.SelectedDocumentId}
	if convCtx.ChatMode == enum.ChatQueryModeAutoDocument {
		docIds = convCtx.KnowledgeBaseSelectionSnapshot.SelectedDocumentIds()
	}
	return filterEvidenceAnchorsByDocument(anchors, docIds...)
}

// loadRecentEvidenceAnchors 从最近几轮已完成回答中加载证据锚点（追问承接信息源）
func (m *MemoryLoadMiddleware) loadRecentEvidenceAnchors(ctx context.Context, conversationId string) (vo.EvidenceAnchors, error) {
	if conversationId == "" {
		return nil, nil
	}
	exchanges, err := m.repo.ListRecentExchanges(ctx, conversationId, maxRecentExchanges)
	if err != nil || len(exchanges) == 0 {
		return nil, err
	}

	var anchors vo.EvidenceAnchors
	for _, exchange := range exchanges {
		if exchange == nil || !exchange.IsCompleted() || len(exchange.References) == 0 {
			continue
		}
		for _, ref := range exchange.References {
			anchor := ref.ToEvidenceAnchor(maxEvidenceAnchorSnippetChars)
			if anchor == nil {
				continue
			}
			anchors = append(anchors, anchor)
			if len(anchors) >= maxRecentExchanges {
				return anchors, nil
			}
		}
	}
	return anchors, nil
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

	analyzer := NewQueryAnalyzer(question)
	requiresCurrentDateAnchoring := analyzer.RequiresCurrentDateAnchoring()
	requiresRealTimeSearch := analyzer.RequiresRealTimeSearch()
	execPlan.RequiresRealTimeSearch = requiresRealTimeSearch
	execPlan.RequiresCurrentDateAnchoring = requiresCurrentDateAnchoring

	var hints []string
	addHint := func(condition bool, content string) {
		if condition {
			hints = append(hints, content)
		}
	}
	addHint(utils.IsNotBlank(convCtx.CurrentDateText), "当前日期："+convCtx.CurrentDateText+"。")
	addHint(requiresCurrentDateAnchoring, "当前问题包含相对时间或强时效表达（如“今天、明天、现在、最新、本周、本月”等），涉及日期必须以此为准，不要用检索结果中的旧日期替代今天。")
	addHint(requiresRealTimeSearch, "当前问题需要核实最新信息，检索证据不足或日期滞后时请明确说明不确定性，不要编造。")
	addHint(utils.IsNotBlank(execPlan.HistorySummary), "相关会话背景：\n"+execPlan.HistorySummary)
	addHint(len(execPlan.RecentEvidenceAnchors) > 0, execPlan.RecentEvidenceAnchors.RenderFollowUpHint())

	return hints
}

// filterEvidenceAnchorsByDocument 保留属于指定文档集合内的证据锚点；文档集合为空时原样返回。
func filterEvidenceAnchorsByDocument(anchors vo.EvidenceAnchors, allDocumentIds ...int64) vo.EvidenceAnchors {
	if len(anchors) == 0 || len(allDocumentIds) == 0 {
		return anchors
	}
	return utils.Filter(anchors, func(anchor *vo.EvidenceAnchor) bool {
		return anchor != nil && utils.ContainsAny(allDocumentIds, anchor.DocumentId)
	})
}

// buildRetrievalPlan 前置构建检索执行计划（agent 启动前完成）
func buildRetrievalPlan(convCtx *conversation.Context, execPlan *vo.ConversationExecutionPlan) *vo.RetrievalPlan {
	snapshot := convCtx.KnowledgeBaseSelectionSnapshot
	runtime := snapshot.RagRuntimeOptions
	if runtime == nil {
		runtime = vo.NewDefaultRagRuntimeOptions()
	}

	hybrid := runtime.Hybrid
	if hybrid == nil {
		hybrid = vo.NewDefaultHybridOptions()
	}

	channels := newChannelPlans(runtime, hybrid)
	chatMode := execPlan.ChatMode
	documentScope := snapshot.SelectedDocumentIds()
	taskScope := snapshot.SelectedTaskIds()
	if chatMode == enum.ChatQueryModeDocument {
		documentScope = []int64{convCtx.SelectedDocumentId}
		taskScope = []int64{convCtx.SelectedTaskId}
	}
	return &vo.RetrievalPlan{
		ChatMode:           enum.ChatQueryModeName(chatMode),
		PrimaryIntent:      enum.RetrievalIntentGeneral,
		ScopeMode:          snapshot.SelectionModeName(),
		KnowledgeBaseIds:   utils.Copy(snapshot.SelectedKnowledgeBaseIds),
		DocumentScope:      utils.Copy(documentScope),
		TaskScope:          utils.Copy(taskScope),
		MetadataFilters:    vo.NewMetadataFilters(execPlan.RewriteQuestion),
		Channels:           channels,
		NavigationAction:   execPlan.NavigationDecision.NavigationActionText(),
		RankFeatures:       vo.BuildRankFeatures(hybrid),
		CandidateTopK:      runtime.CandidateTopK,
		RerankTopK:         runtime.RerankCandidateTopK,
		RerankEnabled:      runtime.RerankEnabled,
		FinalTopK:          runtime.FinalTopK,
		SubQuestionTimeout: runtime.SubQuestionTimeout,
	}
}

// newChannelPlans 构建检索通道计划列表
func newChannelPlans(runtime *vo.RagRuntimeOptions, hybrid *vo.HybridOptions) []*vo.RetrievalChannelPlan {
	return []*vo.RetrievalChannelPlan{
		vo.NewVectorChannelPlan(true, runtime.VectorTopK, runtime.ChannelTimeout, hybrid.VectorWeight, runtime.MinVectorSimilarity),
		vo.NewKeywordChannelPlan(runtime.KeywordChannelEnabled, runtime.KeywordTopK, runtime.ChannelTimeout, hybrid.KeywordWeight, runtime.KeywordRelativeScoreFloor),
		vo.NewTableChannelPlan(runtime.TableChannelEnabled, runtime.CandidateTopK, runtime.ChannelTimeout, hybrid.TableWeight),
		vo.NewGraphRAGChannelPlan(runtime.GraphRagChannelEnabled, runtime.GraphRagTopK, runtime.ChannelTimeout, hybrid.GraphRagWeight),
		vo.NewRaptorChannelPlan(runtime.RaptorChannelEnabled, runtime.RaptorTopK, runtime.ChannelTimeout, hybrid.RaptorWeight),
	}
}
