package executor

import (
	"context"

	"github.com/duke-git/lancet/v2/strutil"
	"github.com/zeromicro/go-zero/core/logx"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/conversation"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/graph"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/trace"
	ragvo "github.com/swiftbit/know-agent/internal/domain/chat/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
)

// GraphOnlyExecutor 结构图直答执行器
//
// 当问题属于纯目录/章节导航类（如 "第 3 章有哪些小节"、"3.2 的上一节是什么"）时，
// 仅通过结构图查询（父章节 / 兄弟章节 / 子章节），再由 AnswerRender 渲染一个纯文本的导航答复。
type GraphOnlyExecutor struct {
	structureQuerier logic.StructureGraphQuerier
	answerRender     graph.AnswerRender
	tracer           *trace.ConversationTraceRecorder
}

// NewGraphOnlyExecutor 构造结构图直答执行器
func NewGraphOnlyExecutor(
	structureQuerier logic.StructureGraphQuerier,
	answerRender graph.AnswerRender,
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

// Execute 执行结构图查询并渲染答案（与 Java GraphOnlyExecutor 对齐）
func (e *GraphOnlyExecutor) Execute(ctx context.Context, convCtx *vo.ConversationContext) (<-chan string, error) {
	plan := convCtx.ExecutionPlan.Load()
	if plan == nil {
		return nil, nil
	}

	decision := plan.NavigationDecision
	if decision == nil || decision.StructureAnchor == nil || decision.StructureAnchor.StructureNodeId == 0 {
		logx.Infof("GRAPH_ONLY 执行器直接返回无证据: decisionPresent=%v, structureNodeId=%v",
			decision != nil,
			safeStructureNodeId(decision))
		publishText(convCtx, utils.BlankToDefault(plan.NoEvidenceReply, defaultNoEvidenceReply))
		return nil, nil
	}

	publishThinking(convCtx, "正在通过结构图直接查询章节关系。")

	graphStage, _ := e.tracer.StartStage(ctx, convCtx.Trace,
		vo.ConversationTraceStageGraphQuery, e.Mode().String(), "正在执行结构图查询。", nil)

	documentId := plan.SelectedDocumentId
	sectionNodeId := decision.StructureAnchor.StructureNodeId

	logx.Infof("GRAPH_ONLY 执行开始: documentId=%v, sectionNodeId=%v, action=%v, navigationSummary=%q",
		documentId, sectionNodeId, decision.NavigationAction, decision.SummaryText)

	graphResult, err := e.buildGraphResult(ctx, documentId, sectionNodeId, decision)
	if err != nil {
		_ = e.tracer.FailStage(ctx, graphStage, "结构图查询失败。", err, nil)
		publishText(convCtx, utils.BlankToDefault(plan.NoEvidenceReply, defaultNoEvidenceReply))
		return nil, err
	}

	answer := e.answerRender.RenderAnswer(e.Mode(), decision, graphResult)

	logx.Infof("GRAPH_ONLY 执行完成: documentId=%v, sectionNodeId=%v, targetSection=%q, answerLength=%v",
		documentId, sectionNodeId, sectionDisplayTitle(graphResult.TargetSection), len(answer))

	snapshot := map[string]any{
		"targetSection":   sectionDisplayTitle(graphResult.TargetSection),
		"parentSection":   sectionDisplayTitle(graphResult.ParentSection),
		"childCount":      len(graphResult.Children),
		"previousSibling": sectionDisplayTitle(graphResult.PreviousSibling),
		"nextSibling":     sectionDisplayTitle(graphResult.NextSibling),
		"answer":          strutil.Trim(answer),
	}
	_ = e.tracer.CompleteStage(ctx, graphStage, "结构图查询完成。", snapshot)

	if strutil.IsBlank(answer) {
		publishText(convCtx, utils.BlankToDefault(plan.NoEvidenceReply, defaultNoEvidenceReply))
		return nil, nil
	}
	publishText(convCtx, answer)
	return nil, nil
}

// buildGraphResult 根据导航动作构建结构图查询结果
//
// - SECTION_ADJACENCY_LOOKUP：查询目标章节的前后兄弟章节 + 父章节
// - 其他（默认 CHILD_SECTION_DESCEND）：查询目标章节 + 直接下级章节列表
func (e *GraphOnlyExecutor) buildGraphResult(
	ctx context.Context, documentId, sectionNodeId int64, decision *vo.DocumentNavigationDecision,
) (*ragvo.GraphQueryResult, error) {
	if decision != nil && decision.NavigationAction == vo.DocumentNavigationActionSectionAdjacencyLookup {
		siblings, err := e.structureQuerier.FindSectionWithSiblings(ctx, documentId, sectionNodeId)
		if err != nil {
			return nil, err
		}
		if siblings == nil {
			return &ragvo.GraphQueryResult{}, nil
		}
		return &ragvo.GraphQueryResult{
			TargetSection:   siblings.Section,
			ParentSection:   siblings.Parent,
			PreviousSibling: siblings.PreviousSibling,
			NextSibling:     siblings.NextSibling,
		}, nil
	}

	children, err := e.structureQuerier.FindSectionWithChildren(ctx, documentId, sectionNodeId)
	if err != nil {
		return nil, err
	}
	if children == nil {
		return &ragvo.GraphQueryResult{}, nil
	}
	return &ragvo.GraphQueryResult{
		TargetSection: children.Section,
		Children:      children.Children,
	}, nil
}

// sectionDisplayTitle 取章节的展示标题
func sectionDisplayTitle(section *ragvo.GraphSection) string {
	if section == nil {
		return ""
	}
	return section.DisplayTitle()
}
