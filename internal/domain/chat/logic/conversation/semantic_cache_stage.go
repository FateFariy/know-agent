package conversation

import (
	"context"
	"fmt"
	"strings"

	"github.com/swiftbit/know-agent/common/logx"
	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/chat/adapter"
	"github.com/swiftbit/know-agent/internal/domain/chat/adapter/model"
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
// 命中判定采用分区间策略（设计方案：无 Rerank + 大模型终判）：
//  1. 向量候选召回：TopK（≥召回阈值），存储层不判定命中；
//  2. 最高候选得分 ≥ 直接命中阈值（高置信区）→ 直接命中 Top1，零大模型开销；
//  3. 最高候选得分处于灰色区间 → JudgeEnabled 时大模型批量终判（置信度兜底地板校验），否则保守未命中。
//
// 命中后：把缓存的必要执行产物回填进当前执行计划，并打标；后续 Route / Retrieval /
// EvidenceBudget 三段直接早退。两种复用策略都会完整复用检索链路，差异只在 Generate。
type SemanticCacheStage struct {
	store    SemanticCacheStore
	llm      model.ChatModel
	renderer adapter.PromptRenderer
	*cacheOptions
}

type cacheOptions struct {
	enabled             bool               // 是否启用语义缓存
	similarityThreshold float32            // 向量召回粗筛阈值：低于该分直接未命中
	directHitThreshold  float32            // 直接命中阈值：最高候选达到该分跳过 LLM 判定
	recallTopK          int                // 向量召回候选数
	confidenceFloor     float32            // LLM 判定置信度兜底地板
	judgeEnabled        bool               // 灰色区间是否启用大模型等价终判
	judgeTemperature    float32            // 大模型判定温度
	reuseStrategy       enum.ReuseStrategy // 复用策略
}

var _ Stage = (*SemanticCacheStage)(nil)

var _ ConditionalStage = (*SemanticCacheStage)(nil)

func NewSemanticCacheStage(sevCtx *svc.ServiceContext, store SemanticCacheStore, judgeLLM model.ChatModel, renderer adapter.PromptRenderer) *SemanticCacheStage {
	sc := sevCtx.Config.Chat.SemanticCache
	return &SemanticCacheStage{
		store:    store,
		llm:      judgeLLM,
		renderer: renderer,
		cacheOptions: &cacheOptions{
			enabled:             sc.Enabled,
			similarityThreshold: sc.SimilarityThreshold,
			directHitThreshold:  sc.DirectHitThreshold,
			recallTopK:          sc.RecallTopK,
			confidenceFloor:     sc.ConfidenceFloor,
			judgeEnabled:        sc.JudgeEnabled,
			judgeTemperature:    sc.JudgeTemperature,
			reuseStrategy:       utils.Ternary(sc.ReuseAnswer, enum.ReuseAnswerAndRetrieval, enum.ReuseRetrievalOnly),
		},
	}
}

func (s *SemanticCacheStage) Name() string {
	return enum.ConversationTraceStageSemanticCache.Name
}

func (s *SemanticCacheStage) Order() int {
	return enum.ConversationTraceStageSemanticCache.Order
}

// ShouldExecute 仅当语义缓存已启用、执行计划就绪且非实时/时间敏感时执行
func (s *SemanticCacheStage) ShouldExecute(ctx context.Context, convCtx *Context) bool {
	if !s.enabled || s.store == nil {
		return false
	}
	execPlan := convCtx.ExecutionPlan.Load()
	if execPlan == nil {
		return false
	}
	return !execPlan.RequiresRealTimeSearch && !execPlan.RequiresCurrentDateAnchoring
}

func (s *SemanticCacheStage) Execute(ctx context.Context, convCtx *Context) error {
	convCtx.cache = &semanticCacheCtx{strategy: s.reuseStrategy}
	execPlan := convCtx.ExecutionPlan.Load()
	queryText := utils.Trim(execPlan.RewriteQuestion)

	// 启动语义缓存追踪阶段
	ctx = vo.OnStart(ctx, enum.ConversationTraceStageSemanticCache, &vo.StageInput{
		SummaryText: "正在查询语义缓存。",
		Snapshot: map[string]any{
			"rewriteQuestion":    queryText,
			"recallThreshold":    s.similarityThreshold,
			"directHitThreshold": s.directHitThreshold,
			"recallTopK":         s.recallTopK,
			"judgeEnabled":       s.judgeEnabled,
			"reuseStrategy":      s.reuseStrategy,
		},
	})

	// 故障降级：查询失败视为未命中，不阻塞主链路
	candidates, err := s.store.SearchCandidates(ctx, &SearchInput{
		Scope:     buildCacheScope(convCtx),
		QueryText: execPlan.RewriteQuestion,
		Threshold: s.similarityThreshold,
		TopK:      s.recallTopK,
	})
	if err != nil {
		logx.Warnf("语义缓存查询失败，降级未命中: conversationId=%s, error=%v", convCtx.ConversationId, err)
		ctx = vo.OnError(ctx, "语义缓存查询失败，降级未命中。", err)
		convCtx.cache.MarkCacheMiss()
		return nil
	}

	hit := s.decideHit(ctx, execPlan.RewriteQuestion, candidates)
	if hit == nil {
		convCtx.cache.MarkCacheMiss()
		ctx = vo.OnEnd(ctx, &vo.StageOutput{
			SummaryText: "语义缓存未命中。",
			Snapshot: map[string]any{
				"hit":             false,
				"candidateCount":  len(candidates),
				"rewriteQuestion": queryText,
			},
		})
		logx.Infof("语义缓存未命中: conversationId=%s, rewriteQuestion='%s', candidateCount=%d",
			convCtx.ConversationId, queryText, len(candidates))
		return nil
	}

	// 轻量校验：检索结果合法性 + Prompt 完整性（防配置漂移导致复用了不符预期的产物）
	if !hit.Entry.Validate() {
		logx.Warnf("语义缓存条目校验失败，降级未命中: conversationId=%s", convCtx.ConversationId)
		convCtx.cache.MarkCacheMiss()
		ctx = vo.OnError(ctx, "语义缓存条目校验失败，降级未命中。", fmt.Errorf("cache entry validation failed, entryId=%d", hit.Entry.ID))
		return nil
	}

	// 回填必要字段
	convCtx.applyCachedExecution(hit.Entry.Execution)
	convCtx.cache.MarkCacheHit(hit)

	ctx = vo.OnEnd(ctx, &vo.StageOutput{
		SummaryText: "语义缓存命中，复用已有执行产物。",
		Snapshot: map[string]any{
			"hit":           true,
			"entryId":       hit.Entry.ID,
			"similarity":    hit.Similarity,
			"confidence":    hit.Confidence,
			"reuseStrategy": s.reuseStrategy,
			"answerLength":  len(hit.Entry.AnswerDraft),
		},
	})
	logx.Infof("语义缓存命中: conversationId=%s, entryId=%d, similarity=%.4f, confidence=%.4f, reuseStrategy=%d",
		convCtx.ConversationId, hit.Entry.ID, hit.Similarity, hit.Confidence, s.reuseStrategy)

	return nil
}

// decideHit 分区间命中判定：
//   - 无候选或最高得分低于召回阈值 → 未命中；
//   - 最高得分 ≥ 直接命中阈值（高置信区）→ 直接命中 Top1，置信度记为 1.0；
//   - 灰色区间：JudgeEnabled 时大模型批量终判，否则保守未命中。
func (s *SemanticCacheStage) decideHit(ctx context.Context, query string, candidates []*CacheHit) *CacheHit {
	if len(candidates) == 0 {
		return nil
	}
	best := candidates[0]
	if best.Similarity >= s.directHitThreshold {
		best.Confidence = 1
		return best
	}
	if s.judgeEnabled {
		return s.judgeCacheEquivalence(ctx, query, candidates)
	}
	return nil
}

// buildCacheScope 仅包含数据隔离维度
func buildCacheScope(convCtx *Context) *vo.CacheScope {
	snap := convCtx.KnowledgeBaseSelectionSnapshot
	return &vo.CacheScope{
		ChatMode:           convCtx.ChatMode,
		AllowedDocumentIds: snap.SelectedDocumentIds(),
		AllowedTaskIds:     snap.SelectedTaskIds(),
		KnowledgeBaseIds:   snap.SelectedKnowledgeBaseIds,
	}
}

// cacheEquivalenceOutput 大模型语义等价终判输出
type cacheEquivalenceOutput struct {
	Hit        bool    `json:"hit"`        // 是否命中
	Index      int     `json:"index"`      // 命中候选在候选列表中的 1-based 序号
	Reason     string  `json:"reason"`     // 命中原因
	Confidence float64 `json:"confidence"` // 置信度
}

// judgeCacheEquivalence 对灰色区间候选做批量等价终判：
// 仅当模型判定命中且置信度 ≥ confidenceFloor，且命中条目确实在候选池内时才返回该候选；
// 其余情况一律保守未命中（返回 nil），避免模型幻觉误召回。
func (s *SemanticCacheStage) judgeCacheEquivalence(ctx context.Context, query string, candidates []*CacheHit) *CacheHit {
	if s.llm == nil || s.renderer == nil || len(candidates) == 0 {
		return nil
	}
	output, err := s.renderAndJudge(ctx, query, candidates)
	if err != nil {
		logx.Warnf("语义缓存大模型终判失败，降级未命中: query='%s', error=%v", query, err)
		return nil
	}
	if !output.Hit {
		return nil
	}
	if output.Confidence < float64(s.confidenceFloor) {
		logx.Warnf("语义缓存大模型终判置信度不足，强制未命中: confidence=%.4f, floor=%.4f, query='%s'",
			output.Confidence, s.confidenceFloor, query)
		return nil
	}
	targetIndex := output.Index
	if targetIndex < 1 || targetIndex > len(candidates) {
		logx.Warnf("语义缓存大模型终判命中索引越界，按未命中处理: index=%d, candidateCount=%d", targetIndex, len(candidates))
		return nil
	}
	cand := candidates[targetIndex-1]
	cand.Confidence = float32(output.Confidence)
	return cand
}

// renderAndJudge 渲染判定模板并调用大模型，返回结构化判定结果
func (s *SemanticCacheStage) renderAndJudge(ctx context.Context, query string, candidates []*CacheHit) (*cacheEquivalenceOutput, error) {
	lines := make([]string, 0, len(candidates))
	for i, cand := range candidates {
		lines = append(lines, fmt.Sprintf("%d. %s", i+1, strings.TrimSpace(cand.Entry.QueryText)))
	}
	prompt, err := s.renderer.Render(enum.SemanticCacheJudge, map[string]any{
		"currentQuery":   query,
		"candidatesList": strings.Join(lines, "\n"),
	})
	if err != nil {
		return nil, fmt.Errorf("渲染语义缓存判定模板失败: %w", err)
	}
	response, err := s.llm.Generate(ctx, "", prompt, model.WithFunction("judge"), model.WithTemperature(s.judgeTemperature))
	if err != nil {
		return nil, fmt.Errorf("调用大模型判定失败: %w", err)
	}
	var output cacheEquivalenceOutput
	if err = utils.Unmarshal(response, &output); err != nil {
		return nil, fmt.Errorf("解析大模型判定结果失败: %w", err)
	}
	return &output, nil
}
