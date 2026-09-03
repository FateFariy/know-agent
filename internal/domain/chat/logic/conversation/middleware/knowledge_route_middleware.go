package middleware

import (
	"context"

	"github.com/swiftbit/know-agent/common/logx"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/conversation"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/enum"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
)

const (
	recommendationThreshold = 0.55 // 推荐候选文档的置信度阈值
)

// KnowledgeRouteMiddleware 知识路由中间件：在 agent 启动前完成不依赖 agent 决策的知识路由。
// 职责范围：
//   - Document：仅记录影子路由（离线分析）；
//   - AutoDocument：解析允许范围 → 知识路由 → 选择主文档并写回 convCtx / 刷新检索计划范围，
//     范围不可执行时置澄清兜底。
//
// 必须注册在 MemoryLoadMiddleware 之后（依赖其创建的 execPlan 与检索计划）。
type KnowledgeRouteMiddleware struct {
	BaseAgentMiddleware
	knowledgeRouter conversation.KnowledgeRouter
}

// NewKnowledgeRouteMiddleware 创建知识路由中间件
func NewKnowledgeRouteMiddleware(knowledgeRouter conversation.KnowledgeRouter) *KnowledgeRouteMiddleware {
	return &KnowledgeRouteMiddleware{knowledgeRouter: knowledgeRouter}
}

// Name 中间件名称
func (m *KnowledgeRouteMiddleware) Name() string { return "knowledge-route" }

// BeforeAgent 在 agent 启动前按 ChatMode 执行知识路由
func (m *KnowledgeRouteMiddleware) BeforeAgent(ctx context.Context, convCtx *conversation.Context, input *BeforeAgentInput) (*BeforeAgentOutput, error) {
	if convCtx == nil {
		return &BeforeAgentOutput{Instruction: input.Instruction}, nil
	}
	execPlan := convCtx.ExecutionPlan.Load()
	if execPlan == nil {
		return &BeforeAgentOutput{Instruction: input.Instruction}, nil
	}

	switch convCtx.ChatMode {
	case enum.ChatQueryModeDocument:
		if err := m.knowledgeRouter.RecordShadowRoute(ctx, conversation.NewKnowledgeRouteInput(convCtx, execPlan.RewriteQuestion)); err != nil {
			logx.Warnf("记录影子路由失败: %v", err)
		}
	case enum.ChatQueryModeAutoDocument:
		if err := m.routeAutoDocument(ctx, convCtx, execPlan); err != nil {
			return &BeforeAgentOutput{Instruction: input.Instruction}, err
		}
	}
	return &BeforeAgentOutput{Instruction: input.Instruction}, nil
}

// routeAutoDocument 自动文档问答：范围解析 → 知识路由 → 选择主文档写回，
// 范围不可执行或主文档不明确时按原语义兜底。
func (m *KnowledgeRouteMiddleware) routeAutoDocument(ctx context.Context, convCtx *conversation.Context, execPlan *vo.ConversationExecutionPlan) error {
	// 解析知识库选择快照，确定允许执行的知识范围
	allowedScope := convCtx.KnowledgeBaseSelectionSnapshot.ResolveAllowedExecutionScope()

	// 启动知识路由阶段追踪
	ctx = vo.OnStart(ctx, enum.ConversationTraceStageRoute, &vo.StageInput{SummaryText: "正在执行知识范围、主题、候选文档路由。", Snapshot: nil})

	// 执行知识路由（原始问题 + 改写问题做双路输入）
	routeDecision, err := m.knowledgeRouter.Route(ctx, conversation.NewKnowledgeRouteInput(convCtx, execPlan.RewriteQuestion))
	if err != nil {
		routeDecision = vo.NewUnavailableRouteDecision("ROUTE_ADVISOR_FAILURE")
		logx.Warnf("知识路由失败: %v", err)
	}

	// 选择候选文档（基于路由决策 + 允许范围过滤）
	inScopeCandidates := allowedScope.FilterCandidates(routeDecision.Documents)

	// 选择推荐候选作为主文档
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

	return nil
}
