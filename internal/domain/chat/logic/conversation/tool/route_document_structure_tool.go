package tool

import (
	"context"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/conversation"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/route"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/enum"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
)

type RouteStructureInput struct {
	Question         string                        `json:"query" jsonschema_description:"改写问题"`            // 改写问题
	NavigationAction enum.DocumentNavigationAction `json:"navigationAction" jsonschema_description:"导航动作"` // 导航动作
	SectionAnchors   []string                      `json:"sectionAnchors" jsonschema_description:"显式章节锚点"` // 显式章节锚点
}

type RouteStructureOutput struct {
	Action        string `json:"action"`                  // fresh_topic / item_reference / structure_navigation
	RetrievalMode string `json:"retrievalMode"`           // 恒为 RETRIEVAL
	TargetSection string `json:"targetSection,omitempty"` // 结构锚点提示
	ItemIndex     *int   `json:"itemIndex,omitempty"`
}

type RouteDocumentStructureTool struct {
	route DocumentRouter
}

func NewRouteDocumentStructureTool(route DocumentRouter) *RouteDocumentStructureTool {
	return &RouteDocumentStructureTool{route: route}
}

func (r *RouteDocumentStructureTool) Info(ctx context.Context) *Info {
	return &Info{
		Name:        "route_document_structure",
		Description: "Route document structure",
	}
}

func (r *RouteDocumentStructureTool) Invoke(ctx context.Context, input *RouteStructureInput) (*RouteStructureOutput, error) {
	convCtx := conversation.AgentContextFrom(ctx)
	execPlan := convCtx.ExecutionPlan.Load()
	// 启动文档内路由阶段追踪
	ctx = vo.OnStart(ctx, enum.ConversationTraceStageRoute, &vo.StageInput{SummaryText: "正在进行结构导航查询", Snapshot: nil})

	// 构造导航输入，调用 Router 做文档内意图路由（输出执行模式、章节锚点等）
	nav := &route.NavigationInput{
		DocumentId:       convCtx.SelectedDocumentId,
		Question:         input.Question,
		RewriteQuestion:  input.Question,
		SubQuestions:     execPlan.SubQuestions(),
		SectionAnchors:   input.SectionAnchors,
		NavigationAction: input.NavigationAction,
	}
	navigationDecision, err := r.route.Route(ctx, nav)
	if err != nil {
		ctx = vo.OnError(ctx, "执行路由失败。", err)
		return nil, err
	}

	// 构造路由结果快照（执行模式 / 章节提示 / 条目编号 / 摘要文本），写入追踪
	snapshot := map[string]any{
		"targetSectionHint": utils.Trim(navigationDecision.StructureAnchor.TargetSectionHint),
		"navigationSummary": utils.Trim(navigationDecision.SummaryText),
	}
	if navigationDecision.ItemAnchor != nil {
		snapshot["targetItemIndex"] = navigationDecision.ItemAnchor.ItemIndex
	}

	// 组装最终执行计划：写入执行模式、导航决策、无证据回复提示、检索计划
	execPlan.NavigationDecision = navigationDecision
	convCtx.AddUsedTools("route_document_structure_tool")

	ctx = vo.OnEnd(ctx, &vo.StageOutput{SummaryText: "执行路由完成。", Snapshot: snapshot})

	return nil, nil
}

// copyStructureNavigationResult 复制结构导航结果 todo 待实现
func copyStructureNavigationResult(decision *vo.DocumentNavigationDecision) *vo.StructureNavigationResult {
	if decision == nil {
		return nil
	}
	return nil
}

// copyStructureAnchor 复制结构锚点
func copyStructureAnchor(decision *vo.DocumentNavigationDecision) *vo.ConversationStructureAnchor {
	if decision == nil || decision.StructureAnchor == nil {
		return nil
	}
	anchor := *decision.StructureAnchor
	return &anchor
}

// copyItemAnchor 复制条目锚点
func copyItemAnchor(decision *vo.DocumentNavigationDecision) *vo.ConversationItemAnchor {
	if decision == nil || decision.ItemAnchor == nil {
		return nil
	}
	item := *decision.ItemAnchor
	return &item
}
