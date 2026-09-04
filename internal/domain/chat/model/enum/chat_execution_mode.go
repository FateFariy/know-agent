package enum

import "github.com/duke-git/lancet/v2/enum"

// ============================================================
// ExecutionMode 执行模式
// ============================================================

type ExecutionMode = *enum.Item[int]

// 执行模式以 {"value":N,"name":"..."} 形式持久化在语义缓存条目中，因此取值固定不复用：
// 2 为已下线的开放式 ReAct Agent 模式（react_agent），保留空档避免历史数据错位。
var (
	ExecutionModeRetrieval     = enum.NewItem(1, "retrieval")     // 知识库检索问答模式
	ExecutionModeClarification = enum.NewItem(3, "clarification") // 澄清模式
)

// ============================================================
// ConversationTraceStage 追踪阶段定义
// ============================================================

type ConversationTraceStage struct {
	Code  string
	Name  string
	Order int
}

const (
	memory         = "MEMORY"
	rewrite        = "REWRITE"
	semanticCache  = "SEMANTIC_CACHE"
	route          = "ROUTE"
	ragRetrieve    = "RAG_RETRIEVE"
	answerEvaluate = "ANSWER_EVALUATE"
	recommendation = "RECOMMENDATION"
	cacheWrite     = "CACHE_WRITE"
	finalize       = "FINALIZE"
)

var (
	ConversationTraceStageMemory         = &ConversationTraceStage{memory, "会话记忆", 10}
	ConversationTraceStageRewrite        = &ConversationTraceStage{rewrite, "问题改写", 20}
	ConversationTraceStageSemanticCache  = &ConversationTraceStage{semanticCache, "语义缓存查询", 30}
	ConversationTraceStageRoute          = &ConversationTraceStage{route, "路由判定", 40}
	ConversationTraceStageRAGRetrieve    = &ConversationTraceStage{ragRetrieve, "RAG 检索", 50}
	ConversationTraceStageAnswerEvaluate = &ConversationTraceStage{answerEvaluate, "回答评估", 60}
	ConversationTraceStageRecommendation = &ConversationTraceStage{recommendation, "推荐问题", 70}
	ConversationTraceStageCacheWrite     = &ConversationTraceStage{cacheWrite, "缓存写入", 80}
	ConversationTraceStageFinalize       = &ConversationTraceStage{finalize, "收尾归档", 90}
)
