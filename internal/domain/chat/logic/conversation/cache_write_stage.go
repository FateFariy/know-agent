package conversation

import (
	"context"

	"github.com/swiftbit/know-agent/common/logx"
	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/enum"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
	"github.com/swiftbit/know-agent/internal/svc"
)

// CacheWriteStage 语义缓存回写阶段
type CacheWriteStage struct {
	store   SemanticCacheStore
	enabled bool
}

var _ Stage = (*CacheWriteStage)(nil)

var _ ConditionalStage = (*CacheWriteStage)(nil)

func NewCacheWriteStage(svcCtx *svc.ServiceContext, store SemanticCacheStore) *CacheWriteStage {
	return &CacheWriteStage{
		store:   store,
		enabled: svcCtx.Config.Chat.SemanticCache.Enabled,
	}
}

func (s *CacheWriteStage) Name() string {
	return enum.ConversationTraceStageCacheWrite.Name
}

func (s *CacheWriteStage) Order() int {
	return enum.ConversationTraceStageCacheWrite.Order
}

// ShouldExecute 仅当语义缓存已启用且存储可用时执行
func (s *CacheWriteStage) ShouldExecute(ctx context.Context, convCtx *Context) bool {
	if !s.enabled || s.store == nil {
		return false
	}
	entry := convCtx.cache.CacheEntry()
	return !(convCtx.cache.IsCacheHit() && convCtx.cache.ReuseStrategy() == enum.ReuseAnswerAndRetrieval &&
		entry != nil && utils.IsNotBlank(entry.AnswerDraft))
}

func (s *CacheWriteStage) Execute(ctx context.Context, convCtx *Context) error {
	// 启动缓存写入追踪阶段
	ctx = vo.OnStart(ctx, enum.ConversationTraceStageCacheWrite, &vo.StageInput{SummaryText: "正在回写语义缓存。"})

	// 路径1 & 2：Put 写入（新条目或命中后答案更新）
	plan := convCtx.ExecutionPlan.Load()
	entry := convCtx.cache.CacheEntry()
	if entry == nil {
		// 未命中：构造完整条目（MySQL 真值）；向量记录由 store 按 QueryText 向量化写入 Milvus
		entry = &entity.ChatCacheEntry{
			ChatMode:           convCtx.ChatMode,
			AllowedDocumentIds: convCtx.KnowledgeBaseSelectionSnapshot.SelectedDocumentIds(),
			AllowedTaskIds:     convCtx.KnowledgeBaseSelectionSnapshot.SelectedTaskIds(),
			KnowledgeBaseIds:   convCtx.KnowledgeBaseSelectionSnapshot.SelectedKnowledgeBaseIds,
			QueryText:          plan.RewriteQuestion,
			Execution:          buildCachedExecution(plan),
			AnswerDraft:        convCtx.Answer(),
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
		logx.Infof("语义缓存写入完成: conversationId=%s, entryId=%d,  answerLength=%d",
			convCtx.ConversationId, entry.ID, len(entry.AnswerDraft))
		ctx = vo.OnEnd(ctx, &vo.StageOutput{
			SummaryText: "语义缓存写入完成。",
			Snapshot: map[string]any{
				"entryId":      entry.ID,
				"queryText":    utils.Trim(entry.QueryText),
				"answerLength": len(entry.AnswerDraft),
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
