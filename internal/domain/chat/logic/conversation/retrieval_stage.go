package conversation

import (
	"context"
	"fmt"

	"github.com/swiftbit/know-agent/common/logx"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/enum"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
)

type RetrievalStage struct {
	retriever Retriever
}

func NewRetrievalStage(retriever Retriever) *RetrievalStage {
	return &RetrievalStage{
		retriever: retriever,
	}
}

func (r *RetrievalStage) Name() string {
	return enum.ConversationTraceStageRAGRetrieve.Name
}

func (r *RetrievalStage) Execute(ctx context.Context, convCtx *Context) error {
	if convCtx.ChatMode == enum.ChatQueryModeOpenChat {
		return nil
	}

	// 加载执行计划，缺失时直接报错
	execPlan := convCtx.ExecutionPlan.Load()
	if execPlan == nil {
		return fmt.Errorf("invalid value")
	}

	ctx = vo.OnStart(ctx, enum.ConversationTraceStageRAGRetrieve, r.Name(), &vo.StageInput{SummaryText: "正在执行多通道混合检索。"})
	if err := convCtx.PublishThinking("正在根据问题规划知识检索范围。"); err != nil {
		return err
	}

	retrievalResult, err := r.retriever.Retrieve(ctx, execPlan.RetrievalPlan)
	if err != nil {
		logx.Errorf("RAG 检索失败: conversationId=%s, error=%v", convCtx.ConversationId, err)
		vo.OnError(ctx, "RAG 检索失败。", err)
		return err
	}

	// 先下发思考事件（检索笔记、渠道列表）
	notes := retrievalResult.RetrievalNotes()
	for _, note := range notes {
		if err = convCtx.PublishThinking(note); err != nil {
			return err
		}
	}
	references := retrievalResult.FlattenReferences()
	if len(references) > 0 {
		if err = convCtx.PublishReferences(references); err != nil {
			return err
		}
	}

	// 合并渠道记录到上下文与调试轨迹
	chs := retrievalResult.UsedChannels()
	convCtx.AddUsedTools(chs...)
	if debugTrace := convCtx.DebugTrace.Load(); debugTrace != nil {
		debugTrace.SetUsedChannels(chs...)
		debugTrace.SetRetrievalNotes(retrievalResult.RetrievalNotes()...)
	}

	execPlan.RetrievalResult = retrievalResult

	_ = vo.OnEnd(ctx, &vo.StageOutput{
		SummaryText: "RAG 检索完成。",
		Snapshot:    retrievalResult.ToSnapshot(execPlan.RetrievalPlan),
	})

	return nil
}
