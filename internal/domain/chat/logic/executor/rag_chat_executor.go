package executor

import (
	"context"
	"fmt"

	"github.com/swiftbit/know-agent/common/logx"
	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/chat/adapter/model"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/conversation"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/rag"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/enum"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
)

// RagChatExecutor 知识问答执行器
// 流程：双通道混合检索 -> 引用排序 / 预算 / Prompt 装配 -> 模型流式输出
type RagChatExecutor struct {
	retriever       rag.Retriever
	promptAssembler RagPromptAssembler
	chatModel       model.ChatModel
}

// NewRagChatExecutor 构造知识问答执行器
func NewRagChatExecutor(retriever rag.Retriever, ragPromptAssembler RagPromptAssembler, chatModel model.ChatModel) *RagChatExecutor {
	return &RagChatExecutor{
		retriever:       retriever,
		promptAssembler: ragPromptAssembler,
		chatModel:       chatModel,
	}
}

var _ Executor = (*RagChatExecutor)(nil)

// Mode 返回执行模式 RETRIEVAL
func (e *RagChatExecutor) Mode() enum.ExecutionMode {
	return enum.ExecutionModeRetrieval
}

// Execute 执行检索 + Prompt 装配 + 模型流式回答
func (e *RagChatExecutor) Execute(ctx context.Context, convCtx *conversation.Context) (<-chan string, error) {
	// 加载执行计划，缺失时直接报错
	plan := convCtx.ExecutionPlan.Load()
	if plan == nil {
		return nil, fmt.Errorf("invalid value")
	}

	if err := convCtx.PublishThinking("正在根据问题规划知识检索范围。"); err != nil {
		return nil, err
	}

	ctx = vo.OnStart(ctx, enum.ConversationTraceStageRAGRetrieve,
		e.Mode().String(), &vo.StageInput{SummaryText: "正在执行双通道混合检索。"})

	retrievalCtx, err := e.retriever.Retrieve(ctx, plan)
	if err != nil {
		logx.Errorf("RAG 检索失败: conversationId=%s, error=%v", convCtx.ConversationId, err)
		vo.OnError(ctx, "RAG 检索失败。", err)
		return nil, err
	}

	_ = vo.OnEnd(ctx, &vo.StageOutput{
		SummaryText: "RAG 检索完成。",
		Snapshot:    retrievalCtx.ToSnapshot(plan),
	})

	return e.streamFromRetrievalContext(ctx, convCtx, plan, retrievalCtx)
}

// streamFromRetrievalContext 基于检索上下文生成流式回答
func (e *RagChatExecutor) streamFromRetrievalContext(ctx context.Context, convCtx *conversation.Context,
	plan *vo.ConversationExecutionPlan, retrievalCtx *vo.RetrievalResult) (<-chan string, error) {
	// 先下发思考事件（检索笔记、渠道列表）
	notes := retrievalCtx.RetrievalNotes()
	for _, note := range notes {
		if err := convCtx.PublishThinking(note); err != nil {
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
		if err := convCtx.PublishThinking("当前没有足够证据，直接返回无证据兜底回复。"); err != nil {
			return nil, err
		}
		return singleValueChan(utils.BlankToDefault(plan.NoEvidenceReply, defaultNoEvidenceReply)), nil
	}

	references := retrievalCtx.FlattenReferences()
	if len(references) > 0 {
		if err := convCtx.PublishReferences(references); err != nil {
			return nil, err
		}
	}

	if err := convCtx.PublishThinking("证据整理完成，正在基于证据生成回答。"); err != nil {
		return nil, err
	}

	// Prompt 装配与预算
	ctx = vo.OnStart(ctx, enum.ConversationTraceStageEvidenceBudget,
		e.Mode().String(), &vo.StageInput{SummaryText: "正在组装证据与 Prompt 预算。"})
	promptResult, err := e.promptAssembler.Assemble(ctx, plan, retrievalCtx)
	if err != nil {
		logx.Errorf("Prompt 组装失败: conversationId=%s, err=%v", convCtx.ConversationId, err)
		vo.OnError(ctx, "证据预算与 Prompt 组装失败。", err)
		return nil, err
	}

	// 填充调试轨迹
	if debugTrace := convCtx.DebugTrace.Load(); debugTrace != nil {
		debugTrace.RagSystemPrompt = promptResult.SystemPrompt
		debugTrace.RagUserPrompt = promptResult.UserPrompt
	}

	_ = vo.OnEnd(ctx, &vo.StageOutput{
		SummaryText: "证据预算与 Prompt 组装完成。",
		Snapshot:    promptResult.ToSnapshot(retrievalCtx),
	})

	ctx = vo.OnStart(ctx, enum.ConversationTraceStageAnswerGenerate, e.Mode().String(),
		&vo.StageInput{SummaryText: "正在基于证据生成回答。"})

	callbackOpt := model.WithCallback(func() {
		vo.OnEnd(ctx, &vo.StageOutput{SummaryText: "答案生成完成。", Snapshot: map[string]any{
			"firstResponseTimeMs": convCtx.FirstResponseTimeMs.Load(),
			"answerLength":        convCtx.AnswerLength(),
		}})
	})
	streamCh, err := e.chatModel.Stream(ctx, enum.ChatStageRagAnswer, promptResult.SystemPrompt, promptResult.UserPrompt, callbackOpt)
	if err != nil {
		logx.Errorf("模型流式调用失败: conversationId=%s, error=%v", convCtx.ConversationId, err)
		vo.OnError(ctx, "答案生成失败。", err)
		return nil, err
	}

	return streamCh, nil
}
