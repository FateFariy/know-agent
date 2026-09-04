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
	Question         string                        `json:"query" jsonschema:"required" jsonschema_description:"用户原始问题，原样传递，不要改写"`
	NavigationAction enum.DocumentNavigationAction `json:"navigationAction" jsonschema:"required,enum=SECTION_ADJACENCY_LOOKUP,enum=CHILD_SECTION_DESCEND,enum=ITEM_REFERENCE,enum=FRESH_TOPIC,enum=TOPIC_CONTINUE,enum=TOPIC_SWITCH,enum=SIBLING_SECTION_SWITCH,enum=ANCESTOR_SECTION_RETURN" jsonschema_description:"导航动作类型，可选值：SECTION_ADJACENCY_LOOKUP（查询相邻章节）、CHILD_SECTION_DESCEND（展开下级章节）、ITEM_REFERENCE（定位到具体条目/步骤型问题）、FRESH_TOPIC（普通文档检索主题）、TOPIC_CONTINUE（继续当前主题）、TOPIC_SWITCH（切换主题）、SIBLING_SECTION_SWITCH（切换兄弟章节）、ANCESTOR_SECTION_RETURN（返回上级章节）"`
	SectionAnchors   []string                      `json:"sectionAnchors,omitempty" jsonschema_description:"显式章节锚点列表，当用户明确指定了章节标识时传入，否则不传"`
}

type RouteStructureOutput struct {
	NavigationAction  enum.DocumentNavigationAction `json:"navigationAction"`  // 导航动作
	RootSectionCode   string                        `json:"rootSectionCode"`   // 根章节代码
	RootSectionTitle  string                        `json:"rootSectionTitle"`  // 根章节标题
	TargetSectionHint string                        `json:"targetSectionHint"` // 目标章节提示
	ItemIndex         int                           `json:"itemIndex"`         // 项目索引
	ItemText          string                        `json:"itemText"`          // 项目文本
}

type RouteDocumentStructureTool struct {
	route DocumentRouter
}

func NewRouteDocumentStructureTool(route DocumentRouter) *RouteDocumentStructureTool {
	return &RouteDocumentStructureTool{route: route}
}

func (r *RouteDocumentStructureTool) Info(ctx context.Context) *Info {
	return &Info{
		Name: "route_document_structure",
		Description: `根据用户问题在文档结构中进行导航路由，确定目标章节和条目位置。
				使用场景：当用户提问涉及文档中特定章节、段落、列表条目或需要在章节树中跳转（如“上一章”、“返回上级”、“查看第3节”）时使用此工具。
				参数要求：
				- query：用户的原始提问，请原样传入，不要进行改写。
				- navigationAction：必须根据问题意图从指定的枚举值中选择一个精确匹配的动作。
				- sectionAnchors：仅当用户明确提及章节编号/锚点（如“第2.1节”）时才传入，否则省略。
				不适用场景：全文搜索、跨文档检索或纯知识问答（请使用其他工具）。`,
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
		SectionAnchors:   input.SectionAnchors,
		NavigationAction: input.NavigationAction,
	}
	decision, err := r.route.Route(ctx, nav)
	if err != nil {
		ctx = vo.OnError(ctx, "执行路由失败。", err)
		return nil, err
	}

	// 构造路由结果快照（执行模式 / 章节提示 / 条目编号 / 摘要文本），写入追踪
	snapshot := map[string]any{
		"targetSectionHint": utils.Trim(decision.StructureAnchor.TargetSectionHint),
		"navigationSummary": utils.Trim(decision.SummaryText),
	}
	if decision.ItemAnchor != nil {
		snapshot["targetItemIndex"] = decision.ItemAnchor.ItemIndex
	}

	execPlan.NavigationDecision = decision
	execPlan.RetrievalPlan.ItemAnchor = copyItemAnchor(decision)
	execPlan.RetrievalPlan.StructureAnchor = copyStructureAnchor(decision)
	execPlan.RetrievalPlan.NavigationAction = decision.NavigationAction
	convCtx.AddUsedTools("route_document_structure")

	ctx = vo.OnEnd(ctx, &vo.StageOutput{SummaryText: "执行路由完成。", Snapshot: snapshot})
	output := &RouteStructureOutput{
		NavigationAction:  decision.NavigationAction,
		RootSectionCode:   decision.StructureAnchor.RootSectionCode,
		RootSectionTitle:  decision.StructureAnchor.RootSectionTitle,
		TargetSectionHint: decision.StructureAnchor.TargetSectionHint,
	}
	if decision.ItemAnchor != nil {
		output.ItemIndex = decision.ItemAnchor.ItemIndex
		output.ItemText = decision.ItemAnchor.ItemText
	}
	return output, nil
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
