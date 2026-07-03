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

	"github.com/duke-git/lancet/v2/maputil"
	"github.com/duke-git/lancet/v2/stream"
	"github.com/duke-git/lancet/v2/strutil"
	"github.com/zeromicro/go-zero/core/logx"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/chat/adapter"
	logic2 "github.com/swiftbit/know-agent/internal/domain/chat/logic"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/trace"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
	"github.com/swiftbit/know-agent/internal/domain/chat/support"
	"github.com/swiftbit/know-agent/internal/domain/knowledge/logic"
	klvo "github.com/swiftbit/know-agent/internal/domain/knowledge/model/vo"
	rvo "github.com/swiftbit/know-agent/internal/domain/rag/model/vo"
	"github.com/swiftbit/know-agent/internal/svc"
)

var (
	capabilityHints    = []string{"你都能干什么", "你能做什么", "你可以做什么", "你会什么", "你是谁", "怎么用你", "你能帮我什么"}
	openChatHints      = []string{"天气", "温度", "下雨", "新闻", "股价", "汇率", "热搜", "今天", "明天", "最新", "现在"}
	chitchatHints      = []string{"你好", "您好", "hello", "hi", "谢谢", "感谢", "再见", "拜拜"}
	fallbackCleanRegex = regexp.MustCompile(`[\s>\` + "`" + `*#_\-，,。；;：:（）()“”\"'\\[\\]]+`)
	fallbackSplitRegex = regexp.MustCompile(`[\s、，,；;：:（）()\-的和及与或]+`)
)

// PreparationOrchestratorImpl 聊天准备编排器实现
type PreparationOrchestratorImpl struct {
	repo                   adapter.ChatRepository
	memoryLogic            logic2.SessionMemoryLogic
	rewriteLogic           logic2.QueryRewriteLogic
	documentQuestionRouter logic2.DocumentQuestionRouteLogic
	knowledgeRouteLogic    logic.KnowledgeRouteLogic
	knowledgeLogic         logic.KnowledgeLogic
	tracer                 *trace.ConversationTraceRecorder
	*option
}

type option struct {
	ragEnabled              bool    // 是否启用rag
	planningHistoryMaxChars int     // 规划历史最大字符数
	questionHistoryMaxChars int     // 问题历史最大字符数
	noEvidenceReply         string  // 无证据回复
	rewriteEnabled          bool    // 是否启用问题改写
	maxSubQuestions         int     // 最大子问题数量
	temperature             float32 // 温度参数
	topP                    float32 // TopP参数
	thinking                bool    // 是否启用思考过程
}

// NewChatPreparationOrchestrator 创建聊天准备编排器实例
func NewChatPreparationOrchestrator(svcCtx *svc.ServiceContext,
	repo adapter.ChatRepository,
	memoryLogic logic2.SessionMemoryLogic,
	rewriteLogic logic2.QueryRewriteLogic,
	documentQuestionRouter logic2.DocumentQuestionRouteLogic,
	knowledgeRoute logic.KnowledgeRouteLogic,
	knowledgeLogic logic.KnowledgeLogic,
) *PreparationOrchestratorImpl {
	return &PreparationOrchestratorImpl{
		repo:                   repo,
		memoryLogic:            memoryLogic,
		rewriteLogic:           rewriteLogic,
		documentQuestionRouter: documentQuestionRouter,
		knowledgeRouteLogic:    knowledgeRoute,
		knowledgeLogic:         knowledgeLogic,
		tracer:                 trace.NewConversationTraceRecorder(repo),
		option: &option{
			ragEnabled:              svcCtx.Config.Chat.Rag.Enabled,
			planningHistoryMaxChars: svcCtx.Config.Chat.Rag.PlanningHistoryMaxChars,
			questionHistoryMaxChars: max(1, svcCtx.Config.Chat.Rag.QuestionHistoryMaxChars),
			noEvidenceReply:         utils.BlankToDefault(svcCtx.Config.Chat.Rag.NoEvidenceReply, "当前没有从已接入文档中检索到足够证据，暂时不能给出可靠结论。"),
			rewriteEnabled:          svcCtx.Config.Chat.Rewrite.Enabled,
			maxSubQuestions:         svcCtx.Config.Chat.Rewrite.MaxSubQuestions,
			temperature:             svcCtx.Config.Chat.Rewrite.Temperature,
			thinking:                svcCtx.Config.Chat.Rewrite.Thinking,
			topP:                    svcCtx.Config.Chat.Rewrite.TopP,
		},
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
		err = o.prepareOpenChat(ctx, convCtx, execPlan)
	case vo.ChatQueryModeDocument:
		err = o.prepareDocumentMode(ctx, convCtx, execPlan)
	case vo.ChatQueryModeAutoDocument:
		err = o.prepareAutoDocumentMode(ctx, convCtx, execPlan)
	default:
		return nil, fmt.Errorf("不支持的聊天模式: %s", convCtx.ChatMode.Name())
	}
	if err != nil {
		return nil, err
	}
	return execPlan, err
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
		execPlan.RewriteQuestion = rewriteResult.RewrittenQuestion
		execPlan.RewriteSubQuestions = rewriteResult.SubQuestions
		execPlan.RetrievalQuestion = rewriteResult.RewrittenQuestion
		execPlan.RetrievalSubQuestions = rewriteResult.SubQuestions
	}
	return execPlan, nil
}

// prepareOpenChat 开放式聊天：直接返回 ReactAgent 模式执行计划
func (o *PreparationOrchestratorImpl) prepareOpenChat(ctx context.Context, convCtx *vo.ConversationContext, execPlan *vo.ConversationExecutionPlan) error {
	execPlan.Mode = vo.ExecutionModeReactAgent
	stage, err := o.tracer.StartStage(ctx, convCtx.Trace, vo.ConversationTraceStageRewrite, vo.ExecutionModeReactAgent.String(), "路由到开放式 Agent。", nil)
	if err != nil {
		return err
	}
	snapshot := map[string]any{
		"chatMode":                     convCtx.ChatMode.String(),
		"executionMode":                vo.ExecutionModeReactAgent.String(),
		"requiresRealTimeSearch":       execPlan.RequiresRealTimeSearch,
		"requiresCurrentDateAnchoring": execPlan.RequiresCurrentDateAnchoring,
	}
	if err = o.tracer.CompleteStage(ctx, stage, "已判定走开放式 Agent 路径。", snapshot); err != nil {
		return err
	}

	return nil
}

// prepareDocumentMode 指定文档问答：记录影子路由 → 文档内导航 → 生成执行计划。
func (o *PreparationOrchestratorImpl) prepareDocumentMode(ctx context.Context, convCtx *vo.ConversationContext, execPlan *vo.ConversationExecutionPlan) error {
	if convCtx.SelectedDocumentId == 0 || convCtx.SelectedTaskId == 0 {
		return fmt.Errorf("当前文档问答模式缺少有效的文档范围")
	}

	// 记录影子路由（不影响业务结果，失败仅告警）
	if err := o.knowledgeRouteLogic.RecordShadowRoute(ctx, convCtx.ExchangeId, convCtx.ConversationId, convCtx.SelectedDocumentId, convCtx.Question, execPlan.RewriteQuestion); err != nil {
		Warnf("记录影子路由失败: %v", err)
	}

	return o.funcName(ctx, convCtx, execPlan)
}

// prepareAutoDocumentMode 自动文档问答：执行知识路由 → 必要时澄清 → 确定主文档 → 导航 → 生成计划。
func (o *PreparationOrchestratorImpl) prepareAutoDocumentMode(ctx context.Context, convCtx *vo.ConversationContext, execPlan *vo.ConversationExecutionPlan) error {
	// 路由跟踪
	stage, err := o.tracer.StartStage(ctx, convCtx.Trace, vo.ConversationTraceStageRoute, "AUTO_DOCUMENT", "正在生成知识范围候选。", nil)
	if err != nil {
		return err
	}

	// 执行知识路由
	routeDecision, err := o.knowledgeRouteLogic.Route(ctx, convCtx.Question, execPlan.RewriteQuestion)
	if err != nil {
		routeDecision = &klvo.KnowledgeRouteDecision{}
		Warnf("知识路由失败: %v", err)
	}
	if err = o.knowledgeRouteLogic.RecordAutoRoute(ctx, convCtx.ExchangeId, convCtx.ConversationId, convCtx.Question, execPlan.RewriteQuestion, routeDecision); err != nil {
		Warnf("记录自动路由失败: %v", err)
	}

	// 选择候选文档（含低置信度时的 fallback）
	candidateDocuments := o.selectAutoCandidates(ctx, routeDecision, convCtx.Question, execPlan.RewriteQuestion)
	execPlan.RetrievalDocumentIds = o.extractDocumentIds(candidateDocuments)
	execPlan.RetrievalTaskIds = o.extractTaskIds(candidateDocuments)

	// 选择最高置信度的文档作为主文档；否则回退到“不指定主文档，使用多文档范围检索”
	topDocument := &klvo.DocumentRouteCandidate{}
	confidentTop := routeDecision.Confidence >= 0.55
	if confidentTop && len(candidateDocuments) > 0 {
		topDocument = candidateDocuments[0]
	}

	snapshot := map[string]any{
		"confidence":             routeDecision.Confidence,
		"routeStatus":            routeDecision.RouteStatus,
		"candidateDocumentCount": len(candidateDocuments),
		"confidentTopDocument":   confidentTop,
		"topDocumentId":          topDocument.DocumentId,
		"topDocumentName":        topDocument.DocumentName,
	}
	if err = o.tracer.CompleteStage(ctx, stage, "知识范围路由完成。", snapshot); err != nil {
		return err
	}

	// 若需要澄清，直接返回澄清模式执行计划
	if o.shouldAskClarification(routeDecision, candidateDocuments) {
		execPlan.Mode = vo.ExecutionModeClarification
		execPlan.ClarificationReply = o.buildClarificationReply(candidateDocuments)
		execPlan.ClarificationOptions = o.buildClarificationOptions(candidateDocuments)
		execPlan.ClarificationReason = o.buildClarificationReason(routeDecision, candidateDocuments)
		return nil
	}

	execPlan.SelectedDocumentId = topDocument.DocumentId
	execPlan.SelectedDocumentName = topDocument.DocumentName
	execPlan.SelectedTaskId = topDocument.LastIndexTaskId

	if err = o.funcName(ctx, convCtx, execPlan); err != nil {
		return err

	}
	return nil
}

func (o *PreparationOrchestratorImpl) funcName(ctx context.Context, convCtx *vo.ConversationContext, execPlan *vo.ConversationExecutionPlan) error {
	// 文档内导航
	stage, err := o.tracer.StartStage(ctx, convCtx.Trace, vo.ConversationTraceStageRoute, vo.ExecutionModeRetrieval.Name(), "正在判定图查询还是混合检索。", nil)
	if err != nil {
		return err
	}

	navigationDecision, err := o.documentQuestionRouter.Route(ctx, execPlan.SelectedDocumentId, convCtx.Question, prepCtx.rewriteResult)
	if err != nil {
		if err = o.tracer.FailStage(ctx, stage, "执行路由失败。", err, nil); err != nil {
			return err
		}
		return err
	}
	snapshot := map[string]any{
		"executionMode":     navigationDecision.ExecutionMode.Name(),
		"targetSectionHint": strutil.Trim(navigationDecision.StructureAnchor.AnchorName),
		"targetItemIndex":   navigationDecision.ItemAnchor.ItemIndex,
		"navigationSummary": strutil.Trim(navigationDecision.SummaryText),
	}
	if err = o.tracer.CompleteStage(ctx, stage, "执行路由完成。", snapshot); err != nil {
		return err
	}

	// 确定检索问题
	if navigationDecision.RetrievalPlan != nil {
		execPlan.RetrievalQuestion = utils.BlankToDefault(navigationDecision.RetrievalPlan.RetrievalQuestion, execPlan.RewriteQuestion)
		if len(navigationDecision.RetrievalPlan.SubQuestions) > 0 {
			execPlan.RetrievalSubQuestions = navigationDecision.RetrievalPlan.SubQuestions
		}
	}

	// 构建最终执行计划
	execPlan.Mode = navigationDecision.ExecutionMode
	execPlan.NavigationDecision = navigationDecision
	execPlan.NoEvidenceReply = o.buildDocumentModeNoEvidenceReply(convCtx.Question, execPlan.RequiresRealTimeSearch)

	logx.Infof("聊天编排完成: conversationId=%s, chatMode=%s, originalQuestion='%s', rewriteQuestion='%s', retrievalQuestion='%s', executionMode=%s, targetSection='%s",
		convCtx.ConversationId, convCtx.ChatMode.String(), strutil.Trim(convCtx.Question),
		execPlan.RewriteQuestion, execPlan.RetrievalQuestion, execPlan.Mode.Name(), navigationDecision.StructureAnchor.TargetSectionHint)

	return nil
}

// summarizeHistory 构建会话记忆
func (o *PreparationOrchestratorImpl) summarizeHistory(ctx context.Context, convCtx *vo.ConversationContext) (*vo.MemoryContext, error) {
	memoryStage, err := o.tracer.StartStage(ctx, convCtx.Trace, vo.ConversationTraceStageMemory, convCtx.ChatMode.Name(), "正在装载会话记忆与最近窗口。", nil)
	if err != nil {
		return nil, err
	}

	memoryContext, err := o.memoryLogic.LoadMemoryContext(ctx, convCtx.ConversationId, convCtx.Trace)
	if err != nil {
		if err = o.tracer.FailStage(ctx, memoryStage, "会话记忆装载失败。", err, nil); err != nil {
			return nil, err
		}
		return nil, err
	}
	snapshot := map[string]any{
		"compressionApplied":       memoryContext.CompressionApplied,
		"coveredExchangeId":        memoryContext.CoveredExchangeId,
		"coveredExchangeCount":     memoryContext.CoveredExchangeCount,
		"compressionCount":         memoryContext.CompressionCount,
		"longTermSummary":          strutil.Trim(memoryContext.LongTermSummary),
		"recentTranscript":         strutil.Trim(memoryContext.RecentTranscript),
		"RecentQuestionTranscript": strutil.Trim(memoryContext.RecentQuestionTranscript),
	}
	if err = o.tracer.CompleteStage(ctx, memoryStage, "会话记忆装载完成。", snapshot); err != nil {
		return nil, err
	}
	return memoryContext, nil
}

func (o *PreparationOrchestratorImpl) questionRewrite(ctx context.Context, convCtx *vo.ConversationContext, historySummary string) (*vo.RagRewriteResult, error) {
	rewriteStage, err := o.tracer.StartStage(ctx, convCtx.Trace, vo.ConversationTraceStageRewrite, vo.ExecutionModeRetrieval.String(), "正在生成检索友好的问题表达。", o.buildRewriteStageSnapshot(convCtx.Question, historySummary, nil))
	if err != nil {
		return nil, err
	}
	rewriteResult, err := o.rewriteLogic.Rewrite(ctx, convCtx.Question, historySummary, convCtx.Trace)
	if err != nil {
		if err = o.tracer.FailStage(ctx, rewriteStage, "问题改写失败。", err, o.buildRewriteStageSnapshot(convCtx.Question, historySummary, nil)); err != nil {
			return nil, err
		}
		return nil, err
	}
	if err = o.tracer.CompleteStage(ctx, rewriteStage, "问题改写完成。", o.buildRewriteStageSnapshot(convCtx.Question, historySummary, rewriteResult)); err != nil {
		return nil, err
	}
	// 获取改写后的问题
	rewriteResult.RewrittenQuestion = utils.BlankToDefault(rewriteResult.RewrittenQuestion, strutil.Trim(convCtx.Question))
	if len(rewriteResult.SubQuestions) == 0 {
		rewriteResult.SubQuestions = []string{rewriteResult.RewrittenQuestion}
	}
	return rewriteResult, nil
}

// buildRewriteStageSnapshot 构建改写阶段快照
func (o *PreparationOrchestratorImpl) buildRewriteStageSnapshot(question, historySummary string, rewriteResult *vo.RagRewriteResult) map[string]any {
	snapshot := make(map[string]any)
	snapshot["originalQuestion"] = strutil.Trim(question)
	snapshot["historyContext"] = strutil.Trim(historySummary)

	if rewriteResult != nil {
		snapshot["rewriteQuestion"] = strutil.Trim(rewriteResult.RewrittenQuestion)
		snapshot["subQuestions"] = rewriteResult.SubQuestions
		snapshot["rawModelOutput"] = strutil.Trim(rewriteResult.RawModelOutput)
	}
	snapshot["rewriteOverrideEnabled"] = o.rewriteEnabled
	snapshot["rewriteTemperature"] = o.temperature
	snapshot["rewriteTopP"] = o.topP
	snapshot["rewriteThinking"] = o.thinking
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
func (o *PreparationOrchestratorImpl) selectAutoCandidates(ctx context.Context, routeDecision *klvo.KnowledgeRouteDecision, question, rewriteQuestion string) []*klvo.DocumentRouteCandidate {
	if routeDecision == nil || len(routeDecision.Documents) == 0 {
		return o.fallbackDocuments(ctx, question, rewriteQuestion, 5)
	}

	candidateLimit := utils.Ternary(routeDecision.Confidence >= 0.80, 5, 3)
	var candidates []*klvo.DocumentRouteCandidate
	for _, doc := range routeDecision.Documents {
		if doc.DocumentId > 0 && doc.LastIndexTaskId > 0 {
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
func (o *PreparationOrchestratorImpl) fallbackDocuments(ctx context.Context, question, rewriteQuestion string, limit int) []*klvo.DocumentRouteCandidate {
	docs, err := o.knowledgeLogic.ListRetrievableDocuments(ctx)
	if err != nil {
		Warnf("获取可检索文档失败: %v", err)
		return []*klvo.DocumentRouteCandidate{}
	}
	if len(docs) == 0 {
		return []*klvo.DocumentRouteCandidate{}
	}

	queryTerms := o.extractFallbackTerms(question, rewriteQuestion)

	sort.Slice(docs, func(i, j int) bool {
		return o.fallbackDescriptorScore(docs[j], queryTerms) < o.fallbackDescriptorScore(docs[i], queryTerms)
	})

	result := make([]*klvo.DocumentRouteCandidate, 0, limit)
	for i, desc := range docs {
		if i >= limit {
			break
		}
		score := o.fallbackDescriptorScore(desc, queryTerms)
		result = append(result, &klvo.DocumentRouteCandidate{
			DocumentId:         desc.DocumentId,
			DocumentName:       desc.DocumentName,
			LastIndexTaskId:    desc.LastIndexTaskId,
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
func (o *PreparationOrchestratorImpl) mergeCandidates(primary, secondary []*klvo.DocumentRouteCandidate, limit int) []*klvo.DocumentRouteCandidate {
	merged := make(map[int64]*klvo.DocumentRouteCandidate)
	for _, doc := range primary {
		merged[doc.DocumentId] = doc
	}
	for _, doc := range secondary {
		if _, exists := merged[doc.DocumentId]; !exists {
			merged[doc.DocumentId] = doc
		}
	}

	result := make([]*klvo.DocumentRouteCandidate, 0, limit)
	for _, doc := range merged {
		if len(result) >= limit {
			break
		}
		result = append(result, doc)
	}
	return result
}

// shouldAskClarification 判断是否需要澄清
func (o *PreparationOrchestratorImpl) shouldAskClarification(routeDecision *klvo.KnowledgeRouteDecision, candidateDocuments []*klvo.DocumentRouteCandidate) bool {
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

	if topScore == 0 && secondScore == 0 {
		return false
	}

	return topScore-secondScore <= 3 && topScope != secondScope
}

// buildClarificationReply 构建澄清回复
func (o *PreparationOrchestratorImpl) buildClarificationReply(candidateDocuments []*klvo.DocumentRouteCandidate) string {
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
		name := utils.BlankToDefault(item.DocumentName, strconv.FormatInt(item.DocumentId, 10))
		sb.WriteString(name)
		sb.WriteString("》")

		scope := utils.BlankToDefault(item.KnowledgeScopeName, item.KnowledgeScopeCode)
		if strutil.IsNotBlank(scope) {
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
func (o *PreparationOrchestratorImpl) buildClarificationOptions(candidateDocuments []*klvo.DocumentRouteCandidate) []string {
	if len(candidateDocuments) == 0 {
		return nil
	}

	result := make([]string, 0, 3)
	for _, item := range utils.LimitSlice(candidateDocuments, 3) {
		name := utils.BlankToDefault(item.DocumentName, strconv.FormatInt(item.DocumentId, 10))
		result = append(result, "我想问《"+name+"》")
	}
	return result
}

// buildClarificationReason 构建澄清原因
func (o *PreparationOrchestratorImpl) buildClarificationReason(routeDecision *klvo.KnowledgeRouteDecision, candidateDocuments []*klvo.DocumentRouteCandidate) string {
	if routeDecision == nil || len(routeDecision.Documents) == 0 {
		return "当前自动知识路由没有形成稳定候选，已改为先向用户确认文档范围。"
	}

	return fmt.Sprintf("当前自动知识路由置信度为 %.2f，候选文档数为 %d，为避免误选文档，先返回澄清问题。", routeDecision.Confidence, len(candidateDocuments))
}

// extractFallbackTerms 提取后备检索词
func (o *PreparationOrchestratorImpl) extractFallbackTerms(question, rewriteQuestion string) []string {
	routingText := strutil.Trim(question) + " " + strutil.Trim(rewriteQuestion)
	segments := fallbackSplitRegex.Split(routingText, -1)
	terms := make(map[string]struct{})
	for _, segment := range segments {
		trimmed := strutil.Trim(segment)
		trimmedLen := utf8.RuneCountInString(trimmed)
		if trimmedLen >= 2 {
			terms[trimmed] = struct{}{}
			if trimmedLen >= 4 {
				maxGram := max(6, trimmedLen)
				for gram := 2; gram <= maxGram; gram++ {
					for start := 0; start+gram <= trimmedLen; start++ {
						terms[trimmed[start:start+gram]] = struct{}{}
					}
				}
			}
		}
	}
	return utils.LimitSlice(maputil.Keys(terms), 40)
}

// fallbackDescriptorScore 计算后备文档匹配分数
func (o *PreparationOrchestratorImpl) fallbackDescriptorScore(descriptor *rvo.KnowledgeDocument, queryTerms []string) float64 {
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
		return utf8.RuneCountInString(sortedTerms[i]) > utf8.RuneCountInString(sortedTerms[j])
	})

	matched := make([]string, 0, len(sortedTerms))
	for _, term := range sortedTerms {
		if utf8.RuneCountInString(term) < 2 {
			continue
		}

		if strutil.ContainsAny(term, matched) {
			continue
		}

		if strings.Contains(content, term) {
			matched = append(matched, term)
			switch {
			case utf8.RuneCountInString(term) >= 8:
				score += 12
			case utf8.RuneCountInString(term) >= 5:
				score += 8
			case utf8.RuneCountInString(term) >= 3:
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
	cleaned := fallbackCleanRegex.ReplaceAllString(value, "")
	return strings.ToLower(cleaned)
}

// buildDocumentModeNoEvidenceReply 构建文档模式无证据回复
func (o *PreparationOrchestratorImpl) buildDocumentModeNoEvidenceReply(question string, requiresRealTimeSearch bool) string {
	normalizedQuestion := strutil.Trim(question)

	if o.looksLikeCapabilityQuestion(normalizedQuestion) {
		return "当前你正在使用“当前文档问答”模式，我会优先基于所选文档回答。这个问题更像是在询问助手能力，而不是当前文档内容。如果你想了解我能做什么，请切换到“开放式提问”模式。"
	}

	if o.looksLikeOpenChatQuestion(normalizedQuestion, requiresRealTimeSearch) {
		return "当前你正在使用“当前文档问答”模式，我只能基于所选文档回答。这个问题更像开放式提问，例如天气、最新信息或一般交流。如果你想继续问这类问题，请切换到“开放式提问”模式。"
	}

	return utils.BlankToDefault(o.noEvidenceReply, "当前没有从当前文档中检索到足够证据，暂时不能给出可靠结论。你可以补充更具体的标题、术语或关键词后再试。")
}

// looksLikeCapabilityQuestion 判断是否为能力询问
func (o *PreparationOrchestratorImpl) looksLikeCapabilityQuestion(normalizedQuestion string) bool {
	if normalizedQuestion == "" {
		return false
	}
	return strutil.ContainsAny(normalizedQuestion, capabilityHints)
}

// looksLikeOpenChatQuestion 判断是否为开放式聊天问题
func (o *PreparationOrchestratorImpl) looksLikeOpenChatQuestion(normalizedQuestion string, requiresRealTimeSearch bool) bool {
	if normalizedQuestion == "" {
		return false
	}
	return requiresRealTimeSearch || strutil.ContainsAny(normalizedQuestion, openChatHints) || strutil.ContainsAny(normalizedQuestion, chitchatHints)
}

// extractDocumentIds 提取文档ID列表
func (o *PreparationOrchestratorImpl) extractDocumentIds(candidates []*klvo.DocumentRouteCandidate) []int64 {
	result := make([]int64, 0, len(candidates))
	for _, item := range candidates {
		if item.DocumentId > 0 {
			result = append(result, item.DocumentId)
		}
	}
	return result
}

// extractTaskIds 提取任务ID列表
func (o *PreparationOrchestratorImpl) extractTaskIds(candidates []*klvo.DocumentRouteCandidate) []int64 {
	result := make([]int64, 0, len(candidates))
	for _, item := range candidates {
		if item.LastIndexTaskId > 0 {
			result = append(result, item.LastIndexTaskId)
		}
	}
	return result
}

func Warnf(format string, v ...any) {
	logx.Alert(fmt.Sprintf(format, v...))
}
