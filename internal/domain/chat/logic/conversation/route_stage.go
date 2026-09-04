package conversation

import (
	"context"

	"github.com/swiftbit/know-agent/common/logx"
	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/chat/adapter"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/enum"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
)

const (
	recommendationThreshold            = 0.55
	stageMaxEvidenceAnchorSnippetChars = 300 // 证据锚点内容片段最大字符数
	stageMaxRecentExchanges            = 3   // 追问承接最多回溯的对话轮次
)

// RouteStage 路由判定阶段
type RouteStage struct {
	repo            adapter.ChatRepository
	knowledgeRouter KnowledgeRouter
	docGateway      adapter.DocumentGateway
}

var _ Stage = (*RouteStage)(nil)

var _ ConditionalStage = (*RouteStage)(nil)

func NewRouteStage(repo adapter.ChatRepository, knowledgeRouter KnowledgeRouter, docGateway adapter.DocumentGateway) *RouteStage {
	return &RouteStage{
		repo:            repo,
		knowledgeRouter: knowledgeRouter,
		docGateway:      docGateway,
	}
}

// Name 阶段名称
func (r *RouteStage) Name() string {
	return enum.ConversationTraceStageRoute.Name
}

// Order 阶段顺序
func (r *RouteStage) Order() int {
	return enum.ConversationTraceStageRoute.Order
}

// ShouldExecute 仅当执行计划已就绪且语义缓存未命中时执行
func (r *RouteStage) ShouldExecute(ctx context.Context, convCtx *Context) bool {
	if convCtx.ExecutionPlan.Load() == nil {
		return false
	}
	return !convCtx.cache.IsCacheHit()
}

// Execute 执行路由判定
func (r *RouteStage) Execute(ctx context.Context, convCtx *Context) error {
	execPlan := convCtx.ExecutionPlan.Load()

	switch convCtx.ChatMode {
	case enum.ChatQueryModeDocument:
		// 记录影子路由（仅用于离线分析，失败只告警）
		if err := r.knowledgeRouter.RecordShadowRoute(ctx, NewKnowledgeRouteInput(convCtx, execPlan.RewriteQuestion)); err != nil {
			logx.Warnf("记录影子路由失败: %v", err)
		}
	case enum.ChatQueryModeAutoDocument:
		r.routeAutoDocument(ctx, convCtx, execPlan)
	default:
	}

	convCtx.SetExecutePlan(execPlan)
	return nil
}

// routeAutoDocument 自动文档问答路由：解析范围 → 执行路由 → 筛选候选 → 确定主文档。
//
// 处理流程：
//   - 范围解析：从快照解析允许执行的知识范围
//   - 知识路由：双路输入执行路由，失败则生成不可用决策继续执行
//   - 候选过滤：用允许范围过滤路由产出的候选文档
//   - 主文档选择：置信度达标则写回 convCtx；否则不指定主文档
//   - 追踪上报：汇总路由状态、范围信息、候选数及主文档结果
func (r *RouteStage) routeAutoDocument(ctx context.Context, convCtx *Context, execPlan *vo.ConversationExecutionPlan) {
	// 解析知识库选择快照，确定允许执行的知识范围
	allowedScope := convCtx.KnowledgeBaseSelectionSnapshot.ResolveAllowedExecutionScope()

	// 启动路由阶段追踪
	ctx = vo.OnStart(ctx, enum.ConversationTraceStageRoute, &vo.StageInput{SummaryText: "正在执行知识范围、主题、候选文档路由。", Snapshot: nil})

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
	if confidentTop {
		convCtx.SelectedDocumentName = topDocument.DocumentName
		convCtx.SelectedDocumentId = topDocument.DocumentId
		convCtx.SelectedTaskId = topDocument.LastIndexTaskId
	}

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

	return
}

// loadFollowUpAnchors 加载并过滤上一轮证据锚点（失败仅告警，不阻断 agent）
func (r *RouteStage) loadFollowUpAnchors(ctx context.Context, convCtx *Context) vo.EvidenceAnchors {
	anchors, err := r.loadRecentEvidenceAnchors(ctx, convCtx.ConversationId)
	if err != nil {
		logx.Warnf("加载最近证据锚点失败, conversationId=%s, err=%v", convCtx.ConversationId, err)
		return nil
	}
	docIds := []int64{convCtx.SelectedDocumentId}
	if convCtx.ChatMode == enum.ChatQueryModeAutoDocument {
		docIds = convCtx.KnowledgeBaseSelectionSnapshot.SelectedDocumentIds()
	}
	return r.filterValidEvidenceAnchors(anchors, docIds...)
}

// loadRecentEvidenceAnchors 从最近几轮已完成回答中加载证据锚点（追问承接信息源）
func (r *RouteStage) loadRecentEvidenceAnchors(ctx context.Context, conversationId string) (vo.EvidenceAnchors, error) {
	if conversationId == "" {
		return nil, nil
	}
	exchanges, err := r.repo.ListRecentExchanges(ctx, conversationId, stageMaxRecentExchanges)
	if err != nil || len(exchanges) == 0 {
		return nil, err
	}

	var anchors vo.EvidenceAnchors
	for _, exchange := range exchanges {
		if exchange == nil || !exchange.IsCompleted() || len(exchange.References) == 0 {
			continue
		}
		for _, ref := range exchange.References {
			anchor := ref.ToEvidenceAnchor(stageMaxEvidenceAnchorSnippetChars)
			if anchor == nil {
				continue
			}
			anchors = append(anchors, anchor)
			if len(anchors) >= stageMaxRecentExchanges {
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
