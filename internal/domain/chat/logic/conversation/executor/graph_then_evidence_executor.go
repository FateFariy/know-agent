package executor

import (
	"context"
	"fmt"
	"strings"

	"github.com/duke-git/lancet/v2/strutil"
	"github.com/zeromicro/go-zero/core/logx"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/conversation"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/graph"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/trace"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
)

// GraphThenEvidenceExecutor 先结构图定位、再读取章节/编号项证据的执行器
//
// 适用场景：用户明确指向某个章节或编号项（如"第 3 节第 2 项讲的是什么"），
// 先由结构图定位目标节点，再由 AnswerRender 把节点文本/编号项渲染成最终答复。
type GraphThenEvidenceExecutor struct {
	structureQuerier logic.StructureGraphQuerier
	answerRender     graph.AnswerRender
	tracer           *trace.ConversationTraceRecorder
}

// NewGraphThenEvidenceExecutor 构造"结构图定位后取证"执行器
func NewGraphThenEvidenceExecutor(
	structureQuerier logic.StructureGraphQuerier,
	answerRender graph.AnswerRender,
	tracer *trace.ConversationTraceRecorder,
) *GraphThenEvidenceExecutor {
	return &GraphThenEvidenceExecutor{
		structureQuerier: structureQuerier,
		answerRender:     answerRender,
		tracer:           tracer,
	}
}

var _ conversation.Executor = (*GraphThenEvidenceExecutor)(nil)

// Mode 返回 GRAPH_THEN_EVIDENCE
func (e *GraphThenEvidenceExecutor) Mode() vo.ExecutionMode {
	return vo.ExecutionModeGraphThenEvidence
}

// Execute 执行结构图定位与证据渲染
func (e *GraphThenEvidenceExecutor) Execute(ctx context.Context, convCtx *vo.ConversationContext) (<-chan string, error) {
	plan := convCtx.ExecutionPlan.Load()
	var decision *vo.DocumentNavigationDecision
	noEvidenceReply := defaultNoEvidenceReply
	if plan != nil {
		decision = plan.NavigationDecision
		noEvidenceReply = utils.BlankToDefault(plan.NoEvidenceReply, noEvidenceReply)
	}
	if decision == nil || decision.StructureAnchor == nil || decision.StructureAnchor.StructureNodeId == 0 {
		logx.Infof("GRAPH_THEN_EVIDENCE 执行器直接返回无证据: planPresent=%v,decisionPresent=%v, structureNodeId=%v",
			plan != nil, decision != nil, safeStructureNodeId(decision))
		return singleValueChan(noEvidenceReply), nil
	}

	if err := publishThinking(convCtx, "正在通过结构图定位目标章节和编号项。"); err != nil {
		return nil, err
	}

	graphStage, _ := e.tracer.StartStage(ctx, convCtx.Trace,
		vo.ConversationTraceStageGraphQuery, e.Mode().String(), "正在执行结构图定位与取证。", nil)

	documentId := plan.SelectedDocumentId
	sectionNodeId := decision.StructureAnchor.StructureNodeId
	itemIndex := 0
	itemKeywordHint := extractItemKeyword(plan.OriginalQuestion)

	if decision.ItemAnchor != nil {
		itemIndex = decision.ItemAnchor.ItemIndex
	}

	logx.Infof("GRAPH_THEN_EVIDENCE 执行开始: documentId=%v, sectionNodeId=%v, itemIndex=%v，navigationSummary=%v",
		documentId, sectionNodeId, itemIndex, decision.SummaryText)

	graphResult, err := e.structureQuerier.BuildGraphResult(ctx, documentId, sectionNodeId, itemIndex, itemKeywordHint)
	if err != nil {
		_ = e.tracer.FailStage(ctx, graphStage, "结构图查询失败。", err, nil)
		if err := publishThinking(convCtx, "结构图查询失败。"); err != nil {
			return nil, err
		}
		return singleValueChan(noEvidenceReply), nil
	}

	if !hasGraphEvidence(graphResult, decision) {
		snapshot := map[string]any{
			"targetSection":   displayTitleOf(graphResult),
			"targetItemIndex": targetItemIndexOf(graphResult),
			"notes":           []string{"结构图未定位到满足条件的章节或编号项。"},
		}
		_ = e.tracer.CompleteStage(ctx, graphStage, "结构图定位完成，但证据不满足约束。", snapshot)
		logx.Infof("GRAPH_THEN_EVIDENCE 证据校验失败: documentId=%v, sectionNodeId=%v", documentId, sectionNodeId)
		return singleValueChan(noEvidenceReply), nil
	}

	answer := e.answerRender.RenderAnswer(e.Mode(), decision, graphResult)

	snapshot := map[string]any{
		"targetSection":    displayTitleOf(graphResult),
		"targetItemIndex":  targetItemIndexOf(graphResult),
		"matchedItemCount": matchedItemCountOf(graphResult),
		"answer":           strutil.Trim(answer),
	}
	_ = e.tracer.CompleteStage(ctx, graphStage, "结构图取证完成。", snapshot)

	logx.Infof("GRAPH_THEN_EVIDENCE 执行完成: documentId=%v, sectionNodeId=%v, targetSection=%q, targetItemIndex=%v, answerLength=%v",
		documentId, sectionNodeId, displayTitleOf(graphResult), targetItemIndexOf(graphResult), len(answer))

	return singleValueChan(utils.BlankToDefault(answer, noEvidenceReply)), nil
}

// hasGraphEvidence 判断结构图结果是否满足证据要求
func hasGraphEvidence(result *entity.GraphQueryResult, decision *vo.DocumentNavigationDecision) bool {
	if result == nil || result.TargetSection == nil {
		return false
	}
	if decision != nil && decision.ItemAnchor != nil && decision.ItemAnchor.ItemIndex > 0 {
		return result.TargetItem != nil || len(result.MatchedItems) > 0
	}
	return strutil.IsNotBlank(result.TargetSection.ContentText) || len(result.MatchedItems) > 0
}

// extractItemKeyword 从原问题文本中抽取用于编号项匹配的关键词
//
// - 仅当问题包含 "哪一步" / "哪一项" 时处理
// - 取对应前缀后的片段，并剔除常见的疑问/语气词
func extractItemKeyword(question string) string {
	normalized := strutil.Trim(question)
	if !strutil.ContainsAny(normalized, []string{"哪一步", "哪一项"}) {
		return ""
	}
	keyword := strutil.AfterLast(normalized, "哪一步")
	if keyword == "" {
		keyword = strutil.AfterLast(normalized, "哪一项")
	}

	replacer := strings.NewReplacer("要求", "", "需要", "", "执行", "", "进行", "", "包含", "", "的是", "", "是什么", "",
		"什么", "", "？", "", "?", "", "。", "", "，", "")
	keyword = replacer.Replace(keyword)

	return strutil.Trim(keyword)
}

// safeStructureNodeId 返回结构图定位的章节ID，若缺失则返回 0
func safeStructureNodeId(decision *vo.DocumentNavigationDecision) int64 {
	if decision == nil || decision.StructureAnchor == nil {
		return 0
	}
	return decision.StructureAnchor.StructureNodeId
}

// displayTitleOf 返回目标章节的展示标题
func displayTitleOf(result *entity.GraphQueryResult) string {
	if result == nil || result.TargetSection == nil {
		return ""
	}
	return result.TargetSection.DisplayTitle()
}

// targetItemIndexOf 返回目标编号项的索引
func targetItemIndexOf(result *entity.GraphQueryResult) string {
	if result == nil || result.TargetItem == nil || result.TargetItem.ItemIndex <= 0 {
		return ""
	}
	return fmt.Sprintf("%d", result.TargetItem.ItemIndex)
}

// matchedItemCountOf 返回命中的编号项数量
func matchedItemCountOf(result *entity.GraphQueryResult) int {
	if result == nil || len(result.MatchedItems) == 0 {
		return 0
	}
	return len(result.MatchedItems)
}
