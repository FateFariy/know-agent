package executor

import (
	"context"
	"fmt"
	"strings"

	"github.com/duke-git/lancet/v2/strutil"
	"github.com/zeromicro/go-zero/core/logx"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/conversation"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/graph"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/trace"
	ragvo "github.com/swiftbit/know-agent/internal/domain/chat/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
)

// GraphThenEvidenceExecutor 先结构图定位、再读取章节/编号项证据的执行器
//
// 适用场景：用户明确指向某个章节或编号项（如"第 3 节第 2 项讲的是什么"），
// 先由结构图定位目标节点，再由 AnswerRender 把节点文本/编号项渲染成最终答复。
type GraphThenEvidenceExecutor struct {
	structureQuerier graph.StructureGraphQuerier
	answerRender     graph.AnswerRender
	tracer           *trace.ConversationTraceRecorder
}

// NewGraphThenEvidenceExecutor 构造"结构图定位后取证"执行器
func NewGraphThenEvidenceExecutor(
	structureQuerier graph.StructureGraphQuerier,
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
	if plan == nil {
		return nil, nil
	}
	decision := plan.NavigationDecision
	if decision == nil || decision.StructureAnchor == nil || decision.StructureAnchor.StructureNodeId == 0 {
		logx.Infof("GRAPH_THEN_EVIDENCE 执行器直接返回无证据: decisionPresent=%v, structureNodeId=%v",
			decision != nil,
			safeStructureNodeId(decision))
		publishText(convCtx, utils.BlankToDefault(plan.NoEvidenceReply, defaultNoEvidenceReply))
		return nil, nil
	}

	publishThinking(convCtx, "正在通过结构图定位目标章节和编号项。")

	graphStage, _ := e.tracer.StartStage(ctx, convCtx.Trace,
		vo.ConversationTraceStageGraphQuery, e.Mode().String(), "正在执行结构图定位与取证。", nil)

	var (
		documentId      = plan.SelectedDocumentId
		sectionNodeId   = decision.StructureAnchor.StructureNodeId
		itemIndex       *int
		itemKeywordHint = extractItemKeyword(plan.OriginalQuestion, decision)
	)
	if decision.ItemAnchor != nil && decision.ItemAnchor.ItemIndex > 0 {
		idx := decision.ItemAnchor.ItemIndex
		itemIndex = &idx
	}

	logx.Infof("GRAPH_THEN_EVIDENCE 执行开始: documentId=%v, sectionNodeId=%v, itemIndex=%v",
		documentId, sectionNodeId, utils.PointerOrDefault(itemIndex, 0))

	graphResult, err := e.structureQuerier.BuildGraphResult(ctx, documentId, sectionNodeId, itemIndex, itemKeywordHint)
	if err != nil {
		_ = e.tracer.FailStage(ctx, graphStage, "结构图查询失败。", err, nil)
		publishText(convCtx, utils.BlankToDefault(plan.NoEvidenceReply, defaultNoEvidenceReply))
		return nil, err
	}

	if !hasGraphEvidence(graphResult, decision) {
		snapshot := map[string]any{
			"targetSection":   displayTitleOf(graphResult),
			"targetItemIndex": targetItemIndexOf(graphResult),
			"notes":           []string{"结构图未定位到满足条件的章节或编号项。"},
		}
		_ = e.tracer.CompleteStage(ctx, graphStage, "结构图定位完成，但证据不满足约束。", snapshot)
		logx.Infof("GRAPH_THEN_EVIDENCE 证据校验失败: documentId=%v, sectionNodeId=%v", documentId, sectionNodeId)
		publishText(convCtx, utils.BlankToDefault(plan.NoEvidenceReply, defaultNoEvidenceReply))
		return nil, nil
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

	if strutil.IsBlank(answer) {
		publishText(convCtx, utils.BlankToDefault(plan.NoEvidenceReply, defaultNoEvidenceReply))
		return nil, nil
	}
	publishText(convCtx, answer)
	return nil, nil
}

// ==================== 辅助函数 ====================

// safeStructureNodeId 当 decision/anchor 缺失时返回 0，供日志打印使用
func safeStructureNodeId(decision *vo.DocumentNavigationDecision) int64 {
	if decision == nil || decision.StructureAnchor == nil {
		return 0
	}
	return decision.StructureAnchor.StructureNodeId
}

// displayTitleOf 返回目标章节的展示标题；与 Java GraphSection#displayTitle 对齐
func displayTitleOf(result *ragvo.GraphQueryResult) string {
	if result == nil || result.TargetSection == nil {
		return ""
	}
	return result.TargetSection.DisplayTitle()
}

// targetItemIndexOf 返回目标编号项的索引（以字符串返回，便于日志/快照）
func targetItemIndexOf(result *ragvo.GraphQueryResult) string {
	if result == nil || result.TargetItem == nil || result.TargetItem.ItemIndex == nil {
		return ""
	}
	return fmt.Sprintf("%d", *result.TargetItem.ItemIndex)
}

// matchedItemCountOf 返回命中的编号项数量
func matchedItemCountOf(result *ragvo.GraphQueryResult) int {
	if result == nil || len(result.MatchedItems) == 0 {
		return 0
	}
	return len(result.MatchedItems)
}

// hasGraphEvidence 判断结构图结果是否满足证据要求（与 Java 对齐）
//
// - 若决策中指定了 itemIndex，则必须存在 targetItem 或 matchedItems
// - 否则只需要 targetSection 有内容文本 或 存在 matchedItems
func hasGraphEvidence(result *ragvo.GraphQueryResult, decision *vo.DocumentNavigationDecision) bool {
	if result == nil || result.TargetSection == nil {
		return false
	}
	if decision != nil && decision.ItemAnchor != nil && decision.ItemAnchor.ItemIndex > 0 {
		if result.TargetItem != nil {
			return true
		}
		return len(result.MatchedItems) > 0
	}
	if strutil.IsNotBlank(result.TargetSection.ContentText) {
		return true
	}
	return len(result.MatchedItems) > 0
}

// extractItemKeyword 从原问题文本中抽取用于编号项匹配的关键词
//
// - 仅当问题包含 "哪一步" / "哪一项" 时处理
// - 取对应前缀后的片段，并剔除常见的疑问/语气词
func extractItemKeyword(question string, decision *vo.DocumentNavigationDecision) string {
	normalized := strings.TrimSpace(question)
	if !strings.Contains(normalized, "哪一步") && !strings.Contains(normalized, "哪一项") {
		return ""
	}

	keyword := ""
	switch {
	case strings.Contains(normalized, "哪一步"):
		if idx := strings.Index(normalized, "哪一步"); idx >= 0 {
			keyword = normalized[idx+len("哪一步"):]
		}
	case strings.Contains(normalized, "哪一项"):
		if idx := strings.Index(normalized, "哪一项"); idx >= 0 {
			keyword = normalized[idx+len("哪一项"):]
		}
	}
	keyword = strings.NewReplacer(
		"要求", "",
		"需要", "",
		"执行", "",
		"进行", "",
		"包含", "",
		"的是", "",
		"是什么", "",
		"什么", "",
		"？", "",
		"?", "",
		"。", "",
		"，", "",
	).Replace(keyword)
	keyword = strings.TrimSpace(keyword)
	if keyword == "" && decision != nil && decision.ItemAnchor != nil {
		keyword = decision.ItemAnchor.ItemText
	}
	return keyword
}
