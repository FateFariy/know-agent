package conversation

import (
	"context"
	"fmt"

	"github.com/swiftbit/know-agent/common/logx"
	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/chat/adapter"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/enum"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
	"github.com/swiftbit/know-agent/internal/svc"
)

const (
	recommendationThreshold = 0.55
)

// RouteStage 路由判定阶段
// 负责根据 chatMode 分发到对应的路由策略：
//   - OpenChat：直接走 ReactAgent 模式
//   - Document：指定文档问答，在选定的文档内做意图路由
//   - AutoDocument：自动知识路由 + 选文档 + 文档内导航
type RouteStage struct {
	repo            adapter.ChatRepository
	knowledgeRouter KnowledgeRouter
	documentRouter  DocumentRouter
	docGateway      adapter.DocumentGateway
	noEvidenceReply string
}

var _ Stage = (*RouteStage)(nil)

func NewRouteStage(
	svcCtx *svc.ServiceContext,
	repo adapter.ChatRepository,
	knowledgeRouter KnowledgeRouter,
	documentRouter DocumentRouter,
	docGateway adapter.DocumentGateway,
) *RouteStage {
	return &RouteStage{
		repo:            repo,
		knowledgeRouter: knowledgeRouter,
		documentRouter:  documentRouter,
		docGateway:      docGateway,
		noEvidenceReply: svcCtx.Config.Chat.Rag.NoEvidenceReply,
	}
}

// Name 阶段名称
func (r *RouteStage) Name() string {
	return enum.ConversationTraceStageRoute.Name
}

// Execute 执行路由判定
//
// 根据 chatMode 分发到对应的路由策略：
//  1. OpenChat → prepareOpenChat（开放式 Agent）
//  2. Document → prepareDocumentMode（指定文档问答）
//  3. AutoDocument → prepareAutoDocumentMode（自动知识路由 + 文档内导航）
func (r *RouteStage) Execute(ctx context.Context, convCtx *Context) error {
	execPlan := convCtx.ExecutionPlan.Load()
	if execPlan == nil {
		return nil
	}

	// 语义缓存命中：检索链路结果已由缓存提供，跳过路由
	if convCtx.cache.IsCacheHit() {
		return nil
	}

	var err error
	switch convCtx.ChatMode {
	case enum.ChatQueryModeOpenChat:
		// 开放式聊天：直接走 ReactAgent
		err = r.prepareOpenChat(ctx, convCtx, execPlan)
	case enum.ChatQueryModeDocument:
		// 指定文档问答：路由到所选文档内做导航
		err = r.prepareDocumentMode(ctx, convCtx, execPlan)
	case enum.ChatQueryModeAutoDocument:
		// 自动文档问答：先做知识路由选文档，再在文档内导航
		err = r.prepareAutoDocumentMode(ctx, convCtx, execPlan)
	default:
		return fmt.Errorf("不支持的聊天模式: %s", enum.ChatQueryModeName(convCtx.ChatMode))
	}
	if err != nil {
		return err
	}

	convCtx.SetExecutePlan(execPlan)
	return nil
}

// ============================================================================
// 路由策略：OpenChat
// ============================================================================

// prepareOpenChat 开放式聊天：直接走 ReactAgent 模式，不做文档路由与检索准备。
//
// 步骤：
//  1. 设置执行模式为 ExecutionModeReactAgent
//  2. 启动并完成路由追踪阶段，写入快照（chatMode / executionMode / 时间信号）
func (r *RouteStage) prepareOpenChat(ctx context.Context, convCtx *Context, execPlan *vo.ConversationExecutionPlan) error {
	// 设置执行模式为 ReactAgent（完全由下游 Agent 自主规划）
	execPlan.Mode = enum.ExecutionModeReactAgent

	// 启动路由追踪阶段
	ctx = vo.OnStart(ctx, enum.ConversationTraceStageRoute, enum.ExecutionModeReactAgent.String(), &vo.StageInput{SummaryText: "路由到开放式 Agent。", Snapshot: nil})
	snapshot := map[string]any{
		"chatMode":                     enum.ChatQueryModeName(convCtx.ChatMode),
		"executionMode":                enum.ExecutionModeReactAgent.String(),
		"requiresRealTimeSearch":       execPlan.RequiresRealTimeSearch,
		"requiresCurrentDateAnchoring": execPlan.RequiresCurrentDateAnchoring,
	}
	ctx = vo.OnEnd(ctx, &vo.StageOutput{SummaryText: "已判定走开放式 Agent 路径。", Snapshot: snapshot})

	return nil
}

// ============================================================================
// 路由策略：DocumentMode（指定文档问答）
// ============================================================================

// prepareDocumentMode 指定文档问答：用户已在界面选择具体文档
//
// 步骤：
//  1. 解析知识库选择快照，确定允许的执行范围
//  2. 校验所选文档/索引任务 ID 是否在允许范围内
//  3. 记录影子路由（便于后续优化自动路由；失败不影响业务流程）
//  4. 调用文档内路由与终稿组装（routeAndFinalizePlan）
func (r *RouteStage) prepareDocumentMode(ctx context.Context, convCtx *Context, execPlan *vo.ConversationExecutionPlan) error {
	// 解析知识库选择快照，确定允许执行的知识范围
	allowedScope := convCtx.KnowledgeBaseSelectionSnapshot.ResolveAllowedExecutionScope()

	// 校验所选文档/任务是否在允许范围内
	// 先执行基础校验（非零），再执行范围一致性校验
	if convCtx.SelectedDocumentId == 0 || convCtx.SelectedTaskId == 0 {
		return fmt.Errorf("当前文档问答模式缺少有效的文档范围")
	}
	if !allowedScope.Consistent || !allowedScope.Contains(convCtx.SelectedDocumentId, convCtx.SelectedTaskId) {
		return fmt.Errorf("所选文档/任务不在当前允许的知识范围之内")
	}

	// 记录影子路由（仅用于离线分析，失败只告警）
	if err := r.knowledgeRouter.RecordShadowRoute(ctx, NewKnowledgeRouteInput(convCtx, execPlan.RewriteQuestion)); err != nil {
		logx.Warnf("记录影子路由失败: %v", err)
	}

	// 在选定的文档内做路由，并组装最终执行计划
	return r.routeAndFinalizePlan(ctx, convCtx, execPlan)
}

// ============================================================================
// 路由策略：AutoDocumentMode（自动文档问答）
// ============================================================================

// prepareAutoDocumentMode 自动文档问答：执行知识路由 → 必要时澄清 → 确定主文档 → 文档内导航 → 生成计划。
//
// 关键分支：
//   - 路由失败：记录告警，使用空路由决策继续执行
//   - 推荐选择成功且有候选：使用推荐候选作为主文档
//   - 否则：不指定主文档，退化为多文档范围混合检索
//   - 需要澄清：返回 Clarification 模式，由用户选择目标知识
func (r *RouteStage) prepareAutoDocumentMode(ctx context.Context, convCtx *Context, execPlan *vo.ConversationExecutionPlan) error {
	// 解析知识库选择快照，确定允许执行的知识范围
	allowedScope := convCtx.KnowledgeBaseSelectionSnapshot.ResolveAllowedExecutionScope()

	// 启动路由阶段追踪（标识为 auto_document）
	ctx = vo.OnStart(ctx, enum.ConversationTraceStageRoute, "auto_document", &vo.StageInput{SummaryText: "正在执行知识范围、主题、候选文档路由。", Snapshot: nil})

	// 执行知识路由（原始问题 + 改写问题做双路输入）
	routeDecision, err := r.knowledgeRouter.Route(ctx, NewKnowledgeRouteInput(convCtx, execPlan.RewriteQuestion))
	if err != nil {
		routeDecision = vo.NewUnavailableRouteDecision("ROUTE_ADVISOR_FAILURE")
		logx.Warnf("知识路由失败: %v", err)
	}

	// 选择候选文档（基于路由决策 + 允许范围过滤），提取候选 ID 列表
	inScopeCandidates := allowedScope.FilterCandidates(routeDecision.Documents)

	// 选择推荐候选作为主文档，使用 SelectRecommendedCandidate 综合判断：置信度阈值、候选有效性、原始top匹配
	topDocument := routeDecision.SelectRecommendedCandidate(inScopeCandidates, recommendationThreshold)
	confidentTop := topDocument != nil && topDocument.DocumentId > 0

	// 构建增强快照（含范围信息、路由状态、置信度、候选数、推荐结果）
	snapshot := map[string]any{
		"confidence":             routeDecision.Confidence,
		"routeStatus":            routeDecision.RouteStatus,
		"routeSource":            routeDecision.Source,
		"degraded":               routeDecision.Degraded,
		"candidateDocumentCount": len(inScopeCandidates),
		"allowedDocumentCount":   len(allowedScope.DocumentIds()),
		"allowedDocumentIds":     allowedScope.DocumentIds(),
		"allowedTaskIds":         allowedScope.TaskIds(),
		"scopeConsistent":        allowedScope.Consistent,
		"scopeReason":            allowedScope.Reason,
	}
	if confidentTop {
		snapshot["topDocumentId"] = topDocument.DocumentId
		snapshot["topDocumentTaskId"] = topDocument.LastIndexTaskId
		snapshot["topDocumentScore"] = topDocument.Score
		snapshot["topDocumentName"] = topDocument.DocumentName
	}
	ctx = vo.OnEnd(ctx, &vo.StageOutput{SummaryText: "知识范围路由完成。", Snapshot: snapshot})

	if !allowedScope.Executable() {
		execPlan.Mode = enum.ExecutionModeClarification
		execPlan.ClarificationReply = "当前选择的知识范围没有可检索的已就绪文档，请重新选择知识库或等待文档完成索引。"
		execPlan.ClarificationReason = allowedScope.Reason
		return nil
	}

	// 在选定的文档内做路由，并组装最终执行计划
	if err = r.routeAndFinalizePlan(ctx, convCtx, execPlan); err != nil {
		return err
	}
	return nil
}

// ============================================================================
// 共享：文档内路由与终稿组装
// ============================================================================

// routeAndFinalizePlan 在文档内完成意图路由与执行计划终稿组装。
//
// 总体流程：
//  1. 启动路由追踪阶段 → 调用 documentRouter 做文档内意图判定
//  2. 路由失败时记录失败并向上返回
//  3. 路由成功后将执行模式/章节锚点/条目锚点写入快照，提交追踪
//  4. 从路由结果中选取检索问题与子问题列表（空值回退到改写问题）
//  5. 组装最终执行计划（执行模式 / 导航决策 / 无证据回复提示）
//  6. 打印关键编排结果并返回
func (r *RouteStage) routeAndFinalizePlan(ctx context.Context, convCtx *Context, execPlan *vo.ConversationExecutionPlan) error {
	// 启动文档内路由阶段追踪，并以 "混合检索" 为默认模式名
	ctx = vo.OnStart(ctx, enum.ConversationTraceStageRoute, enum.ExecutionModeRetrieval.Name(), &vo.StageInput{SummaryText: "正在判定图查询还是混合检索。", Snapshot: nil})

	// 构造改写结果对象，调用 Router 做文档内意图路由（输出执行模式、章节锚点等）
	rewriteResult := vo.NewQuestionRewriteResult(execPlan.RewriteQuestion, execPlan.RewriteSubQuestions)
	input := &DocumentRouteInput{
		DocumentId:        convCtx.SelectedDocumentId,
		OriginalQuestion:  convCtx.Question,
		RewriteResult:     rewriteResult,
		RecognitionResult: execPlan.RecognitionResult,
	}
	navigationDecision, err := r.documentRouter.Route(ctx, input)
	if err != nil {
		ctx = vo.OnError(ctx, "执行路由失败。", err)
		return err
	}

	// 构造路由结果快照（执行模式 / 章节提示 / 条目编号 / 摘要文本），写入追踪
	snapshot := map[string]any{
		"executionMode":     navigationDecision.ExecutionModeName,
		"targetSectionHint": utils.Trim(navigationDecision.StructureAnchor.TargetSectionHint),
		"navigationSummary": utils.Trim(navigationDecision.SummaryText),
	}
	if navigationDecision.ItemAnchor != nil {
		snapshot["targetItemIndex"] = navigationDecision.ItemAnchor.ItemIndex
	}

	anchors, err := r.loadRecentEvidenceAnchors(ctx, convCtx.ConversationId, maxRecentExchanges)
	if err != nil {
		ctx = vo.OnError(ctx, "加载最近证据锚点失败。", err)
		return err
	}
	if convCtx.ChatMode == enum.ChatQueryModeDocument {
		anchors = r.filterValidEvidenceAnchors(anchors, convCtx.SelectedDocumentId)
	} else {
		anchors = r.filterValidEvidenceAnchors(anchors, convCtx.KnowledgeBaseSelectionSnapshot.SelectedDocumentIds()...)
	}
	execPlan.QuestionHistoryContext.ApplyFollowUpAndEvidence(convCtx.Question, execPlan.RecognitionResult, anchors)
	execPlan.RecentEvidenceAnchors = anchors

	// 组装最终执行计划：写入执行模式、导航决策、无证据回复提示、检索计划
	execPlan.Mode = navigationDecision.ExecutionMode
	execPlan.NavigationDecision = navigationDecision
	execPlan.ApplyNoEvidenceReply()
	execPlan.RetrievalPlan = r.buildRetrievalPlan(convCtx, execPlan)

	// 打印关键编排结果（会话ID、模式、原始问题、改写问题、检索问题、执行模式、目标章节）
	logx.Infof("聊天编排完成: conversationId=%s, chatMode=%s, originalQuestion='%s', rewriteQuestion='%s', executionMode=%s, targetSection='%s",
		convCtx.ConversationId, enum.ChatQueryModeName(convCtx.ChatMode), utils.Trim(convCtx.Question),
		execPlan.RewriteQuestion, execPlan.Mode.Name(), navigationDecision.StructureAnchor.TargetSectionHint)

	ctx = vo.OnEnd(ctx, &vo.StageOutput{SummaryText: "执行路由完成。", Snapshot: snapshot})

	return nil
}

// loadRecentEvidenceAnchors 加载最近的证据锚点，从对话历史中抽取追问可继承的结构锚点
func (r *RouteStage) loadRecentEvidenceAnchors(ctx context.Context, conversationId string, limit int) (vo.EvidenceAnchors, error) {
	if conversationId == "" || limit <= 0 {
		return nil, nil
	}

	exchanges, err := r.repo.ListRecentExchanges(ctx, conversationId, maxRecentExchanges)
	if err != nil || len(exchanges) == 0 {
		return nil, err
	}

	var anchors vo.EvidenceAnchors
	for _, exchange := range exchanges {
		if exchange == nil || !exchange.IsCompleted() || len(exchange.References) == 0 {
			continue
		}
		for _, ref := range exchange.References {
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

func (r *RouteStage) filterValidEvidenceAnchors(anchors vo.EvidenceAnchors, allDocumentIds ...int64) vo.EvidenceAnchors {
	if len(anchors) == 0 || len(allDocumentIds) == 0 {
		return anchors
	}

	return utils.Filter(anchors, func(anchor *vo.EvidenceAnchor) bool {
		return anchor != nil && (utils.ContainsAny(allDocumentIds, anchor.DocumentId))
	})
}

// buildRetrievalPlan
func (r *RouteStage) buildRetrievalPlan(convCtx *Context, execPlan *vo.ConversationExecutionPlan) *vo.RetrievalPlan {
	snapshot := convCtx.KnowledgeBaseSelectionSnapshot
	runtime := snapshot.RagRuntimeOptions
	if runtime == nil {
		runtime = vo.NewDefaultRagRuntimeOptions()
	}
	questionPlan := r.newQuestionPlan(execPlan)
	intentResult := execPlan.RecognitionResult
	hybrid := runtime.Hybrid
	if hybrid == nil {
		hybrid = vo.NewDefaultHybridOptions()
	}

	channels := r.newChannelPlans(runtime, hybrid)
	chatMode := execPlan.ChatMode
	documentScope := snapshot.SelectedDocumentIds()
	taskScope := snapshot.SelectedTaskIds()
	if chatMode == enum.ChatQueryModeDocument {
		documentScope = []int64{convCtx.SelectedDocumentId}
		taskScope = []int64{convCtx.SelectedTaskId}
	}
	plan := &vo.RetrievalPlan{
		QuestionPlan:              questionPlan,
		ChatMode:                  enum.ChatQueryModeName(execPlan.ChatMode),
		PrimaryIntent:             execPlan.NavigationDecision.PrimaryIntent(),
		SuggestedIntents:          intentResult.SuggestedChannels(),
		ScopeMode:                 snapshot.SelectionModeName(),
		KnowledgeBaseIds:          utils.Copy(snapshot.SelectedKnowledgeBaseIds),
		AllowedDocumentScope:      utils.Copy(snapshot.SelectedDocumentIds()),
		DocumentScope:             utils.Copy(documentScope),
		TaskScope:                 utils.Copy(taskScope),
		MetadataFilters:           vo.NewMetadataFilters(questionPlan.RetrievalQuestion, intentResult),
		Channels:                  channels,
		StructureNavigation:       intentResult.StructureNavigationIntent.Clone(),
		NavigationAction:          execPlan.NavigationDecision.NavigationActionText(),
		StructureNavigationResult: r.copyStructureNavigationResult(execPlan.NavigationDecision),
		StructureAnchor:           r.copyStructureAnchor(execPlan.NavigationDecision),
		ItemAnchor:                r.copyItemAnchor(execPlan.NavigationDecision),
		TableIntent:               intentResult.ToTableIntent(),
		GraphIntent:               intentResult.ToGraphIntent(runtime.GraphRagMaxHops),
		RaptorIntent:              intentResult.ToRaptorIntent(runtime.RaptorSourceChunkTopK),
		RankFeatures:              vo.BuildRankFeatures(hybrid),
		CandidateTopK:             runtime.CandidateTopK,
		RerankTopK:                runtime.RerankCandidateTopK,
		RerankEnabled:             runtime.RerankEnabled,
		FinalTopK:                 runtime.FinalTopK,
		SubQuestionTimeout:        runtime.SubQuestionTimeout,
	}

	return plan
}

// newQuestionPlan 构建检索问题计划
func (r *RouteStage) newQuestionPlan(exec *vo.ConversationExecutionPlan) *vo.RetrievalQuestionPlan {
	currentQuestion := utils.CompactWhitespace(exec.OriginalQuestion)
	rewrittenQuestion := utils.CompactWhitespace(exec.RewriteQuestion)
	normalizedQuery := utils.BlankToDefault(rewrittenQuestion, currentQuestion)

	var inheritedAnchors []*vo.RetrievalContextAnchor
	if exec.RecognitionResult != nil && exec.RecognitionResult.QueryType == enum.QueryTypeFollowUp {
		keyOf := func(anchor *vo.EvidenceAnchor) (string, *vo.RetrievalContextAnchor, bool) {
			if inherited := anchor.ToRetrievalContextAnchor(); inherited != nil {
				return inherited.UniqueKey(), inherited, true
			}
			return "", nil, false
		}
		inheritedAnchors = utils.FilterMapUniqueLimit(exec.RecentEvidenceAnchors, 5, keyOf)
	}
	contextHints := make([]string, 0, len(inheritedAnchors))
	for _, anchor := range inheritedAnchors {
		contextHints = append(contextHints, anchor.AnchorHint())
	}

	of := func(sq string) (string, string, bool) {
		sq = utils.CompactWhitespace(sq)
		return sq, sq, sq != ""
	}
	subQuestions := utils.FilterMapUniqueLimit(exec.RewriteSubQuestions, 5, of)
	if len(subQuestions) == 0 && utils.IsNotBlank(normalizedQuery) {
		subQuestions = append(subQuestions, normalizedQuery)
	}

	executionQueries := make([]*vo.RetrievalExecutionQuery, 0, len(subQuestions))
	for i, sq := range subQuestions {
		executionQueries = append(executionQueries, &vo.RetrievalExecutionQuery{
			Index:        i + 1,
			SubQuestion:  sq,
			ContextHints: append([]string{}, contextHints...),
		})
	}

	return &vo.RetrievalQuestionPlan{
		CurrentQuestion:          currentQuestion,
		RewrittenQuestion:        rewrittenQuestion,
		RetrievalQuestion:        normalizedQuery,
		ExecutionQueries:         executionQueries,
		FollowUp:                 len(inheritedAnchors) > 0,
		HistoryInherited:         len(inheritedAnchors) > 0,
		HistoryInheritanceSource: utils.Ternary(len(inheritedAnchors) > 0, "FINAL_EVIDENCE_ANCHOR", "NONE"),
		InheritedContextAnchors:  inheritedAnchors,
		SubQuestions:             subQuestions,
	}
}

// newChannelPlans 构建检索通道计划列表
func (r *RouteStage) newChannelPlans(runtime *vo.RagRuntimeOptions, hybrid *vo.HybridOptions) []*vo.RetrievalChannelPlan {
	return []*vo.RetrievalChannelPlan{
		vo.NewVectorChannelPlan(true, runtime.VectorTopK, runtime.ChannelTimeout, hybrid.VectorWeight, runtime.MinVectorSimilarity),
		vo.NewKeywordChannelPlan(runtime.KeywordChannelEnabled, runtime.KeywordTopK, runtime.ChannelTimeout, hybrid.KeywordWeight, runtime.KeywordRelativeScoreFloor),
		vo.NewTableChannelPlan(runtime.TableChannelEnabled, runtime.CandidateTopK, runtime.ChannelTimeout, hybrid.TableWeight),
		vo.NewGraphRAGChannelPlan(runtime.GraphRagChannelEnabled, runtime.GraphRagTopK, runtime.ChannelTimeout, hybrid.GraphRagWeight),
		vo.NewRaptorChannelPlan(runtime.RaptorChannelEnabled, runtime.RaptorTopK, runtime.ChannelTimeout, hybrid.RaptorWeight),
	}
}

// copyStructureNavigationResult 复制结构导航结果 todo 待实现
func (r *RouteStage) copyStructureNavigationResult(decision *vo.DocumentNavigationDecision) *vo.StructureNavigationResult {
	if decision == nil {
		return nil
	}
	return nil
}

// copyStructureAnchor 复制结构锚点
func (r *RouteStage) copyStructureAnchor(decision *vo.DocumentNavigationDecision) *vo.ConversationStructureAnchor {
	if decision == nil || decision.StructureAnchor == nil {
		return nil
	}
	anchor := *decision.StructureAnchor
	return &anchor
}

// copyItemAnchor 复制条目锚点
func (r *RouteStage) copyItemAnchor(decision *vo.DocumentNavigationDecision) *vo.ConversationItemAnchor {
	if decision == nil || decision.ItemAnchor == nil {
		return nil
	}
	item := *decision.ItemAnchor
	return &item
}
