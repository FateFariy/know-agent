package executor

import (
	"context"
	"time"

	"github.com/cloudwego/eino/schema"
	"github.com/duke-git/lancet/v2/strutil"
	"github.com/zeromicro/go-zero/core/logx"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/conversation"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/rag"
	ragvo "github.com/swiftbit/know-agent/internal/domain/chat/logic/rag/model/vo"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/trace"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
)

// GraphThenEvidenceExecutor 结构图定位后取证执行器
// 典型场景："第 3.2 节介绍了哪些功能"、"第 5 项的步骤是啥"
// 逻辑：先根据锚点查结构图 → 使用 ragRetriever 以"目标结构 + 问题"为线索检索证据
// → Prompt 装配 + 模型流式回答
type GraphThenEvidenceExecutor struct {
	structureQuerier   rag.StructureGraphQuerier
	ragRetriever       ragvo.RagRetriever
	ragPromptAssembler rag.RagPromptAssembler
	chatModel          logic.ChatModelImpl[*schema.AgenticMessage]
	tracer             *trace.ConversationTraceRecorder
}

// NewGraphThenEvidenceExecutor 构造结构图取证执行器
func NewGraphThenEvidenceExecutor(
	structureQuerier rag.StructureGraphQuerier,
	ragRetriever ragvo.RagRetriever,
	ragPromptAssembler rag.RagPromptAssembler,
	chatModel logic.ChatModelImpl[*schema.AgenticMessage],
	tracer *trace.ConversationTraceRecorder,
) *GraphThenEvidenceExecutor {
	return &GraphThenEvidenceExecutor{
		structureQuerier:   structureQuerier,
		ragRetriever:       ragRetriever,
		ragPromptAssembler: ragPromptAssembler,
		chatModel:          chatModel,
		tracer:             tracer,
	}
}

var _ conversation.Executor = (*GraphThenEvidenceExecutor)(nil)

// Mode 返回 GRAPH_THEN_EVIDENCE
func (e *GraphThenEvidenceExecutor) Mode() vo.ExecutionMode {
	return vo.ExecutionModeGraphThenEvidence
}

// Execute 执行结构图定位 → 证据检索 → 生成回答
func (e *GraphThenEvidenceExecutor) Execute(ctx context.Context, convCtx *vo.ConversationContext) error {
	plan := convCtx.ExecutionPlan.Load()
	if plan == nil {
		return nil
	}

	publishThinking(convCtx, "正在定位目标章节/项，再基于对应内容回答你的问题。")

	// 结构图查询阶段
	graphStage, _ := e.tracer.StartStage(ctx, convCtx.Trace, vo.ConversationTraceStageGraphQuery, e.Mode().String(), "正在查询结构图以定位锚点。", nil)

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
	// 回退：找不到锚点时用最佳匹配补齐
	if sectionNodeId == 0 {
		anchorHint := ""
		if decision != nil && decision.StructureAnchor != nil {
			anchorHint = decision.StructureAnchor.TargetSectionHint
		}
		if sec, err := e.structureQuerier.FindBestSection(ctx, plan.SelectedDocumentId, keyword, anchorHint); err == nil && sec != nil {
			sectionNodeId = sec.NodeId
		}
	}

	graphResult, err := e.structureQuerier.BuildGraphResult(ctx, plan.SelectedDocumentId, sectionNodeId, itemIndex, keyword)
	if err != nil {
		logx.Errorf("结构图查询失败: conversationId=%s error=%v", convCtx.ConversationId, err)
		_ = e.tracer.FailStage(ctx, graphStage, "结构图查询失败。", err, nil)
		publishText(convCtx, utils.BlankToDefault(plan.NoEvidenceReply, defaultNoEvidenceReply))
		return err
	}
	e.tracer.CompleteStage(ctx, graphStage,
		"结构图定位完成。",
		map[string]any{
			"documentId": plan.SelectedDocumentId,
			"nodeId":     sectionNodeId,
			"itemIndex":  utils.PointerOrDefault(itemIndex, 0),
		},
	)

	// 将结构图结果注入计划的 RetrievalNotes，便于检索器使用
	if debugTrace := convCtx.DebugTrace.Load(); debugTrace != nil {
		if graphResult != nil && graphResult.TargetSection != nil {
			debugTrace.RetrievalNotes.Add("已定位到章节：" + utils.BlankToDefault(graphResult.TargetSection.Title, ""))
		}
	}

	// 证据检索阶段
	retrieveStage, _ := e.tracer.StartStage(
		ctx, convCtx.Trace,
		vo.ConversationTraceStageRAGRetrieve,
		e.Mode().String(),
		"正在根据结构图线索检索证据。",
		nil,
	)
	retrievalCtx, err := e.ragRetriever.Retrieve(ctx, plan, convCtx.Trace)
	if err != nil {
		logx.Errorf("RAG 检索失败: conversationId=%s error=%v", convCtx.ConversationId, err)
		_ = e.tracer.FailStage(ctx, retrieveStage, "RAG 检索失败。", err, nil)
		publishText(convCtx, utils.BlankToDefault(plan.NoEvidenceReply, defaultNoEvidenceReply))
		return err
	}
	snapshot := map[string]any{
		"referenceCount":   len(retrievalCtx.FlattenReferences()),
		"usedChannels":     retrievalCtx.GetUsedChannels(),
		"retrievalNotes":   retrievalCtx.GetRetrievalNotes(),
		"subQuestionCount": len(retrievalCtx.SubQuestionEvidenceList),
	}
	_ = e.tracer.CompleteStage(ctx, retrieveStage, "证据检索完成。", snapshot)

	// 注入引用
	uniqueRefs := flattenRagReferencesSnapshot(retrievalCtx.FlattenReferences())
	if len(uniqueRefs) > 0 {
		publishReferences(convCtx, uniqueRefs)
	}
	if retrievalCtx.UsedChannels != nil {
		mergeUsedChannels(convCtx, retrievalCtx.GetUsedChannels())
	}

	if retrievalCtx.IsEmpty() {
		publishThinking(convCtx, "未能在目标章节/项中检索到足够证据，返回兜底回答。")
		publishText(convCtx, utils.BlankToDefault(plan.NoEvidenceReply, defaultNoEvidenceReply))
		return nil
	}

	publishThinking(convCtx, "已整理结构图与检索证据，正在生成回答。")

	// Prompt 装配
	budgetStage, _ := e.tracer.StartStage(ctx, convCtx.Trace, vo.ConversationTraceStageEvidenceBudget, e.Mode().Name(), "正在组装证据与 Prompt。", nil)
	promptResult, err := e.ragPromptAssembler.Assemble(ctx, plan, retrievalCtx)
	if err != nil {
		logx.Errorf("Prompt 组装失败: conversationId=%s err=%v", convCtx.ConversationId, err)
		_ = e.tracer.FailStage(ctx, budgetStage, "Prompt 组装失败。", err, nil)
		publishText(convCtx, utils.BlankToDefault(plan.NoEvidenceReply, defaultNoEvidenceReply))
		return err
	}
	snapshot = map[string]any{
		"renderedReferenceCount": promptResult.RenderedReferenceCount,
		"omittedReferenceCount":  promptResult.OmittedReferenceCount,
	}
	_ = e.tracer.CompleteStage(ctx, budgetStage, "Prompt 组装完成。", snapshot)

	// 模型流式回答
	answerStage, err := e.tracer.StartStage(
		ctx, convCtx.Trace,
		vo.ConversationTraceStageAnswerGenerate,
		e.Mode().Name(),
		"正在基于证据生成回答。",
		nil,
	)
	streamCh, streamErr := e.chatModel.StreamWithTrace(ctx, "graph_then_evidence_answer", promptResult.SystemPrompt, promptResult.UserPrompt, convCtx.Trace)
	if streamErr != nil {
		logx.Errorf("模型流式调用失败: conversationId=%s err=%v", convCtx.ConversationId, streamErr)
		e.tracer.FailStage(ctx, answerStage, "模型流式调用失败。", streamErr, nil)
		publishText(convCtx, utils.BlankToDefault(plan.NoEvidenceReply, defaultNoEvidenceReply))
		return streamErr
	}

	firstRespDone := false
	for chunk := range streamCh {
		select {
		case <-ctx.Done():
			return nil
		default:
			if strutil.IsBlank(chunk) {
				continue
			}
			publishText(convCtx, chunk)
			if !firstRespDone {
				firstRespDone = true
				convCtx.FirstResponseTimeMs.CompareAndSwap(0, time.Since(convCtx.StartTime).Milliseconds())
			}
		}
	}

	snapshot = map[string]any{
		"firstResponseTimeMs": convCtx.FirstResponseTimeMs.Load(),
		"answerLength":        convCtx.AnswerBuffer.Len(),
	}
	_ = e.tracer.CompleteStage(ctx, answerStage, "答案生成完成。", snapshot)
	return nil
}
