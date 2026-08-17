package conversation

import (
	"context"

	"github.com/swiftbit/know-agent/common/logx"
	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/chat/adapter"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/memory"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/recommend"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/enum"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
	"github.com/swiftbit/know-agent/internal/svc"
)

type RecommendStage struct {
	repo                adapter.ChatRepository
	manager             memory.SessionMemoryManager
	recommender         recommend.QuestionRecommender
	historyPreviewTurns int
}

func NewRecommendStage(svcCtx svc.ServiceContext, repo adapter.ChatRepository,
	manager memory.SessionMemoryManager, recommender recommend.QuestionRecommender) *RecommendStage {
	return &RecommendStage{
		repo:                repo,
		manager:             manager,
		recommender:         recommender,
		historyPreviewTurns: svcCtx.Config.Chat.Recommendation.HistoryPreviewTurns,
	}
}

func (r *RecommendStage) Name() string {
	return enum.ConversationTraceStageRecommendation.Name
}

func (r *RecommendStage) Execute(ctx context.Context, convCtx *Context) error {
	// 原子检查 Finalized 标志（CAS），确保仅首次调用生效，避免重复收尾
	if !convCtx.Finalized.CompareAndSwap(false, true) {
		return nil
	}

	// 启动 recommendation 阶段
	recommendCtx := vo.OnStart(ctx, enum.ConversationTraceStageRecommendation,
		convCtx.ExecutionModeName(), &vo.StageInput{SummaryText: "正在生成推荐追问。"})

	// 发送 final 事件
	_ = convCtx.PublishFinish()

	// 生成推荐追问
	// - 若本次交互是澄清（NeedClarification 为真），则直接使用澄清选项作为推荐
	// - 否则，拉取最近交互记录，由 recommender 基于当前问答与历史生成推荐
	var recommendations []string
	if convCtx.NeedClarification() {
		recommendations = convCtx.ClarificationOptions()
	} else {
		recentExchanges := r.fetchRecentExchanges(ctx, convCtx.ConversationId, convCtx.ExchangeId)
		recommendations, _ = r.recommender.Generate(ctx, convCtx.Question, convCtx.Answer(), recentExchanges)
	}

	// 完成 recommendation 追踪阶段，并写入推荐数量快照
	snapshot := map[string]any{"recommendationCount": len(recommendations), "recommendations": recommendations}
	_ = vo.OnEnd(recommendCtx, &vo.StageOutput{SummaryText: "推荐追问生成完成。", Snapshot: snapshot})

	// 向客户端流补发推荐事件
	if len(recommendations) > 0 {
		err := convCtx.Sink.Recommendations(recommendations, convCtx.ConversationId, convCtx.ExchangeId)
		if err != nil {
			logx.Warnf("发送推荐事件失败, conversationId=%s, exchangeId=%d, err=%v", convCtx.ConversationId, convCtx.ExchangeId, err)
		}
	}
	convCtx.Recommendations = recommendations

	return nil
}

// fetchRecentExchanges 获取最近的历史轮次（排除当前）
func (r *RecommendStage) fetchRecentExchanges(ctx context.Context, conversationId string, excludeExchangeId int64) []*entity.ChatExchange {
	recent, err := r.repo.ListRecentExchanges(ctx, conversationId, r.historyPreviewTurns)
	if err != nil {
		logx.Warnf("列出最近轮次失败, conversationId=%s, err=%v", conversationId, err)
		return nil
	}
	return utils.Filter(recent, func(ex *entity.ChatExchange) bool {
		return ex != nil && ex.ID != excludeExchangeId
	})
}
