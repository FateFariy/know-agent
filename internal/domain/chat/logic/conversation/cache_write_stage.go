package conversation

import (
	"context"
	"time"

	"github.com/swiftbit/know-agent/common/logx"
	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/enum"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
	"github.com/swiftbit/know-agent/internal/svc"
)

// CacheWriteStage 语义缓存回写阶段（链尾）。
//
// 写入策略（双写，与命中复用策略无关）：
//   - 未命中：构造完整 CacheEntry（AnswerDraft + 可复用执行产物）写入。
//   - 命中 + 复用检索结果：Generate 已产出新答案，覆盖 AnswerDraft 后 Put，保证缓存答案为最新版本。
//   - 命中 + 复用答案：答案无变化，仅 Touch 续期，避免无效全量写。
//
// 故障降级：写入失败仅告警，不阻塞主流程返回结果。
type CacheWriteStage struct {
	store   SemanticCacheStore
	enabled bool
	ttl     time.Duration
}

var _ Stage = (*CacheWriteStage)(nil)

func NewCacheWriteStage(svcCtx *svc.ServiceContext, store SemanticCacheStore) *CacheWriteStage {
	return &CacheWriteStage{
		store:   store,
		enabled: svcCtx.Config.Chat.Rag.SemanticCache.Enabled,
		ttl:     svcCtx.Config.Chat.Rag.SemanticCache.TTL,
	}
}

func (s *CacheWriteStage) Name() string {
	return enum.ConversationTraceStageCacheWrite.Name
}

func (s *CacheWriteStage) Execute(ctx context.Context, convCtx *Context) error {
	if !s.enabled || s.store == nil {
		return nil
	}

	// 启动缓存写入追踪阶段
	ctx = vo.OnStart(ctx, enum.ConversationTraceStageCacheWrite, enum.ExecutionModeRetrieval.String(),
		&vo.StageInput{SummaryText: "正在回写语义缓存。"})

	// 路径1：命中 + 复用答案 → 仅续期（答案无变化，避免无效全量写）
	if convCtx.IsCacheHit() && convCtx.ReuseStrategy() == enum.ReuseAnswerAndRetrieval {
		entryId := convCtx.CacheEntry().ID
		if err := s.store.Touch(ctx, entryId, s.ttl); err != nil {
			logx.Warnf("语义缓存续期失败(忽略): conversationId=%s, entryId=%s, error=%v",
				convCtx.ConversationId, entryId, err)
			ctx = vo.OnError(ctx, "语义缓存续期失败(忽略)。", err)
		} else {
			logx.Infof("语义缓存续期完成: conversationId=%s, entryId=%s, ttl=%v",
				convCtx.ConversationId, entryId, s.ttl)
			ctx = vo.OnEnd(ctx, &vo.StageOutput{
				SummaryText: "语义缓存续期完成。",
				Snapshot: map[string]any{
					"action":      "touch",
					"entryId":     entryId,
					"reuseAnswer": true,
				},
			})
		}
		return nil
	}

	// 路径2 & 3：Put 写入（新条目或命中后答案更新）
	plan := convCtx.ExecutionPlan.Load()
	entry := convCtx.CacheEntry()
	action := "insert"
	if entry == nil {
		// 未命中：构造完整条目（MySQL 真值）；向量记录由 store 按 QueryText 向量化写入 Milvus
		entry = &CacheEntry{
			ID:          utils.GenerateUUIDWithoutHyphen(),
			Scope:       buildCacheScope(convCtx),
			QueryText:   plan.RewriteQuestion,
			Execution:   buildCachedExecution(plan),
			AnswerDraft: convCtx.Answer(),
			Hint:        convCtx.ReuseStrategy(),
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
			ExpireAt:    time.Now().Add(s.ttl),
		}
	} else {
		// 命中 + 复用检索结果：Generate 已产出新答案，仅更新 MySQL 的 answer_draft；
		// Milvus 索引完全不动（查询向量未变），这是本方案最大优势
		action = "update"
		entry.AnswerDraft = convCtx.Answer()
		entry.UpdatedAt = time.Now()
		entry.ExpireAt = time.Now().Add(s.ttl)
	}

	if err := s.store.Put(ctx, entry, s.ttl); err != nil {
		logx.Warnf("语义缓存写入失败(忽略): conversationId=%s, entryId=%s, action=%s, error=%v",
			convCtx.ConversationId, entry.ID, action, err)
		ctx = vo.OnError(ctx, "语义缓存写入失败(忽略)。", err)
	} else {
		logx.Infof("语义缓存写入完成: conversationId=%s, entryId=%s, action=%s, answerLength=%d, ttl=%v",
			convCtx.ConversationId, entry.ID, action, len(entry.AnswerDraft), s.ttl)
		ctx = vo.OnEnd(ctx, &vo.StageOutput{
			SummaryText: "语义缓存写入完成。",
			Snapshot: map[string]any{
				"action":       action,
				"entryId":      entry.ID,
				"queryText":    utils.Trim(entry.QueryText),
				"answerLength": len(entry.AnswerDraft),
				"ttl":          s.ttl.String(),
			},
		})
	}
	return nil
}
