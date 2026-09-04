package conversation

import (
	"context"
	"strconv"

	"github.com/swiftbit/know-agent/common/logx"
	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/enum"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
	"github.com/swiftbit/know-agent/internal/svc"
)

// CacheWriteStage 缓存写入阶段（异步投递）。
//
// 命中即复用：跳过写入（避免用旧答案覆盖最新答案）。
// 未命中：写完整链路（检索结果 + 最终答案）。
// 命中但复用检索结果（需要重新生成答案）：更新答案（复用已有检索结果）。
//
// 写入链路强制异步：主链路只投递 ChatCacheEntry 到 MQ，幂等双写（MySQL + Milvus）
// 由独立消费者完成；向量化在消费端 store.Put 内部完成。投递失败仅告警，不阻塞主链路。
type CacheWriteStage struct {
	store      SemanticCacheStore
	producer   MessageProducer
	writeTopic string
}

var _ Stage = (*CacheWriteStage)(nil)

var _ ConditionalStage = (*CacheWriteStage)(nil)

func NewCacheWriteStage(sevCtx *svc.ServiceContext, store SemanticCacheStore, producer MessageProducer) *CacheWriteStage {
	sc := sevCtx.Config.Chat.SemanticCache
	return &CacheWriteStage{
		store:      store,
		producer:   producer,
		writeTopic: sc.WriteTopic,
	}
}

func (s *CacheWriteStage) Name() string {
	return enum.ConversationTraceStageCacheWrite.Name
}

func (s *CacheWriteStage) Order() int {
	return enum.ConversationTraceStageCacheWrite.Order
}

// ShouldExecute 仅在语义缓存启用、且（未命中 或 命中但需重新生成答案）时执行
func (s *CacheWriteStage) ShouldExecute(ctx context.Context, convCtx *Context) bool {
	if s.store == nil {
		return false
	}
	// 命中且直接复用答案（已含答案）→ 跳过，避免覆盖最新答案
	if convCtx.cache != nil && convCtx.cache.IsCacheHit() && convCtx.cache.ReuseStrategy() == enum.ReuseAnswerAndRetrieval {
		return false
	}
	return true
}

func (s *CacheWriteStage) Execute(ctx context.Context, convCtx *Context) error {
	ctx = vo.OnStart(ctx, enum.ConversationTraceStageCacheWrite, &vo.StageInput{
		SummaryText: "正在异步写入语义缓存。",
		Snapshot: map[string]any{
			"hit":           convCtx.cache != nil && convCtx.cache.IsCacheHit(),
			"reuseStrategy": convCtx.cache.ReuseStrategy(),
			"writeTopic":    s.writeTopic,
		},
	})

	plan := convCtx.ExecutionPlan.Load()
	cachedExecution := buildCachedExecution(plan)
	var answerDraft string
	if convCtx.cache != nil && convCtx.cache.IsCacheHit() {
		// 命中 + 复用检索结果：沿用既有检索结果，仅更新答案
		answerDraft = convCtx.Answer()
		if entry := convCtx.cache.CacheEntry(); entry != nil {
			cachedExecution = entry.Execution
		}
	} else {
		// 未命中：写入完整链路（检索结果 + 最终答案）
		answerDraft = convCtx.Answer()
	}

	snap := convCtx.KnowledgeBaseSelectionSnapshot
	entry := &entity.ChatCacheEntry{
		ID:                 utils.GetSnowflakeNextID(),
		QueryText:          plan.RewriteQuestion,
		ChatMode:           convCtx.ChatMode,
		AllowedDocumentIds: snap.SelectedDocumentIds(),
		AllowedTaskIds:     snap.SelectedTaskIds(),
		KnowledgeBaseIds:   snap.SelectedKnowledgeBaseIds,
		Execution:          cachedExecution,
		AnswerDraft:        answerDraft,
	}

	key := strconv.FormatInt(entry.ID, 10)
	if err := s.producer.Send(ctx, s.writeTopic, key, entry); err != nil {
		logx.Warnf("语义缓存写入消息投递失败，降级不写入（不影响本次响应）: conversationId=%s, cacheId=%d, error=%v",
			convCtx.ConversationId, entry.ID, err)
		ctx = vo.OnError(ctx, "语义缓存写入消息投递失败，降级不写入。", err)
		return nil
	}

	ctx = vo.OnEnd(ctx, &vo.StageOutput{
		SummaryText: "语义缓存写入消息已投递。",
		Snapshot: map[string]any{
			"cacheId":   entry.ID,
			"queryText": utils.Trim(plan.RewriteQuestion),
			"chatMode":  enum.ChatQueryModeName(convCtx.ChatMode),
		},
	})
	logx.Infof("语义缓存写入消息已投递: conversationId=%s, cacheId=%d, queryText='%s'",
		convCtx.ConversationId, entry.ID, utils.Trim(plan.RewriteQuestion))
	return nil
}

// buildCachedExecution 从执行计划抽取可缓存执行产物（检索计划 + 检索结果）
func buildCachedExecution(plan *vo.ConversationExecutionPlan) *vo.CachedExecution {
	if plan == nil {
		return nil
	}
	return &vo.CachedExecution{
		RetrievalPlan:   plan.RetrievalPlan,
		RetrievalResult: plan.RetrievalResult,
	}
}
