package conversation

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/duke-git/lancet/v2/maputil"
	"github.com/duke-git/lancet/v2/strutil"

	"github.com/swiftbit/know-agent/common/logx"
	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/intent"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/enum"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
	doclog "github.com/swiftbit/know-agent/internal/domain/document/logic"
	vo2 "github.com/swiftbit/know-agent/internal/domain/document/model/vo"
	kelog "github.com/swiftbit/know-agent/internal/domain/knowledge/logic/route"
	klvo "github.com/swiftbit/know-agent/internal/domain/knowledge/model/vo"
)

var (
	capabilityHints    = []string{"你都能干什么", "你能做什么", "你可以做什么", "你会什么", "你是谁", "怎么用你", "你能帮我什么"}
	openChatHints      = []string{"天气", "温度", "下雨", "新闻", "股价", "汇率", "热搜", "今天", "明天", "最新", "现在"}
	chitchatHints      = []string{"你好", "您好", "hello", "hi", "谢谢", "感谢", "再见", "拜拜"}
	fallbackCleanRegex = regexp.MustCompile(`[\s>\` + "`" + `*#_\-，,。；;：:（）()“”\"'\\[\\]]+`)
	fallbackSplitRegex = regexp.MustCompile(`[\s、，,；;：:（）()\-的和及与或]+`)
)

// RouteStage 路由判定阶段
// 负责根据 chatMode 分发到对应的路由策略：
//   - OpenChat：直接走 ReactAgent 模式
//   - Document：指定文档问答，在选定的文档内做意图路由
//   - AutoDocument：自动知识路由 + 选文档 + 文档内导航
type RouteStage struct {
	knowledgeRouter kelog.KnowledgeRouter
	documentRouter  intent.DocumentRouter
	lifecycleLogic  doclog.LifecycleLogic
	noEvidenceReply string
}

var _ Stage = (*RouteStage)(nil)

func NewRouteStage(
	knowledgeRouter kelog.KnowledgeRouter,
	documentRouter intent.DocumentRouter,
	lifecycleLogic doclog.LifecycleLogic,
	noEvidenceReply string,
) *RouteStage {
	return &RouteStage{
		knowledgeRouter: knowledgeRouter,
		documentRouter:  documentRouter,
		lifecycleLogic:  lifecycleLogic,
		noEvidenceReply: noEvidenceReply,
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

	// 启动路由追踪阶段（此处以 Rewrite 阶段为名，记录判定结果与时间信号）
	ctx = vo.OnStart(ctx, enum.ConversationTraceStageRewrite, enum.ExecutionModeReactAgent.String(), &vo.StageInput{SummaryText: "路由到开放式 Agent。", Snapshot: nil})
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

// prepareDocumentMode 指定文档问答：用户已在界面选择具体文档。
//
// 步骤：
//  1. 校验所选文档/索引任务 ID 是否有效
//  2. 记录影子路由（便于后续优化自动路由；失败不影响业务流程）
//  3. 调用文档内路由与终稿组装（routeAndFinalizePlan）
func (r *RouteStage) prepareDocumentMode(ctx context.Context, convCtx *Context, execPlan *vo.ConversationExecutionPlan) error {
	// 校验所选文档 ID 与索引任务 ID（必填）
	if convCtx.SelectedDocumentId == 0 || convCtx.SelectedTaskId == 0 {
		return fmt.Errorf("当前文档问答模式缺少有效的文档范围")
	}

	// 记录影子路由（仅用于离线分析，失败只告警）
	if err := r.knowledgeRouter.RecordShadowRoute(ctx, convCtx.ExchangeId, convCtx.SelectedDocumentId, convCtx.ConversationId, convCtx.Question, execPlan.RewriteQuestion); err != nil {
		logx.Warnf("记录影子路由失败: %v", err)
	}

	execPlan.SelectedDocumentId = convCtx.SelectedDocumentId
	execPlan.SelectedDocumentName = convCtx.SelectedDocumentName
	execPlan.SelectedTaskId = convCtx.SelectedTaskId
	if convCtx.SelectedDocumentId > 0 {
		execPlan.RetrievalDocumentIds = []int64{convCtx.SelectedDocumentId}
	}
	if convCtx.SelectedTaskId > 0 {
		execPlan.RetrievalTaskIds = []int64{convCtx.SelectedTaskId}
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
//   - 置信度 ≥ 0.55 且有候选：使用候选首位作为主文档
//   - 否则：不指定主文档，退化为多文档范围混合检索
//   - 需要澄清：返回 Clarification 模式，由用户选择目标知识
func (r *RouteStage) prepareAutoDocumentMode(ctx context.Context, convCtx *Context, execPlan *vo.ConversationExecutionPlan) error {
	// 启动路由阶段追踪（标识为 auto_document）
	ctx = vo.OnStart(ctx, enum.ConversationTraceStageRoute, "auto_document", &vo.StageInput{SummaryText: "正在生成知识范围候选。", Snapshot: nil})

	// 执行知识路由（原始问题 + 改写问题做双路输入）
	//  - 路由失败时仅告警，并以空决策对象兜底（避免后续代码 panic）
	routeDecision, err := r.knowledgeRouter.Route(ctx, convCtx.Question, execPlan.RewriteQuestion)
	if err != nil {
		routeDecision = &klvo.KnowledgeRouteDecision{}
		logx.Warnf("知识路由失败: %v", err)
	}
	// 记录自动路由（用于离线分析；失败只告警）
	if err = r.knowledgeRouter.RecordAutoRoute(ctx, convCtx.ExchangeId, convCtx.ConversationId, convCtx.Question, execPlan.RewriteQuestion, routeDecision); err != nil {
		logx.Warnf("记录自动路由失败: %v", err)
	}

	// 选择候选文档，提取候选的文档 ID 与索引任务 ID 列表（供后续多文档检索使用）
	candidateDocuments := r.selectAutoCandidates(ctx, routeDecision, convCtx.Question, execPlan.RewriteQuestion)
	execPlan.RetrievalDocumentIds = r.extractDocumentIds(candidateDocuments)
	execPlan.RetrievalTaskIds = r.extractTaskIds(candidateDocuments)

	// 选择最高置信度的文档作为主文档
	//  - 阈值 0.55：高于该阈值才信任路由结果的首位
	//  - 不满足条件时 topDocument 保持为空结构，退化为多文档混合检索
	topDocument := &klvo.DocumentRouteCandidate{}
	confidentTop := routeDecision.Confidence >= 0.55
	if confidentTop && len(candidateDocuments) > 0 {
		topDocument = candidateDocuments[0]
	}

	// 提交路由阶段快照（置信度、路由状态、候选数、是否有高置信主文档、主文档信息）
	snapshot := map[string]any{
		"confidence":             routeDecision.Confidence,
		"routeStatus":            routeDecision.RouteStatus,
		"candidateDocumentCount": len(candidateDocuments),
		"confidentTopDocument":   confidentTop,
		"topDocumentId":          topDocument.DocumentId,
		"topDocumentName":        topDocument.DocumentName,
	}
	ctx = vo.OnEnd(ctx, &vo.StageOutput{SummaryText: "知识范围路由完成。", Snapshot: snapshot})

	// 检查是否需要澄清（多个候选相近、路由歧义等）
	//  - 需要澄清时直接返回 Clarification 模式执行计划（含回复文案、选项、理由）
	if r.shouldAskClarification(routeDecision, candidateDocuments) {
		execPlan.Mode = enum.ExecutionModeClarification
		execPlan.ClarificationReply = r.buildClarificationReply(candidateDocuments)
		execPlan.ClarificationOptions = r.buildClarificationOptions(candidateDocuments)
		execPlan.ClarificationReason = r.buildClarificationReason(routeDecision, candidateDocuments)
		return nil
	}

	// 写入主文档信息（若无高置信主文档，则 DocumentId 为 0，退化为多文档混合检索）
	execPlan.SelectedDocumentId = topDocument.DocumentId
	execPlan.SelectedDocumentName = topDocument.DocumentName
	execPlan.SelectedTaskId = topDocument.LastIndexTaskId

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
	navigationDecision, err := r.documentRouter.Route(ctx, execPlan.SelectedDocumentId, convCtx.Question, rewriteResult)
	if err != nil {
		ctx = vo.OnError(ctx, "执行路由失败。", err)
		return err
	}

	// 构造路由结果快照（执行模式 / 章节提示 / 条目编号 / 摘要文本），写入追踪
	snapshot := map[string]any{
		"executionMode":     navigationDecision.ExecutionModeName,
		"targetSectionHint": strutil.Trim(navigationDecision.StructureAnchor.TargetSectionHint),
		"navigationSummary": strutil.Trim(navigationDecision.SummaryText),
	}
	if navigationDecision.ItemAnchor != nil {
		snapshot["targetItemIndex"] = navigationDecision.ItemAnchor.ItemIndex
	}
	ctx = vo.OnEnd(ctx, &vo.StageOutput{SummaryText: "执行路由完成。", Snapshot: snapshot})

	// 从路由结果中选取检索问题与子问题列表
	//  - RetrievalQuestion 优先取自路由计划，为空则回退到改写问题
	//  - RetrievalSubQuestions 仅在路由计划提供时才覆盖；否则保留上层已有值
	if navigationDecision.RetrievalPlan != nil {
		execPlan.RetrievalQuestion = utils.BlankToDefault(navigationDecision.RetrievalPlan.RetrievalQuestion, execPlan.RewriteQuestion)
		if len(navigationDecision.RetrievalPlan.SubQuestions) > 0 {
			execPlan.RetrievalSubQuestions = navigationDecision.RetrievalPlan.SubQuestions
		}
	}

	// 组装最终执行计划：写入执行模式、导航决策、无证据回复提示
	execPlan.Mode = navigationDecision.ExecutionMode
	execPlan.NavigationDecision = navigationDecision
	execPlan.NoEvidenceReply = r.buildDocumentModeNoEvidenceReply(convCtx.Question, execPlan.RequiresRealTimeSearch)

	// 打印关键编排结果（会话ID、模式、原始问题、改写问题、检索问题、执行模式、目标章节）
	logx.Infof("聊天编排完成: conversationId=%s, chatMode=%s, originalQuestion='%s', rewriteQuestion='%s', retrievalQuestion='%s', executionMode=%s, targetSection='%s",
		convCtx.ConversationId, enum.ChatQueryModeName(convCtx.ChatMode), strutil.Trim(convCtx.Question),
		execPlan.RewriteQuestion, execPlan.RetrievalQuestion, execPlan.Mode.Name(), navigationDecision.StructureAnchor.TargetSectionHint)

	return nil
}

// ============================================================================
// 文档模式无证据回复
// ============================================================================

// buildDocumentModeNoEvidenceReply 构建文档模式无证据回复
func (r *RouteStage) buildDocumentModeNoEvidenceReply(question string, requiresRealTimeSearch bool) string {
	normalizedQuestion := strutil.Trim(question)

	if r.looksLikeCapabilityQuestion(normalizedQuestion) {
		return `当前你正在使用"当前文档问答"模式，我会优先基于所选文档回答。这个问题更像是在询问助手能力，而不是当前文档内容。如果你想了解我能做什么，请切换到"开放式提问"模式。`
	}

	if r.looksLikeOpenChatQuestion(normalizedQuestion, requiresRealTimeSearch) {
		return `当前你正在使用"当前文档问答"模式，我只能基于所选文档回答。这个问题更像开放式提问，例如天气、最新信息或一般交流。如果你想继续问这类问题，请切换到"开放式提问"模式。`
	}

	return utils.BlankToDefault(r.noEvidenceReply, "当前没有从当前文档中检索到足够证据，暂时不能给出可靠结论。你可以补充更具体的标题、术语或关键词后再试。")
}

// looksLikeCapabilityQuestion 判断是否为能力询问
func (r *RouteStage) looksLikeCapabilityQuestion(normalizedQuestion string) bool {
	if normalizedQuestion == "" {
		return false
	}
	return strutil.ContainsAny(normalizedQuestion, capabilityHints)
}

// looksLikeOpenChatQuestion 判断是否为开放式聊天问题
func (r *RouteStage) looksLikeOpenChatQuestion(normalizedQuestion string, requiresRealTimeSearch bool) bool {
	if normalizedQuestion == "" {
		return false
	}
	return requiresRealTimeSearch || strutil.ContainsAny(normalizedQuestion, openChatHints) || strutil.ContainsAny(normalizedQuestion, chitchatHints)
}

// ============================================================================
// 自动路由：候选文档选择
// ============================================================================

// selectAutoCandidates 根据路由决策选择自动候选文档。
//
// 策略与分支：
//  1. 路由决策为空或无文档 → 使用 fallbackDocuments 做兜底（上限 5）
//  2. 候选数量阈值：置信度 ≥ 0.80 时取前 5，否则取前 3（高置信度更保守，低置信度多给候选以召回）
//  3. 候选为空时同样回退到 fallbackDocuments
//  4. 置信度 < 0.55 时将路由候选与 fallback 候选合并（扩大范围以弥补低置信度）
//  5. 否则直接返回路由候选
func (r *RouteStage) selectAutoCandidates(ctx context.Context, routeDecision *klvo.KnowledgeRouteDecision, question, rewriteQuestion string) []*klvo.DocumentRouteCandidate {
	// 分支 1：路由决策为空或无文档 → 使用 fallback 做兜底
	if routeDecision == nil || len(routeDecision.Documents) == 0 {
		return r.fallbackDocuments(ctx, question, rewriteQuestion, 5)
	}

	// 候选数量阈值：置信度 ≥ 0.80 时取前 5，否则取前 3
	candidateLimit := utils.Ternary(routeDecision.Confidence >= 0.80, 5, 3)
	var candidates []*klvo.DocumentRouteCandidate
	for _, doc := range routeDecision.Documents {
		// 仅保留具有有效 DocumentId 与 LastIndexTaskId 的候选
		if doc.DocumentId > 0 && doc.LastIndexTaskId > 0 {
			candidates = append(candidates, doc)
			if len(candidates) >= candidateLimit {
				break
			}
		}
	}

	// 预先拉取 fallback 候选（用于分支 3 与 4）
	fallbackDocuments := r.fallbackDocuments(ctx, question, rewriteQuestion, candidateLimit)
	// 分支 3：候选为空 → 返回 fallback
	if len(candidates) == 0 {
		return fallbackDocuments
	}

	// 分支 4：置信度 < 0.55 → 合并路由候选与 fallback 候选，扩大检索范围
	if routeDecision.Confidence < 0.55 {
		return r.mergeCandidates(candidates, fallbackDocuments, candidateLimit)
	}

	// 分支 5：正常情况 → 返回路由候选
	return candidates
}

// fallbackDocuments 获取后备候选文档。
//
// 在路由决策不可用或置信度偏低时，从全部可检索文档中基于元数据（名称/标签等）匹配查询词，
// 返回得分最高的前 limit 个候选，理由统一标注为"低置信度时基于文档元数据进行保守扩范围候选"。
func (r *RouteStage) fallbackDocuments(ctx context.Context, question, rewriteQuestion string, limit int) []*klvo.DocumentRouteCandidate {
	// 拉取全部可检索文档；失败或为空时返回 nil（上游可继续用主文档或混合检索兜底）
	docs, err := r.lifecycleLogic.ListRetrievableDocuments(ctx)
	if err != nil {
		logx.Warnf("获取可检索文档失败: %v", err)
		return nil
	}
	if len(docs) == 0 {
		return nil
	}

	// 从问题与改写问题中抽取 fallback 查询词（用于元数据匹配打分）
	queryTerms := r.extractFallbackTerms(question, rewriteQuestion)

	// 按文档分别计算 fallback 得分（基于名称/标签与查询词的匹配）
	scoreMap := make(map[int64]float64, len(docs))
	for _, desc := range docs {
		scoreMap[desc.DocumentId] = r.fallbackDescriptorScore(desc, queryTerms)
	}

	// 按得分降序排序
	sort.Slice(docs, func(i, j int) bool {
		return scoreMap[docs[i].DocumentId] > scoreMap[docs[j].DocumentId]
	})

	// 取前 limit 个候选，组装为 DocumentRouteCandidate（统一 Reason 标注）
	result := make([]*klvo.DocumentRouteCandidate, 0, limit)
	for i, desc := range docs {
		if i >= limit {
			break
		}
		result = append(result, &klvo.DocumentRouteCandidate{
			DocumentId:      desc.DocumentId,
			DocumentName:    desc.DocumentName,
			LastIndexTaskId: desc.LastIndexTaskId,
			Score:           scoreMap[desc.DocumentId],
			Reason:          "低置信度时基于文档元数据进行保守扩范围候选",
		})
	}

	return result
}

// mergeCandidates 合并主候选与次候选并去重（以 DocumentId 为键），最终数量不超过 limit。
// 去重策略：主候选优先（先遍历 primary，其条目被保留），secondary 仅在未出现时被加入。
func (r *RouteStage) mergeCandidates(primary, secondary []*klvo.DocumentRouteCandidate, limit int) []*klvo.DocumentRouteCandidate {
	// 使用 map 做 DocumentId 维度的去重；primary 先遍历以保证优先级
	merged := make(map[int64]*klvo.DocumentRouteCandidate)
	ids := make([]int64, 0, len(primary)+len(secondary))
	for _, doc := range primary {
		merged[doc.DocumentId] = doc
		ids = append(ids, doc.DocumentId)
	}
	// secondary 仅在 DocumentId 未出现时被加入
	for _, doc := range secondary {
		if _, exists := merged[doc.DocumentId]; !exists {
			merged[doc.DocumentId] = doc
			ids = append(ids, doc.DocumentId)
		}
	}

	// 将去重后的候选按插入顺序收集为结果
	result := make([]*klvo.DocumentRouteCandidate, 0, limit)
	for _, id := range ids {
		if len(result) >= limit {
			break
		}
		result = append(result, merged[id])
	}
	return result
}

// ============================================================================
// 自动路由：澄清判断
// ============================================================================

// shouldAskClarification 判断是否需要向用户澄清知识范围
//
// 判定逻辑（任一成立则需要澄清）：
//  1. 候选文档为空 —— 无任何可检索范围，需要用户补充
//  2. 路由决策为空或无文档 —— 路由失败，可能因问题宽泛或模型响应异常
//  3. 路由决策置信度 < 0.55 —— 低置信度，需要用户在多个可能方向中选择
//  4. 候选数量 < 2 —— 无法进行多方向对比，跳过澄清（返回 false）
//  5. 前两名候选得分差 ≤ 3 且分属不同知识范围（KnowledgeScopeCode 不同）—— 存在真正的歧义
//
// 特别例外：前两名候选得分均为 0（说明打分完全失败）时不做澄清，以避免无意义的空选项提示。
func (r *RouteStage) shouldAskClarification(routeDecision *klvo.KnowledgeRouteDecision, candidateDocuments []*klvo.DocumentRouteCandidate) bool {
	// 判定 1：候选为空 —— 需要澄清
	if len(candidateDocuments) == 0 {
		return true
	}
	// 判定 2：路由决策为空或无文档 —— 需要澄清
	if routeDecision == nil || len(routeDecision.Documents) == 0 {
		return true
	}
	// 判定 3：低置信度（< 0.55）—— 需要澄清
	if routeDecision.Confidence < 0.55 {
		return true
	}
	// 判定 4：候选数量不足 2 —— 不足以形成多选项对比，跳过
	if len(candidateDocuments) < 2 {
		return false
	}

	// 取前两名候选的得分与知识范围
	topScore := candidateDocuments[0].Score
	secondScore := candidateDocuments[1].Score
	topScope := candidateDocuments[0].KnowledgeScopeCode
	secondScope := candidateDocuments[1].KnowledgeScopeCode

	// 特别例外：打分完全失败时不发起澄清，避免给出无意义的多选项提示
	if topScore == 0 && secondScore == 0 {
		return false
	}

	// 判定 5：得分差 ≤ 3 且分属不同知识范围 → 存在真正的歧义，需要澄清
	return topScore-secondScore <= 3 && topScope != secondScope
}

// buildClarificationReply 构建澄清回复
func (r *RouteStage) buildClarificationReply(candidateDocuments []*klvo.DocumentRouteCandidate) string {
	topCandidates := candidateDocuments
	if len(topCandidates) > 3 {
		topCandidates = topCandidates[:3]
	}

	if len(topCandidates) == 0 {
		return `当前我还不能稳定判断你想问哪份知识文档。请补充更具体的文档名、主题词，或者直接切换到"当前文档问答"后指定文档。`
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

	sb.WriteString(`你可以直接回复文档名，或者改用"当前文档问答"模式明确指定文档。`)
	return sb.String()
}

// buildClarificationOptions 构建澄清选项
func (r *RouteStage) buildClarificationOptions(candidateDocuments []*klvo.DocumentRouteCandidate) []string {
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
func (r *RouteStage) buildClarificationReason(routeDecision *klvo.KnowledgeRouteDecision, candidateDocuments []*klvo.DocumentRouteCandidate) string {
	if routeDecision == nil || len(routeDecision.Documents) == 0 {
		return "当前自动知识路由没有形成稳定候选，已改为先向用户确认文档范围。"
	}

	return fmt.Sprintf("当前自动知识路由置信度为 %.2f，候选文档数为 %d，为避免误选文档，先返回澄清问题。", routeDecision.Confidence, len(candidateDocuments))
}

// ============================================================================
// 自动路由：辅助方法
// ============================================================================

// extractFallbackTerms 提取后备检索词
func (r *RouteStage) extractFallbackTerms(question, rewriteQuestion string) []string {
	routingText := strutil.Trim(question) + " " + strutil.Trim(rewriteQuestion)
	segments := fallbackSplitRegex.Split(routingText, -1)
	terms := make(map[string]struct{})
	for _, segment := range segments {
		trimmed := strutil.Trim(segment)
		trimmedLen := utils.Len(trimmed)
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
func (r *RouteStage) fallbackDescriptorScore(descriptor *vo2.KnowledgeDocument, queryTerms []string) float64 {
	content := strings.Join([]string{
		descriptor.DocumentName,
	}, " ")

	content = r.normalizeFallbackText(content)

	if len(queryTerms) == 0 || content == "" {
		return 0
	}

	var score float64
	sortedTerms := make([]string, 0, len(queryTerms))
	for _, term := range queryTerms {
		normalized := r.normalizeFallbackText(term)
		if normalized != "" {
			sortedTerms = append(sortedTerms, normalized)
		}
	}

	sort.Slice(sortedTerms, func(i, j int) bool {
		return utils.Len(sortedTerms[i]) > utils.Len(sortedTerms[j])
	})

	matched := make([]string, 0, len(sortedTerms))
	for _, term := range sortedTerms {
		if utils.Len(term) < 2 || strutil.ContainsAny(term, matched) {
			continue
		}

		if strings.Contains(content, term) {
			matched = append(matched, term)
			switch {
			case utils.Len(term) >= 8:
				score += 12
			case utils.Len(term) >= 5:
				score += 8
			case utils.Len(term) >= 3:
				score += 4
			default:
				score += 2
			}
		}
	}

	return score
}

// normalizeFallbackText 标准化后备文本
func (r *RouteStage) normalizeFallbackText(value string) string {
	if value == "" {
		return ""
	}
	cleaned := fallbackCleanRegex.ReplaceAllString(value, "")
	return strings.ToLower(cleaned)
}

// extractDocumentIds 提取文档ID列表
func (r *RouteStage) extractDocumentIds(candidates []*klvo.DocumentRouteCandidate) []int64 {
	result := make([]int64, 0, len(candidates))
	for _, item := range candidates {
		if item.DocumentId > 0 {
			result = append(result, item.DocumentId)
		}
	}
	return result
}

// extractTaskIds 提取任务ID列表
func (r *RouteStage) extractTaskIds(candidates []*klvo.DocumentRouteCandidate) []int64 {
	result := make([]int64, 0, len(candidates))
	for _, item := range candidates {
		if item.LastIndexTaskId > 0 {
			result = append(result, item.LastIndexTaskId)
		}
	}
	return result
}
