package conversation

import (
	"context"
	"fmt"
	"time"

	"github.com/swiftbit/know-agent/common/logx"
	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/enum"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
	"github.com/swiftbit/know-agent/internal/svc"
)

// SemanticCacheStage 语义缓存命中判定阶段。
//
// 位置：QueryRewrite 之后、Route 之前。
// 理由：问题改写必须执行（指代消解/歧义消除），缓存判定基于「改写后的问题」语义向量，
// 命中即语义真正等价。领域层不负责向量化，由 SemanticCacheStore 实现层完成。
//
// 命中后：把缓存的必要执行产物回填进当前执行计划，并打标；后续 Route / Retrieval /
// EvidenceBudget 三段直接早退。两种复用策略都会完整复用检索链路，差异只在 Generate。
type SemanticCacheStage struct {
	store SemanticCacheStore
	*cacheOptions
}

type cacheOptions struct {
	enabled             bool               // 是否启用语义缓存
	similarityThreshold float64            // ANN 相似度阈值
	topK                int                // ANN 候选数
	ttl                 time.Duration      // 缓存条目 TTL
	reuseStrategy       enum.ReuseStrategy // 复用策略
}

var _ Stage = (*SemanticCacheStage)(nil)

func NewSemanticCacheStage(sevCtx *svc.ServiceContext, store SemanticCacheStore) *SemanticCacheStage {
	return &SemanticCacheStage{
		store: store,
		cacheOptions: &cacheOptions{
			enabled:             sevCtx.Config.Chat.Rag.SemanticCache.Enabled,
			similarityThreshold: sevCtx.Config.Chat.Rag.SemanticCache.SimilarityThreshold,
			topK:                sevCtx.Config.Chat.Rag.SemanticCache.TopK,
			ttl:                 sevCtx.Config.Chat.Rag.SemanticCache.TTL,
			reuseStrategy:       utils.Ternary(sevCtx.Config.Chat.Rag.SemanticCache.ReuseAnswer, enum.ReuseRetrievalOnly, enum.ReuseAnswerAndRetrieval),
		},
	}
}

func (s *SemanticCacheStage) Name() string {
	return enum.ConversationTraceStageSemanticCache.Name
}

func (s *SemanticCacheStage) Execute(ctx context.Context, convCtx *Context) error {
	// 未启用或存储不可用：直接走正常链路
	if !s.enabled || s.store == nil {
		return nil
	}
	execPlan := convCtx.ExecutionPlan.Load()
	if execPlan == nil {
		return nil
	}
	convCtx.resetCache(s.reuseStrategy)

	// 不缓存场景：实时/时间敏感/OpenChat → 降级走正常链路
	if execPlan.RequiresRealTimeSearch || execPlan.RequiresCurrentDateAnchoring ||
		convCtx.ChatMode == enum.ChatQueryModeOpenChat {
		return nil
	}

	// 启动语义缓存追踪阶段
	input := &vo.StageInput{
		SummaryText: "正在查询语义缓存。",
		Snapshot: map[string]any{
			"rewriteQuestion":     utils.Trim(execPlan.RewriteQuestion),
			"chatMode":            enum.ChatQueryModeName(convCtx.ChatMode),
			"topK":                s.topK,
			"similarityThreshold": s.similarityThreshold,
			"reuseStrategy":       s.reuseStrategy,
		},
	}
	ctx = vo.OnStart(ctx, enum.ConversationTraceStageSemanticCache, enum.ExecutionModeRetrieval.String(), input)

	// 故障降级：查询失败视为未命中，不阻塞主链路
	scope := buildCacheScope(convCtx)
	hit, err := s.store.Search(ctx, scope, execPlan.RewriteQuestion, s.topK, s.similarityThreshold)
	if err != nil {
		logx.Warnf("语义缓存查询失败，降级未命中: conversationId=%s, error=%v", convCtx.ConversationId, err)
		ctx = vo.OnError(ctx, "语义缓存查询失败，降级未命中。", err)
		convCtx.markCacheMiss()
		return nil
	}
	if hit == nil {
		convCtx.markCacheMiss()
		ctx = vo.OnEnd(ctx, &vo.StageOutput{
			SummaryText: "语义缓存未命中。",
			Snapshot: map[string]any{
				"hit":             false,
				"rewriteQuestion": utils.Trim(execPlan.RewriteQuestion),
				"chatMode":        enum.ChatQueryModeName(convCtx.ChatMode),
			},
		})
		logx.Infof("语义缓存未命中: conversationId=%s, rewriteQuestion='%s'",
			convCtx.ConversationId, execPlan.RewriteQuestion)
		return nil
	}

	// 轻量校验：检索结果合法性 + Prompt 完整性（防配置漂移导致复用了不符预期的产物）
	if !hit.Entry.Validate() {
		logx.Warnf("语义缓存条目校验失败，降级未命中: conversationId=%s", convCtx.ConversationId)
		convCtx.markCacheMiss()
		ctx = vo.OnError(ctx, "语义缓存条目校验失败，降级未命中。", fmt.Errorf("cache entry validation failed, entryId=%s", hit.Entry.ID))
		return nil
	}

	// 单一挂载点：回填必要字段 + 打标（保留当前请求私有上下文）
	convCtx.applyCachedExecution(hit.Entry.Execution)
	convCtx.MarkCacheHit(hit)

	ctx = vo.OnEnd(ctx, &vo.StageOutput{
		SummaryText: "语义缓存命中，复用已有执行产物。",
		Snapshot: map[string]any{
			"hit":           true,
			"entryId":       hit.Entry.ID,
			"similarity":    hit.Similarity,
			"reuseStrategy": s.reuseStrategy,
			"executionMode": hit.Entry.Execution.Mode.Name(),
			"answerLength":  len(hit.Entry.AnswerDraft),
		},
	})
	logx.Infof("语义缓存命中: conversationId=%s, entryId=%s, similarity=%.4f, reuseStrategy=%d",
		convCtx.ConversationId, hit.Entry.ID, hit.Similarity, s.reuseStrategy)

	return nil
}

// buildCacheScope 仅包含数据隔离维度
func buildCacheScope(convCtx *Context) *CacheScope {
	snap := convCtx.KnowledgeBaseSelectionSnapshot
	return &CacheScope{
		ChatMode:           convCtx.ChatMode,
		AllowedDocumentIds: snap.SelectedDocumentIds(),
		AllowedTaskIds:     snap.SelectedTaskIds(),
		KnowledgeBaseIds:   snap.SelectedKnowledgeBaseIds,
	}
}

// buildCachedExecution 从当前执行计划抽取可复用的执行产物
func buildCachedExecution(plan *vo.ConversationExecutionPlan) *CachedExecution {
	if plan == nil {
		return nil
	}
	return &CachedExecution{
		Mode:                 plan.Mode,
		RetrievalPlan:        plan.RetrievalPlan,
		RetrievalResult:      plan.RetrievalResult,
		PromptAssemblyResult: plan.PromptAssemblyResult,
	}
}
