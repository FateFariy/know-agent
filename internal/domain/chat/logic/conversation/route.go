package conversation

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/duke-git/lancet/v2/maputil"
	"github.com/duke-git/lancet/v2/strutil"

	"github.com/swiftbit/know-agent/common/logx"
	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/chat/adapter"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/enum"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
)

var (
	capabilityHints    = []string{"你都能干什么", "你能做什么", "你可以做什么", "你会什么", "你是谁", "怎么用你", "你能帮我什么"}
	openChatHints      = []string{"天气", "温度", "下雨", "新闻", "股价", "汇率", "热搜", "今天", "明天", "最新", "现在"}
	chitchatHints      = []string{"你好", "您好", "hello", "hi", "谢谢", "感谢", "再见", "拜拜"}
	fallbackCleanRegex = regexp.MustCompile(`[\s>\` + "`" + `*#_\-，,。；;：:（）()“”\"'\\[\\]]+`)
	fallbackSplitRegex = regexp.MustCompile(`[\s、，,；;：:（）()\-的和及与或]+`)
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
	knowledgeRouter KnowledgeRouter
	documentRouter  DocumentRouter
	docGateway      adapter.DocumentGateway
	noEvidenceReply string
}

var _ Stage = (*RouteStage)(nil)

func NewRouteStage(
	knowledgeRouter KnowledgeRouter,
	documentRouter DocumentRouter,
	docGateway adapter.DocumentGateway,
	noEvidenceReply string,
) *RouteStage {
	return &RouteStage{
		knowledgeRouter: knowledgeRouter,
		documentRouter:  documentRouter,
		docGateway:      docGateway,
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
	if err := r.knowledgeRouter.RecordShadowRoute(ctx, NewRouteInput(convCtx, execPlan.RewriteQuestion)); err != nil {
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
	routeDecision, err := r.knowledgeRouter.Route(ctx, NewRouteInput(convCtx, execPlan.RewriteQuestion))
	if err != nil {
		routeDecision = vo.NewUnavailableRouteDecision("ROUTE_ADVISOR_FAILURE")
		logx.Warnf("知识路由失败: %v", err)
	}

	// 选择候选文档（基于路由决策 + 允许范围过滤），提取候选 ID 列表
	inScopeCandidates := r.selectAutoCandidates(routeDecision, allowedScope)

	// 选择推荐候选作为主文档，使用 selectRecommendation 综合判断：置信度阈值、候选有效性、原始top匹配
	topDocument := r.selectRecommendation(routeDecision, inScopeCandidates)
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
	navigationDecision, err := r.documentRouter.Route(ctx, convCtx.SelectedDocumentId, convCtx.Question, rewriteResult)
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

	// 组装最终执行计划：写入执行模式、导航决策、无证据回复提示
	execPlan.Mode = navigationDecision.ExecutionMode
	execPlan.NavigationDecision = navigationDecision
	execPlan.NoEvidenceReply = execPlan.RequiresRealTimeSearch

	// 打印关键编排结果（会话ID、模式、原始问题、改写问题、检索问题、执行模式、目标章节）
	logx.Infof("聊天编排完成: conversationId=%s, chatMode=%s, originalQuestion='%s', rewriteQuestion='%s', executionMode=%s, targetSection='%s",
		convCtx.ConversationId, enum.ChatQueryModeName(convCtx.ChatMode), strutil.Trim(convCtx.Question),
		execPlan.RewriteQuestion, execPlan.Mode.Name(), navigationDecision.StructureAnchor.TargetSectionHint)

	ctx = vo.OnEnd(ctx, &vo.StageOutput{SummaryText: "执行路由完成。", Snapshot: snapshot})

	return nil
}

// ============================================================================
// 自动路由：候选文档选择
// ============================================================================

// selectAutoCandidates 根据路由决策和允许范围选择自动候选文档
func (r *RouteStage) selectAutoCandidates(routeDecision *vo.KnowledgeRouteDecision, allowedScope *vo.AllowedExecutionScope) []*vo.DocumentRouteCandidate {
	if routeDecision == nil || len(routeDecision.Documents) == 0 || !allowedScope.Executable() {
		return nil
	}

	return utils.Filter(routeDecision.Documents, func(c *vo.DocumentRouteCandidate) bool {
		return c != nil && allowedScope.Contains(c.DocumentId, c.LastIndexTaskId)
	})
}

// selectRecommendation 从候选文档中选择推荐文档作为主文档
func (r *RouteStage) selectRecommendation(routeDecision *vo.KnowledgeRouteDecision, candidateDocuments []*vo.DocumentRouteCandidate) *vo.DocumentRouteCandidate {
	if routeDecision == nil || routeDecision.Confidence <= 0 ||
		routeDecision.RouteStatus != enum.RouteStatusSuccess ||
		len(candidateDocuments) == 0 {
		return nil
	}

	confidence := routeDecision.Confidence
	if confidence < recommendationThreshold || confidence > 1.0 {
		return nil
	}

	topCandidate := candidateDocuments[0]
	if !topCandidate.IsValidScore() || !r.isOriginalTopCandidate(routeDecision, topCandidate) {
		return nil
	}

	return topCandidate
}

// isOriginalTopCandidate 判断候选是否与路由决策的原始top候选匹配
func (r *RouteStage) isOriginalTopCandidate(routeDecision *vo.KnowledgeRouteDecision, candidate *vo.DocumentRouteCandidate) bool {
	if len(routeDecision.Documents) == 0 || candidate == nil {
		return false
	}
	originalTop := routeDecision.Documents[0]
	if originalTop == nil {
		return false
	}
	return originalTop.DocumentId == candidate.DocumentId &&
		originalTop.LastIndexTaskId == candidate.LastIndexTaskId
}

// fallbackDocuments 获取后备候选文档
//
// 在路由决策不可用或置信度偏低时，从全部可检索文档中基于元数据（名称/标签等）匹配查询词，
// 返回得分最高的前 limit 个候选，理由统一标注为"低置信度时基于文档元数据进行保守扩范围候选"。
func (r *RouteStage) fallbackDocuments(ctx context.Context, question, rewriteQuestion string, limit int) []*vo.DocumentRouteCandidate {
	// 拉取全部可检索文档；失败或为空时返回 nil（上游可继续用主文档或混合检索兜底）
	docs, err := r.docGateway.FetchRetrieveDocuments(ctx)
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
	result := make([]*vo.DocumentRouteCandidate, 0, limit)
	for i, desc := range docs {
		if i >= limit {
			break
		}
		result = append(result, &vo.DocumentRouteCandidate{
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
func (r *RouteStage) mergeCandidates(primary, secondary []*vo.DocumentRouteCandidate, limit int) []*vo.DocumentRouteCandidate {
	// 使用 map 做 DocumentId 维度的去重；primary 先遍历以保证优先级
	merged := make(map[int64]*vo.DocumentRouteCandidate)
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
	result := make([]*vo.DocumentRouteCandidate, 0, limit)
	for _, id := range ids {
		if len(result) >= limit {
			break
		}
		result = append(result, merged[id])
	}
	return result
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
	return utils.Limit(maputil.Keys(terms), 40)
}

// fallbackDescriptorScore 计算后备文档匹配分数
func (r *RouteStage) fallbackDescriptorScore(metadata *vo.DocumentMetadata, queryTerms []string) float64 {
	content := strings.Join([]string{
		metadata.DocumentName,
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
