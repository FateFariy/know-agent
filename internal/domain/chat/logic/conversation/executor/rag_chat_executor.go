package executor

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/schema"
	"github.com/zeromicro/go-zero/core/logx"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/conversation"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/trace"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
)

// RagChatExecutor 知识问答执行器
// 流程：双通道混合检索 -> 引用排序 / 预算 / Prompt 装配 -> 模型流式输出
type RagChatExecutor struct {
	retriever       logic.RagRetriever
	promptAssembler conversation.RagPromptAssembler
	chatModel       logic.ChatModelImpl[*schema.AgenticMessage]
	tracer          *trace.ConversationTraceRecorder
}

// NewRagChatExecutor 构造知识问答执行器
func NewRagChatExecutor(
	retriever logic.RagRetriever,
	ragPromptAssembler conversation.RagPromptAssembler,
	chatModel logic.ChatModelImpl[*schema.AgenticMessage],
	tracer *trace.ConversationTraceRecorder,
) *RagChatExecutor {
	return &RagChatExecutor{
		retriever:       retriever,
		promptAssembler: ragPromptAssembler,
		chatModel:       chatModel,
		tracer:          tracer,
	}
}

var _ conversation.Executor = (*RagChatExecutor)(nil)

// Mode 返回执行模式 RETRIEVAL
func (e *RagChatExecutor) Mode() vo.ExecutionMode {
	return vo.ExecutionModeRetrieval
}

// Execute 执行检索 + Prompt 装配 + 模型流式回答
func (e *RagChatExecutor) Execute(ctx context.Context, convCtx *vo.ConversationContext) (<-chan string, error) {
	// 加载执行计划，缺失时直接报错
	plan := convCtx.ExecutionPlan.Load()
	if plan == nil {
		return nil, fmt.Errorf("invalid value")
	}

	if err := publishThinking(convCtx, "正在根据问题规划知识检索范围。"); err != nil {
		return nil, err
	}

	retrieveStage, _ := e.tracer.StartStage(ctx, convCtx.Trace, vo.ConversationTraceStageRAGRetrieve,
		e.Mode().String(), "正在执行双通道混合检索。", nil)

	retrievalCtx, err := e.retriever.Retrieve(ctx, plan, convCtx.Trace)
	if err != nil {
		logx.Errorf("RAG 检索失败: conversationId=%s, error=%v", convCtx.ConversationId, err)
		_ = e.tracer.FailStage(ctx, retrieveStage, "RAG 检索失败。", err, nil)
		return nil, err
	}

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
		"usedChannels":      retrievalCtx.UsedChannels(),
		"retrievalNotes":    retrievalCtx.RetrievalNotes(),
		"referenceCount":    len(references),
		"subQuestionCount":  len(retrievalCtx.SubQuestionEvidenceList),
		"subQuestions":      subQuestions,
		"references":        refDetails,
	}
	_ = e.tracer.CompleteStage(ctx, retrieveStage, "RAG 检索完成。", snapshot)

	return e.streamFromRetrievalContext(ctx, convCtx, plan, retrievalCtx)
}

// streamFromRetrievalContext 基于检索上下文生成流式回答
func (e *RagChatExecutor) streamFromRetrievalContext(ctx context.Context, convCtx *vo.ConversationContext,
	plan *vo.ConversationExecutionPlan, retrievalCtx *vo.RagRetrievalContext) (<-chan string, error) {
	// 先下发思考事件（检索笔记、渠道列表）
	notes := retrievalCtx.RetrievalNotes()
	for _, note := range notes {
		if err := publishThinking(convCtx, note); err != nil {
			return nil, err
		}
	}

	// 合并渠道记录到上下文与调试轨迹
	chs := retrievalCtx.UsedChannels()
	convCtx.AddUsedTools(chs...)
	if debugTrace := convCtx.DebugTrace.Load(); debugTrace != nil {
		debugTrace.SetUsedChannels(chs...)
		debugTrace.SetRetrievalNotes(retrievalCtx.RetrievalNotes()...)
	}

	// 空证据兜底
	if retrievalCtx.IsEmpty() {
		if err := publishThinking(convCtx, "当前没有足够证据，直接返回无证据兜底回复。"); err != nil {
			return nil, err
		}
		return singleValueChan(utils.BlankToDefault(plan.NoEvidenceReply, defaultNoEvidenceReply)), nil
	}

	references := retrievalCtx.FlattenReferences()
	if len(references) > 0 {
		if err := publishReferences(convCtx, references); err != nil {
			return nil, err
		}
	}

	if err := publishThinking(convCtx, "证据整理完成，正在基于证据生成回答。"); err != nil {
		return nil, err
	}

	// Prompt 装配与预算
	budgetStage, err := e.tracer.StartStage(ctx, convCtx.Trace, vo.ConversationTraceStageEvidenceBudget,
		e.Mode().String(), "正在组装证据与 Prompt 预算。", nil)
	promptResult, err := e.promptAssembler.Assemble(ctx, plan, retrievalCtx)
	if err != nil {
		logx.Errorf("Prompt 组装失败: conversationId=%s, err=%v", convCtx.ConversationId, err)
		_ = e.tracer.FailStage(ctx, budgetStage, "证据预算与 Prompt 组装失败。", err, nil)
		return nil, err
	}

	snapshot := map[string]any{
		"totalBudget":              promptResult.TotalBudget,
		"perSubQuestionBudget":     promptResult.PerSubQuestionBudget,
		"renderedReferenceCount":   promptResult.RenderedReferenceCount,
		"omittedReferenceCount":    promptResult.OmittedReferenceCount,
		"renderedReferenceDetails": promptResult.RenderedReferenceDetails,
		"omittedReferenceDetails":  promptResult.OmittedReferenceDetails,
		"systemPrompt":             promptResult.SystemPrompt,
		"userPrompt":               promptResult.UserPrompt,
	}
	_ = e.tracer.CompleteStage(ctx, budgetStage, "证据预算与 Prompt 组装完成。", snapshot)

	answerStage, err := e.tracer.StartStage(ctx, convCtx.Trace, vo.ConversationTraceStageAnswerGenerate, e.Mode().String(), "正在基于证据生成回答。", nil)

	streamCh, err := e.chatModel.StreamWithTrace(ctx, vo.ChatStageRagAnswer, promptResult.SystemPrompt, promptResult.UserPrompt, convCtx.Trace)
	if err != nil {
		logx.Errorf("模型流式调用失败: conversationId=%s, error=%v", convCtx.ConversationId, err)
		_ = e.tracer.FailStage(ctx, answerStage, "答案生成失败。", err, nil)
		return nil, err
	}
	// todo 完成阶段由调用方记录
	// e.tracer.CompleteStage(ctx, answerStage, "答案生成完成。", map[string]any{
	// 	"firstResponseTimeMs": convCtx.FirstResponseTimeMs.Load(),
	// 	"answerLength":        convCtx.AnswerLength(),
	// })
	return streamCh, nil
}
