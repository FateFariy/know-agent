package conversation

import (
	"context"
	"time"

	"github.com/swiftbit/know-agent/common/logx"
	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/enum"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
	"github.com/swiftbit/know-agent/internal/svc"
)

// CacheWriteStage 语义缓存回写阶段（链尾）。
//
// 写入策略（双写，与命中复用策略无关）：
//   - 未命中：构造完整 ChatCacheEntry（AnswerDraft + 可复用执行产物）写入。
//   - 命中（无论复用检索结果还是复用答案）：覆盖 AnswerDraft 后 Put；命中分支下答案与缓存一致时
//     等价于一次全量重写（含 ExpireAt 续期），不再单独做 Touch 续期，简化写入路径。
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

	// 路径1 & 2：Put 写入（新条目或命中后答案更新）
	plan := convCtx.ExecutionPlan.Load()
	entry := convCtx.cache.CacheEntry()
	if entry == nil {
		// 未命中：构造完整条目（MySQL 真值）；向量记录由 store 按 QueryText 向量化写入 Milvus
		entry = &entity.ChatCacheEntry{
			Scope:       buildCacheScope(convCtx),
			QueryText:   plan.RewriteQuestion,
			Execution:   buildCachedExecution(plan),
			AnswerDraft: convCtx.Answer(),
		}
	} else {
		// 命中 + 复用检索结果：Generate 已产出新答案，仅更新 MySQL 的 answer_draft；
		entry.AnswerDraft = convCtx.Answer()
	}

	if err := s.store.Put(ctx, entry); err != nil {
		logx.Warnf("语义缓存写入失败(忽略): conversationId=%s, entryId=%d, error=%v",
			convCtx.ConversationId, entry.ID, err)
		ctx = vo.OnError(ctx, "语义缓存写入失败(忽略)。", err)
	} else {
		logx.Infof("语义缓存写入完成: conversationId=%s, entryId=%d,  answerLength=%d, ttl=%v",
			convCtx.ConversationId, entry.ID, len(entry.AnswerDraft), s.ttl)
		ctx = vo.OnEnd(ctx, &vo.StageOutput{
			SummaryText: "语义缓存写入完成。",
			Snapshot: map[string]any{
				"entryId":      entry.ID,
				"queryText":    utils.Trim(entry.QueryText),
				"answerLength": len(entry.AnswerDraft),
				"ttl":          s.ttl.String(),
			},
		})
	}
	return nil
}

// buildCachedExecution 从当前执行计划抽取可复用的执行产物
func buildCachedExecution(plan *vo.ConversationExecutionPlan) *vo.CachedExecution {
	if plan == nil {
		return nil
	}
	return &vo.CachedExecution{
		Mode:                 plan.Mode,
		RetrievalPlan:        plan.RetrievalPlan,
		RetrievalResult:      plan.RetrievalResult,
		PromptAssemblyResult: plan.PromptAssemblyResult,
	}
}
