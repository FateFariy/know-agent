package conversation

import (
	"context"
	"fmt"

	"github.com/swiftbit/know-agent/common/logx"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/rag"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/enum"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
)

type RetrievalStage struct {
	retriever rag.Retriever
}

func NewRetrievalStage(retriever rag.Retriever) *RetrievalStage {
	return &RetrievalStage{
		retriever: retriever,
	}
}

func (r *RetrievalStage) Name() string {
	//TODO implement me
	panic("implement me")
}

func (r *RetrievalStage) Execute(ctx context.Context, convCtx *Context) error {
	if convCtx.ChatMode == enum.ChatQueryModeOpenChat {
		return nil
	}

	// 加载执行计划，缺失时直接报错
	plan := convCtx.ExecutionPlan.Load()
	if plan == nil {
		return fmt.Errorf("invalid value")
	}

	if err := convCtx.PublishThinking("正在根据问题规划知识检索范围。"); err != nil {
		return err
	}

	ctx = vo.OnStart(ctx, enum.ConversationTraceStageRAGRetrieve, r.Name(), &vo.StageInput{SummaryText: "正在执行多通道混合检索。"})

	retrievalResult, err := r.retriever.Retrieve(ctx, plan.RetrievalPlan)
	if err != nil {
		logx.Errorf("RAG 检索失败: conversationId=%s, error=%v", convCtx.ConversationId, err)
		vo.OnError(ctx, "RAG 检索失败。", err)
		return err
	}

	_ = vo.OnEnd(ctx, &vo.StageOutput{
		SummaryText: "RAG 检索完成。",
		Snapshot:    retrievalResult.ToSnapshot(plan.RetrievalPlan),
	})

	// 先下发思考事件（检索笔记、渠道列表）
	notes := execPlan.RetrievalResult.RetrievalNotes()
	for _, note := range notes {
		if err := convCtx.PublishThinking(note); err != nil {
			return err
		}
	}
	//TODO implement me
	panic("implement me")
}
