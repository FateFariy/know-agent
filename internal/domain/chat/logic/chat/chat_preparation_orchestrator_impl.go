package chat

import (
	"context"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/duke-git/lancet/v2/stream"
	"github.com/duke-git/lancet/v2/strutil"
	"github.com/zeromicro/go-zero/core/logx"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/chat/adapter"
	logic2 "github.com/swiftbit/know-agent/internal/domain/chat/logic"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
	"github.com/swiftbit/know-agent/internal/domain/chat/support"
	"github.com/swiftbit/know-agent/internal/domain/knowledge/logic"
	klvo "github.com/swiftbit/know-agent/internal/domain/rag/model/vo"
	"github.com/swiftbit/know-agent/internal/svc"
)

var (
	capabilityHints = []string{"你都能干什么", "你能做什么", "你可以做什么", "你会什么", "你是谁", "怎么用你", "你能帮我什么"}
	openChatHints   = []string{"天气", "温度", "下雨", "新闻", "股价", "汇率", "热搜", "今天", "明天", "最新", "现在"}
	chitchatHints   = []string{"你好", "您好", "hello", "hi", "谢谢", "感谢", "再见", "拜拜"}
)

// PreparationOrchestratorImpl 聊天准备编排器实现
type PreparationOrchestratorImpl struct {
	repo                    adapter.ChatRepository
	memoryLogic             logic2.SessionMemoryLogic
	rewriteLogic            logic2.QueryRewriteLogic
	documentQuestionRouter  logic2.DocumentQuestionRouter
	knowledgeRouteService   logic2.KnowledgeRouteService
	documentKnowledgeLogic  logic.DocumentKnowledgeLogic
	ragEnabled              bool
	planningHistoryMaxChars int // 规划历史最大字符数
	questionHistoryMaxChars int // 问题历史最大字符数
	noEvidenceReply         string
}

// NewChatPreparationOrchestrator 创建聊天准备编排器实例
func NewChatPreparationOrchestrator(svcCtx *svc.ServiceContext,
	repo adapter.ChatRepository,
	memoryLogic logic2.SessionMemoryLogic,
	rewriteLogic logic2.QueryRewriteLogic,
	documentQuestionRouter logic2.DocumentQuestionRouter,
	knowledgeRouteService logic2.KnowledgeRouteService,
	documentKnowledgeLogic logic.DocumentKnowledgeLogic,
) *PreparationOrchestratorImpl {
	return &PreparationOrchestratorImpl{
		repo:                    repo,
		memoryLogic:             memoryLogic,
		rewriteLogic:            rewriteLogic,
		documentQuestionRouter:  documentQuestionRouter,
		knowledgeRouteService:   knowledgeRouteService,
		documentKnowledgeLogic:  documentKnowledgeLogic,
		ragEnabled:              svcCtx.Config.Chat.Rag.Enabled,
		planningHistoryMaxChars: svcCtx.Config.Chat.Rag.PlanningHistoryMaxChars,
		questionHistoryMaxChars: max(1, svcCtx.Config.Chat.Rag.QuestionHistoryMaxChars),
		noEvidenceReply:         utils.BlankToDefault(svcCtx.Config.Chat.Rag.NoEvidenceReply, "当前没有从已接入文档中检索到足够证据，暂时不能给出可靠结论。"),
	}
}

// Prepare 准备对话执行计划
// 步骤：公共准备 → 按 chatMode 分发到各自独立的子方法 → 返回执行计划。
func (o *PreparationOrchestratorImpl) Prepare(ctx context.Context, convCtx *vo.ConversationContext) (*vo.ConversationExecutionPlan, error) {
	execPlan, err := o.prepareCommon(ctx, convCtx)
	if err != nil {
		return nil, err
	}

	switch convCtx.ChatMode {
	case vo.ChatQueryModeOpenChat:
		return o.prepareOpenChat(ctx, execPlan)
	case vo.ChatQueryModeDocument:
		return o.prepareDocumentMode(ctx, execPlan)
	case vo.ChatQueryModeAutoDocument:
		return o.prepareAutoDocumentMode(ctx, execPlan)
	default:
		return nil, fmt.Errorf("不支持的聊天模式: %s", convCtx.ChatMode.Name())
	}
}

// prepareCommon 执行所有模式共享的准备步骤：记忆装载、历史规划上下文、时间信号、问题改写。
func (o *PreparationOrchestratorImpl) prepareCommon(ctx context.Context, convCtx *vo.ConversationContext) (*vo.ConversationExecutionPlan, error) {
	question := strutil.Trim(convCtx.Question)

	// 装载会话记忆
	memoryContext, err := o.summarizeHistory(ctx, convCtx)
	if err != nil {
		return nil, err
	}

	// 构建历史规划上下文与问题历史上下文
	historyPlanningContext := vo.NewHistoryPlanningContext(memoryContext.Summary)
	historySummary := o.buildPlanningHistory(memoryContext, historyPlanningContext)
	questionHistoryContext := vo.NewQuestionHistoryContext(question, strutil.Trim(memoryContext.RecentTranscript), o.questionHistoryMaxChars)

	// 判断时间敏感与实时搜索需求
	requiresCurrentDateAnchoring := support.RequiresCurrentDateAnchoring(question)
	requiresRealTimeSearch := support.RequiresRealTimeSearch(question)

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
		CurrentDate:                  time.Now(),
		CurrentDateText:              time.Now().Format(time.DateTime),
		RequiresRealTimeSearch:       requiresRealTimeSearch,
		RequiresCurrentDateAnchoring: requiresCurrentDateAnchoring,
		NoEvidenceReply:              o.noEvidenceReply,
	}

	// 非 OpenChat 模式需要问题改写产物
	if convCtx.ChatMode != vo.ChatQueryModeOpenChat {
		if !o.ragEnabled {
			return nil, fmt.Errorf("当前文档问答模式未启用，请先开启聊天侧 RAG 编排")
		}
		rewriteResult, err := o.questionRewrite(ctx, convCtx, historySummary)
		if err != nil {
			return nil, err
		}
		execPlan.RewriteResult = rewriteResult
		execPlan.RewriteQuestion = utils.BlankToDefault(rewriteResult.RewrittenQuestion, strutil.Trim(convCtx.Question))
		if len(rewriteResult.SubQuestions) > 0 {
			execPlan.rewriteSubQuestions = rewriteResult.SubQuestions
		} else {
			execPlan.rewriteSubQuestions = []string{prepCtx.rewriteQuestion}
		}
	}
	return execPlan, nil
}

// prepareOpenChat 开放式聊天：直接返回 ReactAgent 模式执行计划
func (o *PreparationOrchestratorImpl) prepareOpenChat(ctx context.Context, convCtx *vo.ConversationContext, execPlan *vo.ConversationExecutionPlan) error {
	execPlan.Mode = vo.ExecutionModeReactAgent

	tracer := convCtx.Tracer
	if tracer != nil {
		// todo 未持久化trace
		stage := tracer.StartStage(vo.ConversationTraceStageRewrite, vo.ExecutionModeReactAgent.String(), "路由到开放式 Agent。", nil)
		tracer.CompleteStage(stage, "已判定走开放式 Agent 路径。", map[string]any{
			"chatMode":                     convCtx.ChatMode.String(),
			"executionMode":                vo.ExecutionModeReactAgent.String(),
			"requiresRealTimeSearch":       execPlan.RequiresRealTimeSearch,
			"requiresCurrentDateAnchoring": execPlan.RequiresCurrentDateAnchoring,
		})
	}
	return nil
}

// prepareDocumentMode 指定文档问答：记录影子路由 → 文档内导航 → 生成执行计划。
func (o *PreparationOrchestratorImpl) prepareDocumentMode(ctx context.Context, execPlan *vo.ConversationExecutionPlan) error {
	if execPlan.selectedDocumentId == 0 || execPlan.selectedTaskId == 0 {
		return nil, fmt.Errorf("当前文档问答模式缺少有效的文档范围")
	}

	// 记录影子路由（不影响业务结果，失败仅告警）
	if err := o.knowledgeRouteService.RecordShadowRoute(ctx, execPlan.convCtx.ConversationId,
		strconv.FormatInt(execPlan.convCtx.ExchangeId, 10), execPlan.selectedDocumentId, execPlan.question, execPlan.rewriteQuestion); err != nil {
		Warnf("记录影子路由失败: %v", err)
	}

	// 初始化文档范围（使用 *int64 形式的路由文档ID，便于下游传递给 documentQuestionRouter）
	routedDocIdPtr := o.int64Ptr(prepCtx.selectedDocumentId)
	routedTaskIdPtr := o.int64Ptr(prepCtx.selectedTaskId)
	routedDocumentIds := []int64{prepCtx.selectedDocumentId}
	routedTaskIds := []int64{prepCtx.selectedTaskId}

	// 文档内导航
	navigationDecision := o.runDocumentNavigation(ctx, prepCtx, routedDocIdPtr)

	// 确定执行模式与检索问题
	executionMode := vo.ExecutionModeRetrieval
	if navigationDecision != nil && navigationDecision.ExecutionMode != 0 {
		executionMode = navigationDecision.ExecutionMode
	}
	retrievalQuestion, retrievalSubQuestions := o.resolveRetrievalQuestions(navigationDecision, prepCtx.rewriteQuestion, prepCtx.rewriteSubQuestions)

	// 构建最终执行计划
	plan := o.buildBasePlan(prepCtx)
	plan.Mode = executionMode
	plan.NavigationDecision = navigationDecision
	plan.RewriteQuestion = prepCtx.rewriteQuestion
	plan.RewriteSubQuestions = prepCtx.rewriteSubQuestions
	plan.RetrievalQuestion = retrievalQuestion
	plan.RetrievalSubQuestions = retrievalSubQuestions
	plan.SelectedDocumentId = prepCtx.selectedDocumentId
	plan.SelectedDocumentName = prepCtx.selectedDocumentName
	plan.SelectedTaskId = prepCtx.selectedTaskId
	plan.RetrievalDocumentIds = routedDocumentIds
	plan.RetrievalTaskIds = routedTaskIds
	plan.NoEvidenceReply = o.buildDocumentModeNoEvidenceReply(prepCtx.question, prepCtx.requiresRealTimeSearch)

	logx.Infof("聊天编排完成: conversationId=%s, chatMode=%s, originalQuestion='%s', rewriteQuestion='%s', retrievalQuestion='%s', executionMode=%s",
		prepCtx.convCtx.ConversationId, prepCtx.chatMode.String(), strutil.Trim(prepCtx.question),
		prepCtx.rewriteQuestion, retrievalQuestion, executionMode.String())

	return plan, nil
}

// prepareAutoDocumentMode 自动文档问答：执行知识路由 → 必要时澄清 → 确定主文档 → 导航 → 生成计划。
func (o *PreparationOrchestratorImpl) prepareAutoDocumentMode(ctx context.Context, prepCtx *preparationContext) (*vo.ConversationExecutionPlan, error) {
	tracer := prepCtx.convCtx.Tracer
	exchangeIdText := strconv.FormatInt(prepCtx.convCtx.ExchangeId, 10)

	// 执行知识路由
	routeDecision, err := o.knowledgeRouteService.Route(ctx, prepCtx.question, prepCtx.rewriteQuestion)
	if err != nil {
		Warnf("知识路由失败: %v", err)
		routeDecision = nil
	}
	if recordErr := o.knowledgeRouteService.RecordAutoRoute(ctx, prepCtx.convCtx.ConversationId,
		exchangeIdText, prepCtx.question, prepCtx.rewriteQuestion, routeDecision); recordErr != nil {
		Warnf("记录自动路由失败: %v", recordErr)
	}

	// 选择候选文档（含低置信度时的 fallback）
	candidateDocuments := o.selectAutoCandidates(ctx, routeDecision, prepCtx.question, prepCtx.rewriteQuestion)

	// 若需要澄清，直接返回澄清模式执行计划
	if o.shouldAskClarification(routeDecision, candidateDocuments) {
		return o.buildClarificationPlan(prepCtx, routeDecision, candidateDocuments), nil
	}

	// 路由跟踪
	if tracer != nil && routeDecision != nil {
		confidentTop := routeDecision.Confidence >= 0.55
		tracer.CompleteStage(
			tracer.StartStage(vo.StageCodeRoute, "AUTO_DOCUMENT", "正在生成知识范围候选。", nil),
			"知识范围路由完成。",
			map[string]interface{}{
				"confidence":             routeDecision.Confidence,
				"routeStatus":            routeDecision.RouteStatus,
				"candidateDocumentCount": len(candidateDocuments),
				"confidentTopDocument":   confidentTop,
			},
		)
	}

	// 选择最高置信度的文档作为主文档；否则回退到“不指定主文档，使用多文档范围检索”
	var routedDocIdPtr *int64
	var routedTaskIdPtr *int64
	routedDocumentName := ""
	if routeDecision != nil && routeDecision.Confidence >= 0.55 && len(candidateDocuments) > 0 {
		topDocument := candidateDocuments[0]
		if topDocument.DocumentId != "" && topDocument.LastIndexTaskId != "" {
			if docId, parseErr := strconv.ParseInt(topDocument.DocumentId, 10, 64); parseErr == nil {
				routedDocIdPtr = &docId
			}
			if taskId, parseErr := strconv.ParseInt(topDocument.LastIndexTaskId, 10, 64); parseErr == nil {
				routedTaskIdPtr = &taskId
			}
			routedDocumentName = topDocument.DocumentName
		}
	}

	routedDocumentIds := o.extractDocumentIds(candidateDocuments)
	routedTaskIds := o.extractTaskIds(candidateDocuments)

	// 文档内导航
	navigationDecision := o.runDocumentNavigation(ctx, prepCtx, routedDocIdPtr)

	// 确定执行模式与检索问题
	executionMode := vo.ExecutionModeRetrieval
	if navigationDecision != nil && navigationDecision.ExecutionMode != 0 {
		executionMode = navigationDecision.ExecutionMode
	}
	retrievalQuestion, retrievalSubQuestions := o.resolveRetrievalQuestions(navigationDecision, prepCtx.rewriteQuestion, prepCtx.rewriteSubQuestions)

	// 构建最终执行计划
	plan := o.buildBasePlan(prepCtx)
	plan.Mode = executionMode
	plan.NavigationDecision = navigationDecision
	plan.RewriteQuestion = prepCtx.rewriteQuestion
	plan.RewriteSubQuestions = prepCtx.rewriteSubQuestions
	plan.RetrievalQuestion = retrievalQuestion
	plan.RetrievalSubQuestions = retrievalSubQuestions
	if routedDocIdPtr != nil {
		plan.SelectedDocumentId = *routedDocIdPtr
	}
	plan.SelectedDocumentName = routedDocumentName
	if routedTaskIdPtr != nil {
		plan.SelectedTaskId = *routedTaskIdPtr
	}
	plan.RetrievalDocumentIds = routedDocumentIds
	plan.RetrievalTaskIds = routedTaskIds
	plan.NoEvidenceReply = o.buildDocumentModeNoEvidenceReply(prepCtx.question, prepCtx.requiresRealTimeSearch)

	logx.Infof("聊天编排完成: conversationId=%s, chatMode=%s, originalQuestion='%s', rewriteQuestion='%s', retrievalQuestion='%s', executionMode=%s",
		prepCtx.convCtx.ConversationId, prepCtx.chatMode.String(), strutil.Trim(prepCtx.question),
		prepCtx.rewriteQuestion, retrievalQuestion, executionMode.String())
	return plan, nil
}

// buildClarificationPlan 构建澄清模式的执行计划（仅在自动文档模式下返回）。
func (o *PreparationOrchestratorImpl) buildClarificationPlan(prepCtx *preparationContext, routeDecision *vo.KnowledgeRouteDecision, candidateDocuments []*vo.DocumentRouteCandidate) *vo.ConversationExecutionPlan {
	plan := o.buildBasePlan(prepCtx)
	plan.Mode = vo.ExecutionModeClarification
	plan.RewriteQuestion = prepCtx.rewriteQuestion
	plan.RewriteSubQuestions = prepCtx.rewriteSubQuestions
	plan.RetrievalQuestion = prepCtx.rewriteQuestion
	plan.RetrievalSubQuestions = prepCtx.rewriteSubQuestions
	plan.RetrievalDocumentIds = o.extractDocumentIds(candidateDocuments)
	plan.RetrievalTaskIds = o.extractTaskIds(candidateDocuments)
	plan.ClarificationReply = o.buildClarificationReply(prepCtx.question, routeDecision, candidateDocuments)
	plan.ClarificationOptions = o.buildClarificationOptions(candidateDocuments)
	plan.ClarificationReason = o.buildClarificationReason(routeDecision, candidateDocuments)
	return plan
}

// runDocumentNavigation 统一执行“文档内导航路由”，并负责 tracer 的成功 / 失败记录。
// 使用 recover 风格 try 原语义不变。
func (o *PreparationOrchestratorImpl) runDocumentNavigation(ctx context.Context, prepCtx *preparationContext, routedDocumentId *int64) *vo.DocumentNavigationDecision {
	tracer := prepCtx.convCtx.Tracer
	var stage *vo.StageHandle
	if tracer != nil {
		stage = tracer.StartStage(vo.StageCodeRoute, vo.ExecutionModeRetrieval.String(), "正在判定图查询还是混合检索。", nil)
	}
	var navigationDecision *vo.DocumentNavigationDecision
	try(func() {
		var routeErr error
		navigationDecision, routeErr = o.documentQuestionRouter.Route(ctx, routedDocumentId, prepCtx.question, prepCtx.rewriteResult)
		if routeErr != nil {
			panic(routeErr)
		}
		if tracer != nil && stage != nil && navigationDecision != nil {
			tracer.CompleteStage(stage, "执行路由完成。", map[string]interface{}{
				"executionMode":     navigationDecision.ExecutionMode.String(),
				"targetSectionHint": strutil.Trim(navigationDecision.StructureAnchor.AnchorName),
				"navigationSummary": strutil.Trim(navigationDecision.SummaryText),
			})
		}
	}, func(err error) {
		if tracer != nil && stage != nil {
			tracer.FailStage(stage, "执行路由失败。", err, nil)
		}
	})
	return navigationDecision
}

// resolveRetrievalQuestions 根据导航结果确定最终的检索问题与子问题列表；导航为空时回退到 rewrite 结果。
func (o *PreparationOrchestratorImpl) resolveRetrievalQuestions(navigationDecision *vo.DocumentNavigationDecision, rewriteQuestion string, rewriteSubQuestions []string) (string, []string) {
	retrievalQuestion := rewriteQuestion
	retrievalSubQuestions := rewriteSubQuestions
	if navigationDecision != nil && navigationDecision.RetrievalPlan != nil {
		retrievalQuestion = utils.BlankToDefault(navigationDecision.RetrievalPlan.RetrievalQuestion, rewriteQuestion)
		if len(navigationDecision.RetrievalPlan.SubQuestions) > 0 {
			retrievalSubQuestions = navigationDecision.RetrievalPlan.SubQuestions
		}
	}
	return retrievalQuestion, retrievalSubQuestions
}

// summarizeHistory 构建会话记忆
func (o *PreparationOrchestratorImpl) summarizeHistory(ctx context.Context, convCtx *vo.ConversationContext) (*vo.MemoryContext, error) {
	tracer := convCtx.Tracer
	memoryStage := tracer.StartStage(vo.ConversationTraceStageMemory, convCtx.ChatMode.Name(), "正在装载会话记忆与最近窗口。", nil)
	if err := o.repo.InsertStage(ctx, tracer.ConvChatExchangeTraceStage()); err != nil {
		return nil, err
	}

	memoryContext, err := o.memoryLogic.LoadMemoryContext(ctx, convCtx.ConversationId, tracer)
	if err != nil {
		tracer.FailStage(memoryStage, "会话记忆装载失败。", err, nil)
		if err = o.repo.InsertStage(ctx, tracer.ConvChatExchangeTraceStage()); err != nil {
			return nil, err
		}
		return nil, err
	}
	tracer.CompleteStage(memoryStage, "会话记忆装载完成。", map[string]any{
		"compressionApplied":       memoryContext.CompressionApplied,
		"coveredExchangeId":        memoryContext.CoveredExchangeId,
		"coveredExchangeCount":     memoryContext.CoveredExchangeCount,
		"compressionCount":         memoryContext.CompressionCount,
		"longTermSummary":          strutil.Trim(memoryContext.LongTermSummary),
		"recentTranscript":         strutil.Trim(memoryContext.RecentTranscript),
		"RecentQuestionTranscript": strutil.Trim(memoryContext.RecentQuestionTranscript),
	})
	if err = o.repo.InsertStage(ctx, tracer.ConvChatExchangeTraceStage()); err != nil {
		return nil, err
	}
	return memoryContext, nil
}

func (o *PreparationOrchestratorImpl) questionRewrite(ctx context.Context, convCtx *vo.ConversationContext, historySummary string) (*vo.RagRewriteResult, error) {
	tracer := convCtx.Tracer
	rewriteStage := tracer.StartStage(vo.ConversationTraceStageRewrite, vo.ExecutionModeRetrieval.String(), "正在生成检索友好的问题表达。", o.buildRewriteStageSnapshot(convCtx.Question, historySummary, nil))
	rewriteResult, err := o.rewriteLogic.Rewrite(ctx, convCtx.Question, historySummary, tracer)
	if err != nil {
		tracer.FailStage(rewriteStage, "问题改写失败。", err, o.buildRewriteStageSnapshot(convCtx.Question, historySummary, nil))
	}
	tracer.CompleteStage(rewriteStage, "问题改写完成。", o.buildRewriteStageSnapshot(convCtx.Question, historySummary, rewriteResult))

	// 获取改写后的问题
	rewriteQuestion := utils.BlankToDefault(rewriteResult.RewrittenQuestion, strutil.Trim(convCtx.Question))
	rewriteSubQuestions := rewriteResult.SubQuestions
	if len(rewriteSubQuestions) == 0 {
		rewriteSubQuestions = []string{rewriteQuestion}
	}
	return rewriteResult, err
}

// buildRewriteStageSnapshot 构建改写阶段快照
func (o *PreparationOrchestratorImpl) buildRewriteStageSnapshot(question, historySummary string, rewriteResult *vo.RagRewriteResult) map[string]interface{} {
	snapshot := make(map[string]interface{})
	snapshot["originalQuestion"] = strutil.Trim(question)
	snapshot["historyContext"] = strutil.Trim(historySummary)

	if rewriteResult != nil {
		snapshot["rewriteQuestion"] = strutil.Trim(rewriteResult.RewrittenQuestion)
		snapshot["subQuestions"] = rewriteResult.SubQuestions
		snapshot["rawModelOutput"] = strutil.Trim(rewriteResult.RawModelOutput)
	} else {
		snapshot["rewriteQuestion"] = ""
		snapshot["subQuestions"] = []string{}
		snapshot["rawModelOutput"] = ""
	}

	return snapshot
}

// buildPlanningHistory 构建规划历史
func (o *PreparationOrchestratorImpl) buildPlanningHistory(memoryContext *vo.MemoryContext, historyPlanningContext *vo.HistoryPlanningContext) string {
	var sb strings.Builder
	o.appendSection(&sb, "会话目标", historyPlanningContext.ConversationGoal)
	o.appendBulletSection(&sb, "已确认事实", historyPlanningContext.StableFacts)
	o.appendBulletSection(&sb, "待跟进问题", historyPlanningContext.PendingQuestions)
	o.appendBulletSection(&sb, "检索提示", historyPlanningContext.RetrievalHints)
	structuredHistory := strutil.Trim(sb.String())
	recentTranscript := strutil.Trim(memoryContext.RecentTranscript)

	maxChars := o.planningHistoryMaxChars
	if recentTranscript == "" {
		return utils.ClipHead(structuredHistory, maxChars)
	}

	recentBudget := int(math.Round(float64(maxChars) * 0.65))
	recentPart := utils.ClipTail(recentTranscript, recentBudget)

	structuredBudget := max(0, maxChars-utf8.RuneCountInString(recentPart)-2)
	structuredPart := utils.ClipHead(structuredHistory, structuredBudget)

	return utils.JoinNonBlank(structuredPart, recentPart, "\n\n")
}

// appendSection 追加章节
func (o *PreparationOrchestratorImpl) appendSection(sb *strings.Builder, title, content string) {
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
func (o *PreparationOrchestratorImpl) appendBulletSection(sb *strings.Builder, title string, values []string) {
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

// selectAutoCandidates 选择自动候选文档
func (o *PreparationOrchestratorImpl) selectAutoCandidates(ctx context.Context, routeDecision *vo.KnowledgeRouteDecision, question, rewriteQuestion string) []*vo.DocumentRouteCandidate {
	if routeDecision == nil || len(routeDecision.Documents) == 0 {
		return o.fallbackDocuments(ctx, question, rewriteQuestion, 5)
	}

	candidateLimit := 5
	if routeDecision.Confidence >= 0.80 {
		candidateLimit = 3
	}

	var candidates []*vo.DocumentRouteCandidate
	for _, doc := range routeDecision.Documents {
		if doc.DocumentId != "" && doc.LastIndexTaskId != "" {
			candidates = append(candidates, doc)
			if len(candidates) >= candidateLimit {
				break
			}
		}
	}

	if len(candidates) == 0 {
		return o.fallbackDocuments(ctx, question, rewriteQuestion, candidateLimit)
	}

	if routeDecision.Confidence < 0.55 {
		return o.mergeCandidates(candidates, o.fallbackDocuments(ctx, question, rewriteQuestion, candidateLimit), candidateLimit)
	}

	return candidates
}

// fallbackDocuments 获取后备候选文档
func (o *PreparationOrchestratorImpl) fallbackDocuments(ctx context.Context, question, rewriteQuestion string, limit int) []*vo.DocumentRouteCandidate {
	descriptors, err := o.documentKnowledgeLogic.ListRetrievableDocuments(ctx)
	if err != nil {
		Warnf("获取可检索文档失败: %v", err)
		return []*vo.DocumentRouteCandidate{}
	}
	if len(descriptors) == 0 {
		return []*vo.DocumentRouteCandidate{}
	}

	queryTerms := o.extractFallbackTerms(question, rewriteQuestion)

	sort.Slice(descriptors, func(i, j int) bool {
		return o.fallbackDescriptorScore(descriptors[j], queryTerms) < o.fallbackDescriptorScore(descriptors[i], queryTerms)
	})

	result := make([]*vo.DocumentRouteCandidate, 0, limit)
	for i, desc := range descriptors {
		if i >= limit {
			break
		}
		score := o.fallbackDescriptorScore(desc, queryTerms)
		result = append(result, &vo.DocumentRouteCandidate{
			DocumentId:         strconv.FormatInt(desc.DocumentId, 10),
			DocumentName:       desc.DocumentName,
			LastIndexTaskId:    strconv.FormatInt(desc.LastIndexTaskId, 10),
			KnowledgeScopeCode: desc.KnowledgeScopeCode,
			KnowledgeScopeName: desc.KnowledgeScopeName,
			BusinessCategory:   desc.BusinessCategory,
			DocumentTags:       desc.DocumentTags,
			Score:              score,
			Reason:             "低置信度时基于文档元数据进行保守扩范围候选",
		})
	}

	return result
}

// mergeCandidates 合并候选文档
func (o *PreparationOrchestratorImpl) mergeCandidates(primary, secondary []*vo.DocumentRouteCandidate, limit int) []*vo.DocumentRouteCandidate {
	merged := make(map[string]*vo.DocumentRouteCandidate)
	for _, doc := range primary {
		merged[doc.DocumentId] = doc
	}
	for _, doc := range secondary {
		if _, exists := merged[doc.DocumentId]; !exists {
			merged[doc.DocumentId] = doc
		}
	}

	result := make([]*vo.DocumentRouteCandidate, 0, limit)
	for _, doc := range merged {
		if len(result) >= limit {
			break
		}
		result = append(result, doc)
	}
	return result
}

// shouldAskClarification 判断是否需要澄清
func (o *PreparationOrchestratorImpl) shouldAskClarification(routeDecision *vo.KnowledgeRouteDecision, candidateDocuments []*vo.DocumentRouteCandidate) bool {
	if len(candidateDocuments) == 0 {
		return true
	}
	if routeDecision == nil || len(routeDecision.Documents) == 0 {
		return true
	}
	if routeDecision.Confidence < 0.55 {
		return true
	}
	if len(candidateDocuments) < 2 {
		return false
	}

	topScore := candidateDocuments[0].Score
	secondScore := candidateDocuments[1].Score
	topScope := candidateDocuments[0].KnowledgeScopeCode
	secondScope := candidateDocuments[1].KnowledgeScopeCode

	return topScore-secondScore <= 3 && topScope != secondScope
}

// buildClarificationReply 构建澄清回复
func (o *PreparationOrchestratorImpl) buildClarificationReply(originalQuestion string, routeDecision *vo.KnowledgeRouteDecision, candidateDocuments []*vo.DocumentRouteCandidate) string {
	topCandidates := candidateDocuments
	if len(topCandidates) > 3 {
		topCandidates = topCandidates[:3]
	}

	if len(topCandidates) == 0 {
		return "当前我还不能稳定判断你想问哪份知识文档。请补充更具体的文档名、主题词，或者直接切换到“当前文档问答”后指定文档。"
	}

	var sb strings.Builder
	sb.WriteString("这个问题目前存在文档范围歧义，我先确认你想问哪一份：\n")

	for i, item := range topCandidates {
		sb.WriteString(strconv.Itoa(i + 1))
		sb.WriteString(". 《")
		name := item.DocumentName
		if name == "" {
			name = item.DocumentId
		}
		sb.WriteString(name)
		sb.WriteString("》")

		scope := item.KnowledgeScopeName
		if scope == "" {
			scope = item.KnowledgeScopeCode
		}
		if scope != "" {
			sb.WriteString("（")
			sb.WriteString(scope)
			sb.WriteString("）")
		}
		sb.WriteString("\n")
	}

	sb.WriteString("你可以直接回复文档名，或者改用“当前文档问答”模式明确指定文档。")
	return sb.String()
}

// buildClarificationOptions 构建澄清选项
func (o *PreparationOrchestratorImpl) buildClarificationOptions(candidateDocuments []*vo.DocumentRouteCandidate) []string {
	if len(candidateDocuments) == 0 {
		return []string{}
	}

	result := make([]string, 0, 3)
	for i, item := range candidateDocuments {
		if i >= 3 {
			break
		}
		name := item.DocumentName
		if name == "" {
			name = item.DocumentId
		}
		result = append(result, "我想问《"+name+"》")
	}
	return result
}

// buildClarificationReason 构建澄清原因
func (o *PreparationOrchestratorImpl) buildClarificationReason(routeDecision *vo.KnowledgeRouteDecision, candidateDocuments []*vo.DocumentRouteCandidate) string {
	if routeDecision == nil || len(routeDecision.Documents) == 0 {
		return "当前自动知识路由没有形成稳定候选，已改为先向用户确认文档范围。"
	}

	confidenceText := strconv.FormatFloat(routeDecision.Confidence, 'f', -1, 64)
	candidateCount := len(candidateDocuments)

	return "当前自动知识路由置信度为 " + confidenceText + "，候选文档数为 " + strconv.Itoa(candidateCount) + "，为避免误选文档，先返回澄清问题。"
}

// extractFallbackTerms 提取后备检索词
func (o *PreparationOrchestratorImpl) extractFallbackTerms(question, rewriteQuestion string) []string {
	terms := make(map[string]bool)
	routingText := strutil.Trim(question) + " " + strutil.Trim(rewriteQuestion)

	re := regexp.MustCompile(`[\s、，,；;：:（）()\-的和及与或]+`)
	segments := re.Split(routingText, -1)

	for _, segment := range segments {
		trimmed := strutil.Trim(segment)
		if len(trimmed) >= 2 {
			terms[trimmed] = true
			if len(trimmed) >= 4 {
				maxGram := len(trimmed)
				if maxGram > 6 {
					maxGram = 6
				}
				for gram := 2; gram <= maxGram; gram++ {
					for start := 0; start+gram <= len(trimmed); start++ {
						terms[trimmed[start:start+gram]] = true
					}
				}
			}
		}
	}

	result := make([]string, 0, len(terms))
	for term := range terms {
		result = append(result, term)
	}

	if len(result) > 40 {
		result = result[:40]
	}
	return result
}

// fallbackDescriptorScore 计算后备文档匹配分数
func (o *PreparationOrchestratorImpl) fallbackDescriptorScore(descriptor *klvo.KnowledgeDocument, queryTerms []string) float64 {
	content := strings.Join([]string{
		descriptor.DocumentName,
		descriptor.KnowledgeScopeCode,
		descriptor.KnowledgeScopeName,
		descriptor.BusinessCategory,
		descriptor.DocumentTags,
	}, " ")

	content = o.normalizeFallbackText(content)

	if len(queryTerms) == 0 || content == "" {
		return 0
	}

	var score float64
	sortedTerms := make([]string, 0, len(queryTerms))
	for _, term := range queryTerms {
		normalized := o.normalizeFallbackText(term)
		if normalized != "" {
			sortedTerms = append(sortedTerms, normalized)
		}
	}

	sort.Slice(sortedTerms, func(i, j int) bool {
		return len(sortedTerms[i]) > len(sortedTerms[j])
	})

	matched := make(map[string]bool)
	for _, term := range sortedTerms {
		if len(term) < 2 {
			continue
		}

		isCovered := false
		for existing := range matched {
			if strings.Contains(existing, term) {
				isCovered = true
				break
			}
		}
		if isCovered {
			continue
		}

		if strings.Contains(content, term) {
			matched[term] = true
			switch {
			case len(term) >= 8:
				score += 12
			case len(term) >= 5:
				score += 8
			case len(term) >= 3:
				score += 4
			default:
				score += 2
			}
		}
	}

	return score
}

// normalizeFallbackText 标准化后备文本
func (o *PreparationOrchestratorImpl) normalizeFallbackText(value string) string {
	if value == "" {
		return ""
	}
	re := regexp.MustCompile(`[\s>\` + "`" + `*#_\-，,。；;：:（）()“”\"'\\[\\]]+`)
	return strings.ToLower(re.ReplaceAllString(value, ""))
}

// buildDocumentModeNoEvidenceReply 构建文档模式无证据回复
func (o *PreparationOrchestratorImpl) buildDocumentModeNoEvidenceReply(question string, requiresFreshSearch bool) string {
	normalizedQuestion := strutil.Trim(question)

	if o.looksLikeCapabilityQuestion(normalizedQuestion) {
		return "当前你正在使用“当前文档问答”模式，我会优先基于所选文档回答。这个问题更像是在询问助手能力，而不是当前文档内容。如果你想了解我能做什么，请切换到“开放式提问”模式。"
	}

	if o.looksLikeOpenChatQuestion(normalizedQuestion, requiresFreshSearch) {
		return "当前你正在使用“当前文档问答”模式，我只能基于所选文档回答。这个问题更像开放式提问，例如天气、最新信息或一般交流。如果你想继续问这类问题，请切换到“开放式提问”模式。"
	}

	if o.properties.NoEvidenceReply != "" {
		return o.properties.NoEvidenceReply
	}

	return "当前没有从当前文档中检索到足够证据，暂时不能给出可靠结论。你可以补充更具体的标题、术语或关键词后再试。"
}

// looksLikeCapabilityQuestion 判断是否为能力询问
func (o *PreparationOrchestratorImpl) looksLikeCapabilityQuestion(normalizedQuestion string) bool {
	if normalizedQuestion == "" {
		return false
	}
	for _, hint := range capabilityHints {
		if strings.Contains(normalizedQuestion, hint) {
			return true
		}
	}
	return false
}

// looksLikeOpenChatQuestion 判断是否为开放式聊天问题
func (o *PreparationOrchestratorImpl) looksLikeOpenChatQuestion(normalizedQuestion string, requiresFreshSearch bool) bool {
	if normalizedQuestion == "" {
		return false
	}
	if requiresFreshSearch {
		return true
	}
	for _, hint := range openChatHints {
		if strings.Contains(normalizedQuestion, hint) {
			return true
		}
	}
	for _, hint := range chitchatHints {
		if strings.Contains(normalizedQuestion, hint) {
			return true
		}
	}
	return false
}

// extractDocumentIds 提取文档ID列表
func (o *PreparationOrchestratorImpl) extractDocumentIds(candidates []*vo.DocumentRouteCandidate) []int64 {
	result := make([]int64, 0, len(candidates))
	for _, doc := range candidates {
		if doc.DocumentId != "" {
			if id, err := strconv.ParseInt(doc.DocumentId, 10, 64); err == nil {
				result = append(result, id)
			}
		}
	}
	return result
}

// extractTaskIds 提取任务ID列表
func (o *PreparationOrchestratorImpl) extractTaskIds(candidates []*vo.DocumentRouteCandidate) []int64 {
	result := make([]int64, 0, len(candidates))
	for _, doc := range candidates {
		if doc.LastIndexTaskId != "" {
			if id, err := strconv.ParseInt(doc.LastIndexTaskId, 10, 64); err == nil {
				result = append(result, id)
			}
		}
	}
	return result
}

func (o *PreparationOrchestratorImpl) StartStage(ctx context.Context, convCtx *vo.ConversationContext) *vo.StageContext {
	return convCtx.Tracer.StartStage(vo.ConversationTraceStagePrepare, vo.ExecutionModeRetrieval.String(), "正在准备知识路由。", nil)
}

func Warnf(format string, v ...any) {
	logx.Alert(fmt.Sprintf(format, v...))
}
