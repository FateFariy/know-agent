package conversation

import (
	"context"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/chat/adapter"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/aggregate"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/enum"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
	"github.com/swiftbit/know-agent/internal/svc"
)

// MemoryLoadStage 装载会话记忆并组装初始执行计划，必须在查询改写（QueryRewriteStage）之前执行，
// 使后续改写、语义缓存、检索路由等阶段都能复用历史上下文与检索计划。
type MemoryLoadStage struct {
	memoryManager           SessionMemoryManager
	repo                    adapter.ChatRepository
	planningHistoryMaxChars int
}

func NewMemoryLoadStage(svcCtx *svc.ServiceContext, memoryManager SessionMemoryManager, repo adapter.ChatRepository) *MemoryLoadStage {
	return &MemoryLoadStage{
		memoryManager:           memoryManager,
		repo:                    repo,
		planningHistoryMaxChars: svcCtx.Config.Chat.Rag.PlanningHistoryMaxChars,
	}
}

// Name 阶段名称
func (s *MemoryLoadStage) Name() string { return enum.ConversationTraceStageMemory.Name }

// Order 阶段顺序：排在查询改写之前
func (s *MemoryLoadStage) Order() int { return enum.ConversationTraceStageMemory.Order }

// Execute 装载会话记忆并组装执行计划
func (s *MemoryLoadStage) Execute(ctx context.Context, convCtx *Context) error {
	ctx = vo.OnStart(ctx, enum.ConversationTraceStageMemory, &vo.StageInput{SummaryText: "正在装载会话记忆与最近窗口。"})

	history, err := s.load(ctx, convCtx)
	if err != nil {
		ctx = vo.OnError(ctx, "会话记忆装载失败。", err)
		return err
	}
	ctx = vo.OnEnd(ctx, &vo.StageOutput{SummaryText: "会话记忆装载完成。", Snapshot: history.BuildSnapshot()})
	return nil
}

// load 装载会话记忆（长期摘要、近期转录、压缩信息），并组装初始执行计划
func (s *MemoryLoadStage) load(ctx context.Context, convCtx *Context) (*aggregate.Conversation, error) {
	memoryContext, err := s.memoryManager.LoadMemoryContext(ctx, convCtx.ConversationId)
	if err != nil {
		return nil, err
	}

	// 聚合长期记忆，供 Agent 做规划引用
	historyPlanningContext := vo.NewHistoryPlanningContext(memoryContext.Summary)
	// 以可读文本形式描述规划历史
	historySummary := historyPlanningContext.BuildPlanningText(memoryContext.RecentTranscript, s.planningHistoryMaxChars)
	// 当前问题 + 近期对话转录，用于后续改写与检索
	recentQuestions := utils.Trim(memoryContext.RecentQuestionTranscript)
	questionHistoryContext := vo.NewQuestionHistoryContext(recentQuestions, s.planningHistoryMaxChars)

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
		CurrentDateText:             convCtx.CurrentDateText,
	}
	convCtx.SetExecutePlan(execPlan)

	// 检索计划前置构建：在 agent 启动前按对话范围组装基础计划，
	// 路由/检索等工具后续在计划基础上回填补充（导航决策、子问题、topK 等）
	execPlan.RetrievalPlan = buildRetrievalPlan(convCtx, execPlan)
	debugTrace := vo.NewChatDebugTrace(convCtx.ExecutionPlan.Load())
	convCtx.DebugTrace.Store(debugTrace)

	return memoryContext, nil
}

// buildRetrievalPlan 前置构建检索执行计划（agent 启动前完成）
func buildRetrievalPlan(convCtx *Context, execPlan *vo.ConversationExecutionPlan) *vo.RetrievalPlan {
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
