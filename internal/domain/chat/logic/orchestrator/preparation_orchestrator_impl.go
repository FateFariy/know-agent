package orchestrator

import (
	"context"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/duke-git/lancet/v2/maputil"
	"github.com/duke-git/lancet/v2/stream"
	"github.com/duke-git/lancet/v2/strutil"
	"github.com/zeromicro/go-zero/core/logx"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/chat/adapter"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/trace"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
	"github.com/swiftbit/know-agent/internal/domain/chat/support"
	doclog "github.com/swiftbit/know-agent/internal/domain/document/logic"
	vo2 "github.com/swiftbit/know-agent/internal/domain/document/model/vo"
	kelog "github.com/swiftbit/know-agent/internal/domain/knowledge/logic"
	klvo "github.com/swiftbit/know-agent/internal/domain/knowledge/model/vo"
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
	memoryLogic            logic.SessionMemoryLogic
	rewriteLogic           logic.QueryRewriteLogic
	documentQuestionRouter logic.DocumentQuestionRouteLogic
	knowledgeRouteLogic    kelog.KnowledgeRouteLogic
	lifecycleLogic         doclog.LifecycleLogic
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

// NewChatPreparationOrchestratorImpl 创建聊天准备编排器实例
func NewChatPreparationOrchestratorImpl(svcCtx *svc.ServiceContext,
	repo adapter.ChatRepository,
	memoryLogic logic.SessionMemoryLogic,
	rewriteLogic logic.QueryRewriteLogic,
	documentQuestionRouter logic.DocumentQuestionRouteLogic,
	knowledgeRoute kelog.KnowledgeRouteLogic,
	lifecycleLogic doclog.LifecycleLogic,
) *PreparationOrchestratorImpl {
	return &PreparationOrchestratorImpl{
		repo:                   repo,
		memoryLogic:            memoryLogic,
		rewriteLogic:           rewriteLogic,
		documentQuestionRouter: documentQuestionRouter,
		knowledgeRouteLogic:    knowledgeRoute,
		lifecycleLogic:         lifecycleLogic,
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

// Prepare 准备对话执行计划。
//
// 执行顺序：
//  1. prepareCommon：执行所有模式共享的准备（记忆装载、时间信号、问题改写）
//  2. 按 chatMode 分发到对应的子计划方法：
//     - OpenChat → prepareOpenChat（开放式 Agent）
//     - Document → prepareDocumentMode（指定文档问答）
//     - AutoDocument → prepareAutoDocumentMode（自动知识路由 + 文档内导航）
//  3. 返回最终执行计划
func (o *PreparationOrchestratorImpl) Prepare(ctx context.Context, convCtx *vo.ConversationContext) (*vo.ConversationExecutionPlan, error) {
	// 步公共准备（所有 chatMode 共享）
	execPlan, err := o.prepareCommon(ctx, convCtx)
	if err != nil {
		return nil, err
	}

	// 根据 chatMode 分发到具体的子计划方法
	switch convCtx.ChatMode {
	case vo.ChatQueryModeOpenChat:
		// 开放式聊天：直接走 ReactAgent
		err = o.prepareOpenChat(ctx, convCtx, execPlan)
	case vo.ChatQueryModeDocument:
		// 指定文档问答：路由到所选文档内做导航
		err = o.prepareDocumentMode(ctx, convCtx, execPlan)
	case vo.ChatQueryModeAutoDocument:
		// 自动文档问答：先做知识路由选文档，再在文档内导航
		err = o.prepareAutoDocumentMode(ctx, convCtx, execPlan)
	default:
		return nil, fmt.Errorf("不支持的聊天模式: %s", vo.ChatQueryModeName(convCtx.ChatMode))
	}
	// 子计划方法失败则直接返回错误
	if err != nil {
		return nil, err
	}
	// 返回最终执行计划（子方法已就地写入 execPlan）
	return execPlan, nil
}

// prepareCommon 执行所有聊天模式共享的准备步骤。
//
// 具体内容：
//  1. 装载会话记忆（含长期摘要、近期转录、压缩信息）
//  2. 构建历史规划上下文与问题历史上下文（供模型引用）
//  3. 判断时间敏感/实时搜索需求（输出到执行计划，供检索/Agent 侧使用）
//  4. 组装初始执行计划（所有检索字段先以原始问题为兜底值）
//  5. 非 OpenChat 模式下：调用问题改写，更新 Rewrite/Retrieval 相关字段
func (o *PreparationOrchestratorImpl) prepareCommon(ctx context.Context, convCtx *vo.ConversationContext) (*vo.ConversationExecutionPlan, error) {
	// 规范化原始问题
	question := strutil.Trim(convCtx.Question)

	// 装载会话记忆（含长期摘要、近期转录、压缩信息）
	memoryContext, err := o.summarizeHistory(ctx, convCtx)
	if err != nil {
		return nil, err
	}

	// 构建历史规划上下文与问题历史上下文
	//  - historyPlanningContext：聚合长期记忆，供 Agent 做规划引用
	//  - historySummary：以可读文本形式描述规划历史
	//  - questionHistoryContext：当前问题 + 近期对话转录，用于后续改写与检索
	historyPlanningContext := vo.NewHistoryPlanningContext(memoryContext.Summary)
	historySummary := o.buildPlanningHistory(memoryContext, historyPlanningContext)
	questionHistoryContext := vo.NewQuestionHistoryContext(question, strutil.Trim(memoryContext.RecentTranscript), o.questionHistoryMaxChars)

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
		NoEvidenceReply:              o.noEvidenceReply,
	}

	// 非 OpenChat 模式需要问题改写产物（用于更精准的文档检索）
	//  - 若 RAG 未启用，直接返回错误以避免后续空引用
	//  - 改写完成后，同时更新 Rewrite 与 Retrieval 字段（路由阶段可能进一步覆盖 Retrieval）
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

// prepareOpenChat 开放式聊天：直接走 ReactAgent 模式，不做文档路由与检索准备。
//
// 步骤：
//  1. 设置执行模式为 ExecutionModeReactAgent
//  2. 启动并完成路由追踪阶段，写入快照（chatMode / executionMode / 时间信号）
func (o *PreparationOrchestratorImpl) prepareOpenChat(ctx context.Context, convCtx *vo.ConversationContext, execPlan *vo.ConversationExecutionPlan) error {
	// 设置执行模式为 ReactAgent（完全由下游 Agent 自主规划）
	execPlan.Mode = vo.ExecutionModeReactAgent

	// 启动路由追踪阶段（此处以 Rewrite 阶段为名，记录判定结果与时间信号）
	stage, err := o.tracer.StartStage(ctx, convCtx.Trace, vo.ConversationTraceStageRewrite, vo.ExecutionModeReactAgent.String(), "路由到开放式 Agent。", nil)
	if err != nil {
		return err
	}
	// 写入快照：聊天模式、执行模式、时间信号关键字段
	snapshot := map[string]any{
		"chatMode":                     vo.ChatQueryModeName(convCtx.ChatMode),
		"executionMode":                vo.ExecutionModeReactAgent.String(),
		"requiresRealTimeSearch":       execPlan.RequiresRealTimeSearch,
		"requiresCurrentDateAnchoring": execPlan.RequiresCurrentDateAnchoring,
	}
	if err = o.tracer.CompleteStage(ctx, stage, "已判定走开放式 Agent 路径。", snapshot); err != nil {
		return err
	}

	return nil
}

// prepareDocumentMode 指定文档问答：用户已在界面选择具体文档。
//
// 步骤：
//  1. 校验所选文档/索引任务 ID 是否有效
//  2. 记录影子路由（便于后续优化自动路由；失败不影响业务流程）
//  3. 调用文档内路由与终稿组装（routeAndFinalizePlan）
func (o *PreparationOrchestratorImpl) prepareDocumentMode(ctx context.Context, convCtx *vo.ConversationContext, execPlan *vo.ConversationExecutionPlan) error {
	// 校验所选文档 ID 与索引任务 ID（必填）
	if convCtx.SelectedDocumentId == 0 || convCtx.SelectedTaskId == 0 {
		return fmt.Errorf("当前文档问答模式缺少有效的文档范围")
	}

	// 记录影子路由（仅用于离线分析，失败只告警）
	if err := o.knowledgeRouteLogic.RecordShadowRoute(ctx, convCtx.ExchangeId, convCtx.SelectedDocumentId, convCtx.ConversationId, convCtx.Question, execPlan.RewriteQuestion); err != nil {
		Warnf("记录影子路由失败: %v", err)
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
	return o.routeAndFinalizePlan(ctx, convCtx, execPlan)
}

// prepareAutoDocumentMode 自动文档问答：执行知识路由 → 必要时澄清 → 确定主文档 → 文档内导航 → 生成计划。
//
// 关键分支：
//   - 路由失败：记录告警，使用空路由决策继续执行
//   - 置信度 ≥ 0.55 且有候选：使用候选首位作为主文档
//   - 否则：不指定主文档，退化为多文档范围混合检索
//   - 需要澄清：返回 Clarification 模式，由用户选择目标知识
func (o *PreparationOrchestratorImpl) prepareAutoDocumentMode(ctx context.Context, convCtx *vo.ConversationContext, execPlan *vo.ConversationExecutionPlan) error {
	// 启动路由阶段追踪（标识为 AUTO_DOCUMENT）
	stage, err := o.tracer.StartStage(ctx, convCtx.Trace, vo.ConversationTraceStageRoute, "AUTO_DOCUMENT", "正在生成知识范围候选。", nil)
	if err != nil {
		return err
	}

	// 执行知识路由（原始问题 + 改写问题做双路输入）
	//  - 路由失败时仅告警，并以空决策对象兜底（避免后续代码 panic）
	routeDecision, err := o.knowledgeRouteLogic.Route(ctx, convCtx.Question, execPlan.RewriteQuestion)
	if err != nil {
		routeDecision = &klvo.KnowledgeRouteDecision{}
		Warnf("知识路由失败: %v", err)
	}
	// 记录自动路由（用于离线分析；失败只告警）
	if err = o.knowledgeRouteLogic.RecordAutoRoute(ctx, convCtx.ExchangeId, convCtx.ConversationId, convCtx.Question, execPlan.RewriteQuestion, routeDecision); err != nil {
		Warnf("记录自动路由失败: %v", err)
	}

	// 选择候选文档，提取候选的文档 ID 与索引任务 ID 列表（供后续多文档检索使用）
	candidateDocuments := o.selectAutoCandidates(ctx, routeDecision, convCtx.Question, execPlan.RewriteQuestion)
	execPlan.RetrievalDocumentIds = o.extractDocumentIds(candidateDocuments)
	execPlan.RetrievalTaskIds = o.extractTaskIds(candidateDocuments)

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
	if err = o.tracer.CompleteStage(ctx, stage, "知识范围路由完成。", snapshot); err != nil {
		return err
	}

	// 检查是否需要澄清（多个候选相近、路由歧义等）
	//  - 需要澄清时直接返回 Clarification 模式执行计划（含回复文案、选项、理由）
	if o.shouldAskClarification(routeDecision, candidateDocuments) {
		execPlan.Mode = vo.ExecutionModeClarification
		execPlan.ClarificationReply = o.buildClarificationReply(candidateDocuments)
		execPlan.ClarificationOptions = o.buildClarificationOptions(candidateDocuments)
		execPlan.ClarificationReason = o.buildClarificationReason(routeDecision, candidateDocuments)
		return nil
	}

	// 写入主文档信息（若无高置信主文档，则 DocumentId 为 0，退化为多文档混合检索）
	execPlan.SelectedDocumentId = topDocument.DocumentId
	execPlan.SelectedDocumentName = topDocument.DocumentName
	execPlan.SelectedTaskId = topDocument.LastIndexTaskId

	// 在选定的文档内做路由，并组装最终执行计划
	if err = o.routeAndFinalizePlan(ctx, convCtx, execPlan); err != nil {
		return err

	}
	return nil
}

// routeAndFinalizePlan 在文档内完成意图路由与执行计划终稿组装。
//
// 总体流程：
//  1. 启动路由追踪阶段 → 调用 documentQuestionRouter 做文档内意图判定
//  2. 路由失败时记录失败并向上返回
//  3. 路由成功后将执行模式/章节锚点/条目锚点写入快照，提交追踪
//  4. 从路由结果中选取检索问题与子问题列表（空值回退到改写问题）
//  5. 组装最终执行计划（执行模式 / 导航决策 / 无证据回复提示）
//  6. 打印关键编排结果并返回
func (o *PreparationOrchestratorImpl) routeAndFinalizePlan(ctx context.Context, convCtx *vo.ConversationContext, execPlan *vo.ConversationExecutionPlan) error {
	// 启动文档内路由阶段追踪，并以 "混合检索" 为默认模式名
	stage, err := o.tracer.StartStage(ctx, convCtx.Trace, vo.ConversationTraceStageRoute, vo.ExecutionModeRetrieval.Name(), "正在判定图查询还是混合检索。", nil)
	if err != nil {
		return err
	}

	// 构造改写结果对象，调用 Router 做文档内意图路由（输出执行模式、章节锚点等）
	rewriteResult := vo.NewQuestionRewriteResult(execPlan.RewriteQuestion, execPlan.RewriteSubQuestions)
	navigationDecision, err := o.documentQuestionRouter.Route(ctx, execPlan.SelectedDocumentId, convCtx.Question, rewriteResult)
	if err != nil {
		// 路由失败：先记录追踪失败，再向上层返回错误（FailStage 失败时直接返回）
		if err = o.tracer.FailStage(ctx, stage, "执行路由失败。", err, nil); err != nil {
			return err
		}
		return err
	}

	// 构造路由结果快照（执行模式 / 章节提示 / 条目编号 / 摘要文本），写入追踪
	snapshot := map[string]any{
		"executionMode":     navigationDecision.ExecutionModeName,
		"targetSectionHint": strutil.Trim(navigationDecision.StructureAnchor.TargetSectionHint),
		"targetItemIndex":   navigationDecision.ItemAnchor.ItemIndex,
		"navigationSummary": strutil.Trim(navigationDecision.SummaryText),
	}
	if err = o.tracer.CompleteStage(ctx, stage, "执行路由完成。", snapshot); err != nil {
		return err
	}

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
	execPlan.NoEvidenceReply = o.buildDocumentModeNoEvidenceReply(convCtx.Question, execPlan.RequiresRealTimeSearch)

	// 打印关键编排结果（会话ID、模式、原始问题、改写问题、检索问题、执行模式、目标章节）
	logx.Infof("聊天编排完成: conversationId=%s, chatMode=%s, originalQuestion='%s', rewriteQuestion='%s', retrievalQuestion='%s', executionMode=%s, targetSection='%s",
		convCtx.ConversationId, vo.ChatQueryModeName(convCtx.ChatMode), strutil.Trim(convCtx.Question),
		execPlan.RewriteQuestion, execPlan.RetrievalQuestion, execPlan.Mode.Name(), navigationDecision.StructureAnchor.TargetSectionHint)

	return nil
}

// summarizeHistory 构建会话记忆。
//
// 执行步骤：
//  1. 启动记忆追踪阶段
//  2. 调用 memoryLogic 装载长期摘要与近期转录（含压缩状态）
//  3. 失败时记录失败追踪并返回
//  4. 成功时写入快照（压缩状态、覆盖的 exchange、摘要内容），提交追踪后返回
func (o *PreparationOrchestratorImpl) summarizeHistory(ctx context.Context, convCtx *vo.ConversationContext) (*vo.MemoryContext, error) {
	// 启动记忆追踪阶段，使用 chatMode 作为执行模式名
	memoryStage, err := o.tracer.StartStage(ctx, convCtx.Trace, vo.ConversationTraceStageMemory, vo.ChatQueryModeName(convCtx.ChatMode), "正在装载会话记忆与最近窗口。", nil)
	if err != nil {
		return nil, err
	}

	// 调用 memoryLogic 装载记忆上下文（含长期摘要、近期转录、压缩状态）
	memoryContext, err := o.memoryLogic.LoadMemoryContext(ctx, convCtx.ConversationId, convCtx.Trace)
	if err != nil {
		// 失败：先记录追踪失败，再向上返回错误（FailStage 失败直接返回）
		if err = o.tracer.FailStage(ctx, memoryStage, "会话记忆装载失败。", err, nil); err != nil {
			return nil, err
		}
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
	if err = o.tracer.CompleteStage(ctx, memoryStage, "会话记忆装载完成。", snapshot); err != nil {
		return nil, err
	}
	return memoryContext, nil
}

// questionRewrite 对问题做模型改写，使其更适合后续知识检索与导航判断。
//
// 执行步骤：
//  1. 启动改写追踪阶段（包含原始问题与历史摘要的快照）
//  2. 调用 rewriteLogic.Rewrite 获取改写结果与子问题拆分
//  3. 失败时记录追踪失败并返回
//  4. 成功时提交追踪并对结果做空值兜底（改写失败则回退原始问题、单子问题列表）
func (o *PreparationOrchestratorImpl) questionRewrite(ctx context.Context, convCtx *vo.ConversationContext, historySummary string) (*vo.QuestionRewriteResult, error) {
	// 启动改写追踪阶段
	rewriteStage, err := o.tracer.StartStage(ctx, convCtx.Trace, vo.ConversationTraceStageRewrite, vo.ExecutionModeRetrieval.String(), "正在生成检索友好的问题表达。", o.buildRewriteStageSnapshot(convCtx.Question, historySummary, nil))
	if err != nil {
		return nil, err
	}
	// 调用改写逻辑（原始问题 + 历史摘要 → 改写问题 + 子问题）
	rewriteResult, err := o.rewriteLogic.Rewrite(ctx, convCtx.Question, historySummary, convCtx.Trace)
	if err != nil {
		// 失败：记录追踪失败并向上返回错误
		if err = o.tracer.FailStage(ctx, rewriteStage, "问题改写失败。", err, o.buildRewriteStageSnapshot(convCtx.Question, historySummary, nil)); err != nil {
			return nil, err
		}
		return nil, err
	}
	// 提交改写追踪（包含改写结果快照以便离线分析）
	if err = o.tracer.CompleteStage(ctx, rewriteStage, "问题改写完成。", o.buildRewriteStageSnapshot(convCtx.Question, historySummary, rewriteResult)); err != nil {
		return nil, err
	}

	// 对改写结果做兜底处理
	//  - RewrittenQuestion 为空时回退到原始问题
	//  - SubQuestions 为空时使用改写问题作为单元素列表
	rewriteResult.RewrittenQuestion = utils.BlankToDefault(rewriteResult.RewrittenQuestion, strutil.Trim(convCtx.Question))
	if len(rewriteResult.SubQuestions) == 0 {
		rewriteResult.SubQuestions = []string{rewriteResult.RewrittenQuestion}
	}
	return rewriteResult, nil
}

// buildRewriteStageSnapshot 构建改写阶段的统一快照，供追踪的 Start/Fail/Complete 三处复用。
//
// 快照字段：
//   - 原始问题、历史摘要（始终输出）
//   - 若 rewriteResult 非空，则输出改写后的问题、子问题列表、模型原始输出
//   - 改写开关、temperature、topP、thinking 等模型参数，用于离线分析
func (o *PreparationOrchestratorImpl) buildRewriteStageSnapshot(question, historySummary string, rewriteResult *vo.QuestionRewriteResult) map[string]any {
	snapshot := make(map[string]any)
	snapshot["originalQuestion"] = strutil.Trim(question)
	snapshot["historyContext"] = strutil.Trim(historySummary)

	// 仅当改写结果存在时追加输出相关字段（避免在 Start 阶段填充空值）
	if rewriteResult != nil {
		snapshot["rewriteQuestion"] = strutil.Trim(rewriteResult.RewrittenQuestion)
		snapshot["subQuestions"] = rewriteResult.SubQuestions
		snapshot["rawModelOutput"] = strutil.Trim(rewriteResult.RawModelOutput)
	}
	// 追加当前配置的改写参数（便于离线分析参数影响）
	snapshot["rewriteOverrideEnabled"] = o.rewriteEnabled
	snapshot["rewriteTemperature"] = o.temperature
	snapshot["rewriteTopP"] = o.topP
	snapshot["rewriteThinking"] = o.thinking
	return snapshot
}

// buildPlanningHistory 构建规划历史文本。
//
// 策略：
//  1. 将历史规划上下文中的"会话目标"、"已确认事实"、"待跟进问题"、"检索提示"拼接成结构化文本
//  2. 与近期对话转录合并（占预算 65%，结构化文本占剩余预算）
//  3. 两者以 "\n\n" 分隔；总长度上限由 planningHistoryMaxChars 控制
//
// 最终用于问题改写与 Agent 规划引用。
func (o *PreparationOrchestratorImpl) buildPlanningHistory(memoryContext *vo.MemoryContext, historyPlanningContext *vo.HistoryPlanningContext) string {
	// 拼接结构化历史（会话目标 + 三类要点提示）
	var sb strings.Builder
	o.appendSection(&sb, "会话目标", historyPlanningContext.ConversationGoal)
	o.appendBulletSection(&sb, "已确认事实", historyPlanningContext.StableFacts)
	o.appendBulletSection(&sb, "待跟进问题", historyPlanningContext.PendingQuestions)
	o.appendBulletSection(&sb, "检索提示", historyPlanningContext.RetrievalHints)
	structuredHistory := strutil.Trim(sb.String())
	recentTranscript := strutil.Trim(memoryContext.RecentTranscript)

	maxChars := o.planningHistoryMaxChars
	// 近期转录为空 → 仅返回结构化文本（ClipHead 保留开头，避免尾部上下文缺失）
	if recentTranscript == "" {
		return utils.ClipHead(structuredHistory, maxChars)
	}

	// 按 65% 预算切分近期转录（ClipTail 保留末尾最新的对话），剩余预算留给结构化历史
	recentBudget := int(math.Round(float64(maxChars) * 0.65))
	recentPart := utils.ClipTail(recentTranscript, recentBudget)

	// 结构化历史预算 = 总预算 - 近期转录长度 - 分隔符长度（取 max 0 防止负数）
	structuredBudget := max(0, maxChars-utf8.RuneCountInString(recentPart)-2)
	structuredPart := utils.ClipHead(structuredHistory, structuredBudget)

	// 合并结构化文本与近期转录（非空项以 "\n\n" 分隔）
	return utils.JoinNonBlank("\n\n", structuredPart, recentPart)
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

// selectAutoCandidates 根据路由决策选择自动候选文档。
//
// 策略与分支：
//  1. 路由决策为空或无文档 → 使用 fallbackDocuments 做兜底（上限 5）
//  2. 候选数量阈值：置信度 ≥ 0.80 时取前 5，否则取前 3（高置信度更保守，低置信度多给候选以召回）
//  3. 候选为空时同样回退到 fallbackDocuments
//  4. 置信度 < 0.55 时将路由候选与 fallback 候选合并（扩大范围以弥补低置信度）
//  5. 否则直接返回路由候选
func (o *PreparationOrchestratorImpl) selectAutoCandidates(ctx context.Context, routeDecision *klvo.KnowledgeRouteDecision, question, rewriteQuestion string) []*klvo.DocumentRouteCandidate {
	// 分支 1：路由决策为空或无文档 → 使用 fallback 做兜底
	if routeDecision == nil || len(routeDecision.Documents) == 0 {
		return o.fallbackDocuments(ctx, question, rewriteQuestion, 5)
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
	fallbackDocuments := o.fallbackDocuments(ctx, question, rewriteQuestion, candidateLimit)
	// 分支 3：候选为空 → 返回 fallback
	if len(candidates) == 0 {
		return fallbackDocuments
	}

	// 分支 4：置信度 < 0.55 → 合并路由候选与 fallback 候选，扩大检索范围
	if routeDecision.Confidence < 0.55 {
		return o.mergeCandidates(candidates, fallbackDocuments, candidateLimit)
	}

	// 分支 5：正常情况 → 返回路由候选
	return candidates
}

// fallbackDocuments 获取后备候选文档。
//
// 在路由决策不可用或置信度偏低时，从全部可检索文档中基于元数据（名称/标签等）匹配查询词，
// 返回得分最高的前 limit 个候选，理由统一标注为"低置信度时基于文档元数据进行保守扩范围候选"。
func (o *PreparationOrchestratorImpl) fallbackDocuments(ctx context.Context, question, rewriteQuestion string, limit int) []*klvo.DocumentRouteCandidate {
	// 拉取全部可检索文档；失败或为空时返回 nil（上游可继续用主文档或混合检索兜底）
	docs, err := o.lifecycleLogic.ListRetrievableDocuments(ctx)
	if err != nil {
		Warnf("获取可检索文档失败: %v", err)
		return nil
	}
	if len(docs) == 0 {
		return nil
	}

	// 从问题与改写问题中抽取 fallback 查询词（用于元数据匹配打分）
	queryTerms := o.extractFallbackTerms(question, rewriteQuestion)

	// 按文档分别计算 fallback 得分（基于名称/标签与查询词的匹配）
	scoreMap := make(map[int64]float64, len(docs))
	for _, desc := range docs {
		scoreMap[desc.DocumentId] = o.fallbackDescriptorScore(desc, queryTerms)
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
			DocumentId:         desc.DocumentId,
			DocumentName:       desc.DocumentName,
			LastIndexTaskId:    desc.LastIndexTaskId,
			KnowledgeScopeCode: desc.KnowledgeScopeCode,
			KnowledgeScopeName: desc.KnowledgeScopeName,
			BusinessCategory:   desc.BusinessCategory,
			DocumentTags:       desc.DocumentTags,
			Score:              scoreMap[desc.DocumentId],
			Reason:             "低置信度时基于文档元数据进行保守扩范围候选",
		})
	}

	return result
}

// mergeCandidates 合并主候选与次候选并去重（以 DocumentId 为键），最终数量不超过 limit。
// 去重策略：主候选优先（先遍历 primary，其条目被保留），secondary 仅在未出现时被加入。
func (o *PreparationOrchestratorImpl) mergeCandidates(primary, secondary []*klvo.DocumentRouteCandidate, limit int) []*klvo.DocumentRouteCandidate {
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
func (o *PreparationOrchestratorImpl) shouldAskClarification(routeDecision *klvo.KnowledgeRouteDecision, candidateDocuments []*klvo.DocumentRouteCandidate) bool {
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
func (o *PreparationOrchestratorImpl) fallbackDescriptorScore(descriptor *vo2.KnowledgeDocument, queryTerms []string) float64 {
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
		if utf8.RuneCountInString(term) < 2 || strutil.ContainsAny(term, matched) {
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
