package executor

import (
	"context"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/conversation"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/trace"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
	ragvo "github.com/swiftbit/know-agent/internal/domain/rag/model/vo"
)

// GraphOnlyExecutor 结构图直答执行器
// 当问题属于纯目录/章节导航类（如"第 3 章有哪些小节"）时，直接查询结构图
// 并由 GraphAnswerRender 渲染一个纯文本的导航答复
type GraphOnlyExecutor struct {
	structureQuerier ragvo.StructureGraphQuerier
	answerRender     ragvo.GraphAnswerRender
	tracer           *trace.ConversationTraceRecorder
}

// NewGraphOnlyExecutor 构造结构图直答执行器
func NewGraphOnlyExecutor(
	structureQuerier ragvo.StructureGraphQuerier,
	answerRender ragvo.GraphAnswerRender,
	tracer *trace.ConversationTraceRecorder,
) *GraphOnlyExecutor {
	return &GraphOnlyExecutor{
		structureQuerier: structureQuerier,
		answerRender:     answerRender,
		tracer:           tracer,
	}
}

var _ conversation.Executor = (*GraphOnlyExecutor)(nil)

// Mode 返回 GRAPH_ONLY
func (e *GraphOnlyExecutor) Mode() vo.ExecutionMode { return vo.ExecutionModeGraphOnly }

// Execute 执行结构图查询并渲染答案
func (e *GraphOnlyExecutor) Execute(ctx context.Context, convCtx *vo.ConversationContext) error {
	plan := convCtx.ExecutionPlan.Load()
	if plan == nil {
		return nil
	}

	publishThinking(convCtx, "正在定位对应的文档结构信息。")

	graphStage, _ := e.tracer.StartStage(ctx, convCtx.Trace, vo.ConversationTraceStageGraphQuery, e.Mode().String(), "正在查询结构图信息。", nil)

	// 根据导航决策中的锚点查询结构图
	decision := plan.NavigationDecision
	var (
		sectionNodeId int64
		itemIndex     *int
		keyword       = plan.OriginalQuestion
	)
	if decision != nil {
		if decision.StructureAnchor != nil {
			sectionNodeId = decision.StructureAnchor.StructureNodeId
		}
		if decision.ItemAnchor != nil && decision.ItemAnchor.ItemIndex > 0 {
			idx := decision.ItemAnchor.ItemIndex
			itemIndex = &idx
			keyword = decision.ItemAnchor.ItemText
		}
	}
	if sectionNodeId == 0 && decision != nil && decision.StructureAnchor != nil {
		if sec, err := e.structureQuerier.FindBestSection(ctx, plan.SelectedDocumentId, keyword, decision.StructureAnchor.TargetSectionHint); err == nil && sec != nil {
			sectionNodeId = sec.NodeId
		}
	}

	// 构建完整结构图结果（包含父章节、子章节、编号项等）
	graphResult, err := e.structureQuerier.BuildGraphResult(ctx, plan.SelectedDocumentId, sectionNodeId, itemIndex, keyword)
	if err != nil {
		_ = e.tracer.FailStage(ctx, graphStage, "结构图查询失败。", err, nil)
		publishText(convCtx, utils.BlankToDefault(plan.NoEvidenceReply, defaultNoEvidenceReply))
		return err
	}

	snapshot := map[string]any{
		"documentId": plan.SelectedDocumentId,
		"nodeId":     sectionNodeId,
		"itemIndex":  utils.PointerOrDefault(itemIndex, 0),
	}
	_ = e.tracer.CompleteStage(ctx, graphStage, "结构图查询完成。", snapshot)

	answer := e.answerRender.RenderAnswer(e.Mode(), decision, graphResult)
	publishText(convCtx, utils.BlankToDefault(answer, utils.BlankToDefault(plan.NoEvidenceReply, defaultNoEvidenceReply)))
	return nil
}
