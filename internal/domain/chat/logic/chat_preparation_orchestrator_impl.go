package logic

import (
	"context"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/duke-git/lancet/v2/stream"
	"github.com/duke-git/lancet/v2/strutil"
	"github.com/zeromicro/go-zero/core/logx"

	"github.com/swiftbit/know-agent/internal/config"
	"github.com/swiftbit/know-agent/internal/domain/chat/adapter"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
	"github.com/swiftbit/know-agent/internal/domain/chat/support"
	"github.com/swiftbit/know-agent/internal/domain/knowledge/logic"
	klvo "github.com/swiftbit/know-agent/internal/domain/rag/model/vo"
)

var (
	capabilityHints = []string{"你都能干什么", "你能做什么", "你可以做什么", "你会什么", "你是谁", "怎么用你", "你能帮我什么"}
	openChatHints   = []string{"天气", "温度", "下雨", "新闻", "股价", "汇率", "热搜", "今天", "明天", "最新", "现在"}
	chitchatHints   = []string{"你好", "您好", "hello", "hi", "谢谢", "感谢", "再见", "拜拜"}
)

// ChatPreparationOrchestratorImpl 聊天准备编排器实现
type ChatPreparationOrchestratorImpl struct {
	repo                   adapter.ChatRepository
	properties             config.RagConf
	sessionMemoryLogic     SessionMemoryLogic
	queryRewriteLogic      QueryRewriteLogic
	documentQuestionRouter DocumentQuestionRouter
	knowledgeRouteService  KnowledgeRouteService
	documentKnowledgeLogic logic.DocumentKnowledgeLogic
}

// NewChatPreparationOrchestrator 创建聊天准备编排器实例
func NewChatPreparationOrchestrator(
	repo adapter.ChatRepository,
	properties config.RagConf,
	sessionMemoryLogic SessionMemoryLogic,
	queryRewriteLogic QueryRewriteLogic,
	documentQuestionRouter DocumentQuestionRouter,
	knowledgeRouteService KnowledgeRouteService,
	documentKnowledgeLogic logic.DocumentKnowledgeLogic,
) *ChatPreparationOrchestratorImpl {
	return &ChatPreparationOrchestratorImpl{
		repo:                   repo,
		properties:             properties,
		sessionMemoryLogic:     sessionMemoryLogic,
		queryRewriteLogic:      queryRewriteLogic,
		documentQuestionRouter: documentQuestionRouter,
		knowledgeRouteService:  knowledgeRouteService,
		documentKnowledgeLogic: documentKnowledgeLogic,
	}
}

// Prepare 准备对话执行计划
func (o *ChatPreparationOrchestratorImpl) Prepare(ctx context.Context, convCtx *vo.ConversationContext) (*vo.ConversationExecutionPlan, error) {
	conversationId := convCtx.ConversationId
	question := convCtx.Question
	chatMode := convCtx.ChatMode
	selectedDocumentId := convCtx.SelectedDocumentId
	selectedDocumentName := convCtx.SelectedDocumentName
	selectedTaskId := convCtx.SelectedTaskId
	currentDate := convCtx.CurrentDate
	currentDateText := convCtx.CurrentDateText
	tracer := convCtx.Tracer

	// 装载会话记忆
	memoryContext, err := o.summarizeHistory(ctx, convCtx)
	if err != nil {
		return nil, err
	}

	historyPlanningContext, historySummary, answerHistoryContext := o.buildHistoryContext(memoryContext, question)

	// 判断时间敏感和实时搜索需求
	requiresCurrentDateAnchoring := support.RequiresCurrentDateAnchoring(question)
	requiresRealTimeSearch := support.RequiresRealTimeSearch(question)

	// 处理开放式聊天模式
	if chatMode == vo.ChatQueryModeOpenChat {
		plan := o.basePlan(question, chatMode, memoryContext, historyPlanningContext, historySummary, answerHistoryContext, currentDate, currentDateText, requiresCurrentDateAnchoring, requiresFreshSearch)
		plan.Mode = vo.ExecutionModeReactAgent

		if tracer != nil {
			routeStage := tracer.StartStage(vo.ConversationTraceStageRewrite, vo.ExecutionModeReactAgent.String(), "路由到开放式 Agent。", nil)
			tracer.CompleteStage(routeStage, "已判定走开放式 Agent 路径。", map[string]interface{}{
				"chatMode":                     chatMode.String(),
				"executionMode":                vo.ExecutionModeReactAgent.String(),
				"requiresFreshSearch":          requiresRealTimeSearch,
				"requiresCurrentDateAnchoring": requiresCurrentDateAnchoring,
			})
		}
		return plan, nil
	}

	// 验证RAG启用状态
	if !o.properties.Enabled {
		return nil, fmt.Errorf("当前文档问答模式未启用，请先开启聊天侧 RAG 编排")
	}

	// 验证文档模式参数
	if chatMode == vo.ChatQueryModeDocument && (selectedDocumentId == nil || selectedTaskId == nil) {
		return nil, fmt.Errorf("当前文档问答模式缺少有效的文档范围")
	}

	// 问题改写阶段
	rewriteResult, err := o.questionRewrite(ctx, convCtx, historySummary)

	// 处理文档路由
	routedDocumentId := selectedDocumentId
	routedDocumentName := selectedDocumentName
	routedTaskId := selectedTaskId

	var routedDocumentIds []int64
	if routedDocumentId != nil {
		routedDocumentIds = []int64{*routedDocumentId}
	} else {
		routedDocumentIds = []int64{}
	}

	var routedTaskIds []int64
	if routedTaskId != nil {
		routedTaskIds = []int64{*routedTaskId}
	} else {
		routedTaskIds = []int64{}
	}

	// 自动文档模式下的路由
	if chatMode == vo.ChatQueryModeAutoDocument {
		routeDecision, err := o.knowledgeRouteService.Route(ctx, question, rewriteQuestion)
		if err != nil {
			Warnf("知识路由失败: %v", err)
			routeDecision = nil
		}

		err = o.knowledgeRouteService.RecordAutoRoute(ctx, conversationId, taskInfo.ExchangeId, question, rewriteQuestion, routeDecision)
		if err != nil {
			Warnf("记录自动路由失败: %v", err)
		}

		candidateDocuments := o.selectAutoCandidates(ctx, routeDecision, question, rewriteQuestion)

		if o.shouldAskClarification(routeDecision, candidateDocuments) {
			plan := o.basePlan(question, chatMode, memoryContext, historyPlanningContext, historySummary, answerHistoryContext, currentDate, currentDateText, requiresCurrentDateAnchoring, requiresFreshSearch)
			plan.Mode = vo.ExecutionModeClarification
			plan.RewriteQuestion = rewriteQuestion
			plan.RewriteSubQuestions = rewriteSubQuestions
			plan.RetrievalQuestion = rewriteQuestion
			plan.RetrievalSubQuestions = rewriteSubQuestions
			plan.RetrievalDocumentIds = o.extractDocumentIds(candidateDocuments)
			plan.RetrievalTaskIds = o.extractTaskIds(candidateDocuments)
			plan.ClarificationReply = o.buildClarificationReply(question, routeDecision, candidateDocuments)
			plan.ClarificationOptions = o.buildClarificationOptions(candidateDocuments)
			plan.ClarificationReason = o.buildClarificationReason(routeDecision, candidateDocuments)
			return plan, nil
		}

		// 选择最置信的文档
		confidentTopDocument := routeDecision != nil && routeDecision.Confidence >= 0.55
		var topDocument *vo.DocumentRouteCandidate
		if confidentTopDocument && len(candidateDocuments) > 0 {
			topDocument = candidateDocuments[0]
		}

		if topDocument != nil && topDocument.DocumentId != "" && topDocument.LastIndexTaskId != "" {
			id, _ := strconv.ParseInt(topDocument.DocumentId, 10, 64)
			routedDocumentId = &id
			routedDocumentName = topDocument.DocumentName
			taskId, _ := strconv.ParseInt(topDocument.LastIndexTaskId, 10, 64)
			routedTaskId = &taskId
		} else {
			routedDocumentId = nil
			routedDocumentName = ""
			routedTaskId = nil
		}

		routedDocumentIds = o.extractDocumentIds(candidateDocuments)
		routedTaskIds = o.extractTaskIds(candidateDocuments)

		if tracer != nil && routeDecision != nil {
			tracer.CompleteStage(
				tracer.StartStage(vo.StageCodeRoute, "AUTO_DOCUMENT", "正在生成知识范围候选。", nil),
				"知识范围路由完成。",
				map[string]interface{}{
					"confidence":             routeDecision.Confidence,
					"routeStatus":            routeDecision.RouteStatus,
					"candidateDocumentCount": len(candidateDocuments),
					"confidentTopDocument":   confidentTopDocument,
					"topDocumentId":          strutil.Trim(topDocument.DocumentId),
					"topDocumentName":        strutil.Trim(topDocument.DocumentName),
				},
			)
		}
	} else if chatMode == vo.ChatQueryModeDocument && selectedDocumentId != nil {
		err := o.knowledgeRouteService.RecordShadowRoute(ctx, conversationId, taskInfo.ExchangeId, *selectedDocumentId, question, rewriteQuestion)
		if err != nil {
			Warnf("记录影子路由失败: %v", err)
		}
	}

	// 文档内导航路由
	var navigationDecision *vo.DocumentNavigationDecision
	var routeStage *vo.StageHandle
	if tracer != nil {
		routeStage = tracer.StartStage(vo.StageCodeRoute, vo.ExecutionModeRetrieval.String(), "正在判定图查询还是混合检索。", nil)
	}
	try(func() {
		var err error
		navigationDecision, err = o.documentQuestionRouter.Route(ctx, routedDocumentId, question, rewriteResult)
		if err != nil {
			panic(err)
		}
		if tracer != nil && routeStage != nil {
			tracer.CompleteStage(routeStage, "执行路由完成。", map[string]interface{}{
				"executionMode":     navigationDecision.ExecutionMode.String(),
				"targetSectionHint": strutil.Trim(navigationDecision.StructureAnchor.AnchorName),
				"navigationSummary": strutil.Trim(navigationDecision.SummaryText),
			})
		}
	}, func(err error) {
		if tracer != nil && routeStage != nil {
			tracer.FailStage(routeStage, "执行路由失败。", err, nil)
		}
	})

	// 确定执行模式和检索问题
	executionMode := vo.ExecutionModeRetrieval
	if navigationDecision != nil && navigationDecision.ExecutionMode != 0 {
		executionMode = navigationDecision.ExecutionMode
	}

	retrievalQuestion := rewriteQuestion
	if navigationDecision != nil && navigationDecision.RetrievalPlan != nil {
		retrievalQuestion = o.firstNonBlank(navigationDecision.RetrievalPlan.RetrievalQuestion, rewriteQuestion)
	}

	retrievalSubQuestions := rewriteSubQuestions
	if navigationDecision != nil && navigationDecision.RetrievalPlan != nil && len(navigationDecision.RetrievalPlan.SubQuestions) > 0 {
		retrievalSubQuestions = navigationDecision.RetrievalPlan.SubQuestions
	}

	logx.Infof("聊天编排完成: conversationId=%s, chatMode=%s, originalQuestion='%s', rewriteQuestion='%s', retrievalQuestion='%s', executionMode=%s",
		conversationId, chatMode.String(), strutil.Trim(question), rewriteQuestion, retrievalQuestion, executionMode.String())

	// 构建最终执行计划
	plan := o.basePlan(question, chatMode, memoryContext, historyPlanningContext, historySummary, answerHistoryContext, currentDate, currentDateText, requiresCurrentDateAnchoring, requiresFreshSearch)
	plan.Mode = executionMode
	plan.NavigationDecision = navigationDecision
	plan.RewriteQuestion = rewriteQuestion
	plan.RewriteSubQuestions = rewriteSubQuestions
	plan.RetrievalQuestion = retrievalQuestion
	plan.RetrievalSubQuestions = retrievalSubQuestions
	plan.SelectedDocumentId = routedDocumentId
	plan.SelectedDocumentName = routedDocumentName
	plan.SelectedTaskId = routedTaskId
	plan.RetrievalDocumentIds = routedDocumentIds
	plan.RetrievalTaskIds = routedTaskIds
	plan.NoEvidenceReply = o.buildDocumentModeNoEvidenceReply(question, requiresFreshSearch)

	return plan, nil
}

func (o *ChatPreparationOrchestratorImpl) buildHistoryContext(memoryContext *vo.MemoryContext, question string) (vo.HistoryPlanningContext, string, *vo.QuestionHistoryContext) {
	// 构建历史规划上下文
	historyPlanningContext := vo.NewHistoryPlanningContext(memoryContext.Summary)
	historySummary := o.buildPlanningHistory(memoryContext, historyPlanningContext)
	answerHistoryContext := o.buildQuestionHistoryContext(question, strutil.Trim(memoryContext.RecentTranscript))
	return historyPlanningContext, historySummary, answerHistoryContext
}

// summarizeHistory 构建会话记忆
func (o *ChatPreparationOrchestratorImpl) summarizeHistory(ctx context.Context, convCtx *vo.ConversationContext) (*vo.MemoryContext, error) {
	tracer := convCtx.Tracer
	memoryStage := tracer.StartStage(vo.ConversationTraceStageMemory, convCtx.ChatMode.Name(), "正在装载会话记忆与最近窗口。", nil)
	if err := o.repo.InsertStage(ctx, tracer.ConvChatExchangeTraceStage()); err != nil {
		return nil, err
	}

	memoryContext, err := o.sessionMemoryLogic.LoadMemoryContext(ctx, convCtx.ConversationId, tracer)
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

func (o *ChatPreparationOrchestratorImpl) questionRewrite(ctx context.Context, convCtx *vo.ConversationContext, historySummary string) (*vo.RagRewriteResult, error) {
	tracer := convCtx.Tracer
	rewriteStage := tracer.StartStage(vo.ConversationTraceStageRewrite, vo.ExecutionModeRetrieval.String(), "正在生成检索友好的问题表达。", o.buildRewriteStageSnapshot(convCtx.Question, historySummary, nil))
	rewriteResult, err := o.queryRewriteLogic.Rewrite(ctx, convCtx.Question, historySummary, tracer)
	if err != nil {
		tracer.FailStage(rewriteStage, "问题改写失败。", err, o.buildRewriteStageSnapshot(convCtx.Question, historySummary, nil))
	}
	tracer.CompleteStage(rewriteStage, "问题改写完成。", o.buildRewriteStageSnapshot(convCtx.Question, historySummary, rewriteResult))

	// 获取改写后的问题
	rewriteQuestion := o.firstNonBlank(rewriteResult.RewrittenQuestion, strutil.Trim(convCtx.Question))
	rewriteSubQuestions := rewriteResult.SubQuestions
	if len(rewriteSubQuestions) == 0 {
		rewriteSubQuestions = []string{rewriteQuestion}
	}
	return rewriteResult, err
}

// basePlan 构建基础执行计划
func (o *ChatPreparationOrchestratorImpl) basePlan(question string, chatMode vo.ChatQueryMode, memoryContext *vo.MemoryContext,
	historyPlanningContext vo.HistoryPlanningContext, historySummary string, answerHistoryContext *vo.QuestionHistoryContext,
	currentDate time.Time, currentDateText string, requiresCurrentDateAnchoring, requiresFreshSearch bool) *vo.ConversationExecutionPlan {

	return &vo.ConversationExecutionPlan{
		ChatMode:                     chatMode,
		OriginalQuestion:             question,
		AgentQuestion:                question,
		RewriteQuestion:              question,
		RewriteSubQuestions:          []string{question},
		RetrievalQuestion:            question,
		RetrievalSubQuestions:        []string{question},
		HistorySummary:               historySummary,
		LongTermSummary:              memoryContext.LongTermSummary,
		HistoryPlanningContext:       historyPlanningContext,
		RecentHistoryTranscript:      memoryContext.RecentTranscript,
		AnswerRecentTranscript:       memoryContext.RecentTranscript,
		QuestionHistoryContext:       answerHistoryContext,
		HistoryCompressionApplied:    memoryContext.CompressionApplied,
		HistoryCoveredExchangeId:     &memoryContext.CoveredExchangeId,
		HistoryCoveredExchangeCount:  &memoryContext.CoveredExchangeCount,
		HistoryCompressionCount:      &memoryContext.CompressionCount,
		CurrentDate:                  currentDate,
		CurrentDateText:              currentDateText,
		RequiresCurrentDateAnchoring: requiresCurrentDateAnchoring,
		RequiresFreshSearch:          requiresFreshSearch,
		NoEvidenceReply:              o.properties.NoEvidenceReply,
	}
}

// buildRewriteStageSnapshot 构建改写阶段快照
func (o *ChatPreparationOrchestratorImpl) buildRewriteStageSnapshot(question, historySummary string, rewriteResult *vo.RagRewriteResult) map[string]interface{} {
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
func (o *ChatPreparationOrchestratorImpl) buildPlanningHistory(memoryContext *vo.MemoryContext, historyPlanningContext *vo.HistoryPlanningContext) string {
	var sb strings.Builder
	o.appendSection(&sb, "会话目标", historyPlanningContext.ConversationGoal)
	o.appendBulletSection(&sb, "已确认事实", historyPlanningContext.StableFacts)
	o.appendBulletSection(&sb, "待跟进问题", historyPlanningContext.PendingQuestions)
	o.appendBulletSection(&sb, "检索提示", historyPlanningContext.RetrievalHints)
	structuredHistory := strings.TrimSpace(sb.String())
	recentTranscript := strutil.Trim(memoryContext.RecentTranscript)

	maxChars := max(1, o.properties.PlanningHistoryMaxChars)
	if recentTranscript == "" {
		return o.clipHead(structuredHistory, maxChars)
	}

	recentBudget := int(math.Round(float64(maxChars) * 0.65))
	recentPart := o.clipTail(recentTranscript, recentBudget)

	structuredBudget := max(0, maxChars-len(recentPart)-2)
	structuredPart := o.clipHead(structuredHistory, structuredBudget)

	return o.joinNonBlank(structuredPart, recentPart)
}

// buildQuestionHistoryContext 构建回答历史上下文
func (o *ChatPreparationOrchestratorImpl) buildQuestionHistoryContext(question, questionRecentTranscript string) *vo.QuestionHistoryContext {
	return support.Assemble(question, questionRecentTranscript)
}

// appendSection 追加章节
func (o *ChatPreparationOrchestratorImpl) appendSection(sb *strings.Builder, title, content string) {
	if content == "" {
		return
	}
	if sb.Len() > 0 {
		sb.WriteString("\n")
	}
	sb.WriteString("【")
	sb.WriteString(title)
	sb.WriteString("】\n")
	sb.WriteString(strings.TrimSpace(content))
	sb.WriteString("\n")
}

// appendBulletSection 追加带项目符号的章节
func (o *ChatPreparationOrchestratorImpl) appendBulletSection(sb *strings.Builder, title string, values []string) {
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

// clipHead 截取头部
func (o *ChatPreparationOrchestratorImpl) clipHead(text string, maxChars int) string {
	normalized := []rune(strutil.Trim(text))
	if len(normalized) <= maxChars {
		return string(normalized)
	}
	if maxChars <= 1 {
		return ""
	}
	return string(normalized[:maxChars-1]) + "…"
}

// clipTail 截取尾部
func (o *ChatPreparationOrchestratorImpl) clipTail(text string, maxChars int) string {
	normalized := []rune(strutil.Trim(text))
	if len(normalized) <= maxChars {
		return string(normalized)
	}
	if maxChars <= 1 {
		return ""
	}
	start := max(0, len(normalized)-maxChars+1)
	return "…" + string(normalized[start:])
}

// joinNonBlank 连接非空字符串
func (o *ChatPreparationOrchestratorImpl) joinNonBlank(left, right string) string {
	if left == "" {
		return strutil.Trim(right)
	}
	if right == "" {
		return strutil.Trim(left)
	}
	return strutil.Trim(left) + "\n\n" + strutil.Trim(left)
}

// firstNonBlank 返回第一个非空字符串
func (o *ChatPreparationOrchestratorImpl) firstNonBlank(left, right string) string {
	if strings.TrimSpace(left) != "" {
		return strings.TrimSpace(left)
	}
	return strutil.Trim(right)
}

// selectAutoCandidates 选择自动候选文档
func (o *ChatPreparationOrchestratorImpl) selectAutoCandidates(ctx context.Context, routeDecision *vo.KnowledgeRouteDecision, question, rewriteQuestion string) []*vo.DocumentRouteCandidate {
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
func (o *ChatPreparationOrchestratorImpl) fallbackDocuments(ctx context.Context, question, rewriteQuestion string, limit int) []*vo.DocumentRouteCandidate {
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
func (o *ChatPreparationOrchestratorImpl) mergeCandidates(primary, secondary []*vo.DocumentRouteCandidate, limit int) []*vo.DocumentRouteCandidate {
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
func (o *ChatPreparationOrchestratorImpl) shouldAskClarification(routeDecision *vo.KnowledgeRouteDecision, candidateDocuments []*vo.DocumentRouteCandidate) bool {
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
func (o *ChatPreparationOrchestratorImpl) buildClarificationReply(originalQuestion string, routeDecision *vo.KnowledgeRouteDecision, candidateDocuments []*vo.DocumentRouteCandidate) string {
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
func (o *ChatPreparationOrchestratorImpl) buildClarificationOptions(candidateDocuments []*vo.DocumentRouteCandidate) []string {
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
func (o *ChatPreparationOrchestratorImpl) buildClarificationReason(routeDecision *vo.KnowledgeRouteDecision, candidateDocuments []*vo.DocumentRouteCandidate) string {
	if routeDecision == nil || len(routeDecision.Documents) == 0 {
		return "当前自动知识路由没有形成稳定候选，已改为先向用户确认文档范围。"
	}

	confidenceText := strconv.FormatFloat(routeDecision.Confidence, 'f', -1, 64)
	candidateCount := len(candidateDocuments)

	return "当前自动知识路由置信度为 " + confidenceText + "，候选文档数为 " + strconv.Itoa(candidateCount) + "，为避免误选文档，先返回澄清问题。"
}

// extractFallbackTerms 提取后备检索词
func (o *ChatPreparationOrchestratorImpl) extractFallbackTerms(question, rewriteQuestion string) []string {
	terms := make(map[string]bool)
	routingText := strutil.Trim(question) + " " + strutil.Trim(rewriteQuestion)

	re := regexp.MustCompile(`[\s、，,；;：:（）()\-的和及与或]+`)
	segments := re.Split(routingText, -1)

	for _, segment := range segments {
		trimmed := strings.TrimSpace(segment)
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
func (o *ChatPreparationOrchestratorImpl) fallbackDescriptorScore(descriptor *klvo.KnowledgeDocument, queryTerms []string) float64 {
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
func (o *ChatPreparationOrchestratorImpl) normalizeFallbackText(value string) string {
	if value == "" {
		return ""
	}
	re := regexp.MustCompile(`[\s>\` + "`" + `*#_\-，,。；;：:（）()“”\"'\\[\\]]+`)
	return strings.ToLower(re.ReplaceAllString(value, ""))
}

// buildDocumentModeNoEvidenceReply 构建文档模式无证据回复
func (o *ChatPreparationOrchestratorImpl) buildDocumentModeNoEvidenceReply(question string, requiresFreshSearch bool) string {
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
func (o *ChatPreparationOrchestratorImpl) looksLikeCapabilityQuestion(normalizedQuestion string) bool {
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
func (o *ChatPreparationOrchestratorImpl) looksLikeOpenChatQuestion(normalizedQuestion string, requiresFreshSearch bool) bool {
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
func (o *ChatPreparationOrchestratorImpl) extractDocumentIds(candidates []*vo.DocumentRouteCandidate) []int64 {
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
func (o *ChatPreparationOrchestratorImpl) extractTaskIds(candidates []*vo.DocumentRouteCandidate) []int64 {
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

func (o *ChatPreparationOrchestratorImpl) StartStage(ctx context.Context, convCtx *vo.ConversationContext) *vo.StageContext {
	return convCtx.Tracer.StartStage(vo.ConversationTraceStagePrepare, vo.ExecutionModeRetrieval.String(), "正在准备知识路由。", nil)
}

func Warnf(format string, v ...any) {
	logx.Alert(fmt.Sprintf(format, v...))
}
