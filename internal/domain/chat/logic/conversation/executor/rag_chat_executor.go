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
	vo3 "github.com/swiftbit/know-agent/internal/domain/chat/logic/rag/model/vo"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/trace"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
)

// RagChatExecutor 知识问答执行器
// 流程：双通道混合检索 -> 引用排序 / 预算 / Prompt 装配 -> 模型流式输出
type RagChatExecutor struct {
	ragRetriever       vo3.RagRetriever
	ragPromptAssembler rag.RagPromptAssembler
	chatModel          logic.ChatModelImpl[*schema.AgenticMessage]
	tracer             *trace.ConversationTraceRecorder
}

// NewRagChatExecutor 构造知识问答执行器
func NewRagChatExecutor(
	ragRetriever vo3.RagRetriever,
	ragPromptAssembler rag.RagPromptAssembler,
	chatModel logic.ChatModelImpl[*schema.AgenticMessage],
	tracer *trace.ConversationTraceRecorder,
) *RagChatExecutor {
	return &RagChatExecutor{
		ragRetriever:       ragRetriever,
		ragPromptAssembler: ragPromptAssembler,
		chatModel:          chatModel,
		tracer:             tracer,
	}
}

var _ conversation.Executor = (*RagChatExecutor)(nil)

// Mode 返回执行模式 RETRIEVAL
func (e *RagChatExecutor) Mode() vo.ExecutionMode {
	return vo.ExecutionModeRetrieval
}

// Execute 执行检索 + Prompt 装配 + 模型流式回答
func (e *RagChatExecutor) Execute(ctx context.Context, convCtx *vo.ConversationContext) error {
	plan := convCtx.ExecutionPlan.Load()
	if plan == nil {
		return nil
	}

	publishThinking(convCtx, "正在根据问题规划知识检索范围。")

	retrieveStage, err := e.tracer.StartStage(ctx, convCtx.Trace, vo.ConversationTraceStageRAGRetrieve, e.Mode().String(), "正在执行双通道混合检索。", nil)

	retrievalCtx, err := e.ragRetriever.Retrieve(ctx, plan, convCtx.Trace)
	if err != nil {
		logx.Errorf("RAG 检索失败: conversationId=%s, error=%v", convCtx.ConversationId, err)
		if err := e.tracer.FailStage(ctx, retrieveStage, "RAG 检索失败。", err, nil); err != nil {
			logx.Errorf("保存阶段信息失败: conversationId=%s, error=%v", convCtx.ConversationId, err)
		}
		publishText(convCtx, utils.BlankToDefault(plan.NoEvidenceReply, defaultNoEvidenceReply))
		return err
	}

	if retrievalCtx != nil {
		subQuestions := make([]map[string]any, len(retrievalCtx.SubQuestionEvidenceList))
		for i, sq := range retrievalCtx.SubQuestionEvidenceList {
			subQuestions[i] = map[string]any{
				"index":                  sq.SubQuestionIndex,
				"question":               sq.SubQuestion,
				"referenceCount":         len(sq.References),
				"documentCount":          len(sq.Documents),
				"fusedCandidateCount":    sq.FusedCandidateCount,
				"parentCandidateCount":   sq.ParentCandidateCount,
				"rerankedCandidateCount": sq.RerankedCandidateCount,
				"channelTraces":          sq.GetChannelTraceMaps(),
				"references":             sq.GetReferenceMaps(),
			}
		}

		references := retrievalCtx.FlattenReferences()
		refDetails := make([]map[string]any, len(references))
		for i, ref := range references {
			refDetails[i] = map[string]any{
				"referenceId":  ref.ReferenceId,
				"documentName": utils.BlankToDefault(ref.DocumentName, ref.Title),
				"sectionPath":  ref.SectionPath,
				"channel":      ref.Channel,
			}
		}

		snapshot := map[string]any{
			"retrievalQuestion": retrievalCtx.RetrievalQuestion,
			"usedChannels":      retrievalCtx.GetUsedChannels(),
			"retrievalNotes":    retrievalCtx.GetRetrievalNotes(),
			"referenceCount":    len(references),
			"subQuestionCount":  len(retrievalCtx.SubQuestionEvidenceList),
			"subQuestions":      subQuestions,
			"references":        refDetails,
		}
		if err := e.tracer.CompleteStage(ctx, retrieveStage, "RAG 检索完成。", snapshot); err != nil {
			logx.Errorf("保存阶段信息失败: conversationId=%s, error=%v", convCtx.ConversationId, err)
		}
	}
	e.streamFromRetrievalContext(ctx, convCtx, plan, retrievalCtx)
	return nil
}

// streamFromRetrievalContext 基于检索上下文生成流式回答
func (e *RagChatExecutor) streamFromRetrievalContext(ctx context.Context, convCtx *vo.ConversationContext,
	plan *vo.ConversationExecutionPlan, retrievalCtx *vo3.RagRetrievalContext) {
	// 先下发思考事件（检索笔记、渠道列表）
	if retrievalCtx != nil {
		retrievalCtx.RetrievalNotes.ForEach(func(note string) {
			publishThinking(convCtx, note)
		})
	}

	// 合并渠道记录到上下文与调试轨迹
	if retrievalCtx != nil {
		chs := retrievalCtx.GetUsedChannels()
		mergeUsedChannels(convCtx, chs)
	}

	// 空证据兜底
	if retrievalCtx == nil || retrievalCtx.IsEmpty() {
		publishThinking(convCtx, "当前没有足够证据，直接返回无证据兜底回复。")
		publishText(convCtx, utils.BlankToDefault(plan.NoEvidenceReply, defaultNoEvidenceReply))
		return
	}

	uniqueRefs := flattenRagReferencesSnapshot(retrievalCtx.FlattenReferences())
	if len(uniqueRefs) > 0 {
		publishReferences(convCtx, uniqueRefs)
	}

	publishThinking(convCtx, "证据整理完成，正在基于证据生成回答。")

	// Prompt 装配与预算
	budgetStage, err := e.tracer.StartStage(ctx, convCtx.Trace, vo.ConversationTraceStageEvidenceBudget,
		e.Mode().String(), "正在组装证据与 Prompt 预算。", nil)
	promptResult, err := e.ragPromptAssembler.Assemble(ctx, plan, retrievalCtx)
	if err != nil {
		logx.Errorf("Prompt 组装失败: conversationId=%s, err=%v", convCtx.ConversationId, err)
		e.tracer.FailStage(ctx, budgetStage, "证据预算与 Prompt 组装失败。", err, nil)
		publishText(convCtx, utils.BlankToDefault(plan.NoEvidenceReply, defaultNoEvidenceReply))
		return
	}
	e.tracer.CompleteStage(ctx, budgetStage, "证据预算与 Prompt 组装完成。", map[string]any{
		"totalBudget":              promptResult.TotalBudget,
		"perSubQuestionBudget":     promptResult.PerSubQuestionBudget,
		"renderedReferenceCount":   promptResult.RenderedReferenceCount,
		"omittedReferenceCount":    promptResult.OmittedReferenceCount,
		"renderedReferenceDetails": promptResult.RenderedReferenceDetails,
		"omittedReferenceDetails":  promptResult.OmittedReferenceDetails,
		"systemPrompt":             promptResult.SystemPrompt,
		"userPrompt":               promptResult.UserPrompt,
	})

	answerStage, err := e.tracer.StartStage(ctx, convCtx.Trace, vo.ConversationTraceStageAnswerGenerate, e.Mode().String(), "正在基于证据生成回答。", nil)

	streamCh, streamErr := e.chatModel.StreamWithTrace(ctx, "rag_answer", promptResult.SystemPrompt, promptResult.UserPrompt, convCtx.Trace)
	if streamErr != nil {
		logx.Errorf("模型流式调用失败: conversationId=%s, error=%v", convCtx.ConversationId, streamErr)
		e.tracer.FailStage(ctx, answerStage, "答案生成失败。", streamErr, nil)
		publishText(convCtx, utils.BlankToDefault(plan.NoEvidenceReply, defaultNoEvidenceReply))
		return
	}

	firstRespDone := false
	for chunk := range streamCh {
		select {
		case <-ctx.Done():
			return
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

	e.tracer.CompleteStage(ctx, answerStage, "答案生成完成。", map[string]any{
		"firstResponseTimeMs": convCtx.FirstResponseTimeMs.Load(),
		"answerLength":        convCtx.AnswerBuffer.Len(),
	})
}
