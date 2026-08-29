package enum

import "github.com/duke-git/lancet/v2/enum"

// ============================================================
// ExecutionMode 执行模式
// ============================================================

type ExecutionMode = *enum.Item[int]

var (
	ExecutionModeRetrieval     = enum.NewItem(1, "retrieval")     // 普通知识库检索问答模式
	ExecutionModeReactAgent    = enum.NewItem(2, "react_agent")   // 开放式 ReAct Agent 模式
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
	intent         = "INTENT"
	rewrite        = "REWRITE"
	semanticCache  = "SEMANTIC_CACHE"
	route          = "ROUTE"
	ragRetrieve    = "RAG_RETRIEVE"
	evidenceBudget = "EVIDENCE_BUDGET"
	answerGenerate = "ANSWER_GENERATE"
	reActAgent     = "REACT_AGENT"
	answerEvaluate = "ANSWER_EVALUATE"
	recommendation = "RECOMMENDATION"
	cacheWrite     = "CACHE_WRITE"
	finalize       = "FINALIZE"
)

var (
	ConversationTraceStageMemory         = &ConversationTraceStage{memory, "会话记忆", 10}
	ConversationTraceStageRewrite        = &ConversationTraceStage{rewrite, "问题改写", 20}
	ConversationTraceStageSemanticCache  = &ConversationTraceStage{semanticCache, "语义缓存查询", 25}
	ConversationTraceStageIntent         = &ConversationTraceStage{intent, "意图分析", 30}
	ConversationTraceStageRoute          = &ConversationTraceStage{route, "路由判定", 40}
	ConversationTraceStageRAGRetrieve    = &ConversationTraceStage{ragRetrieve, "RAG 检索", 50}
	ConversationTraceStageEvidenceBudget = &ConversationTraceStage{evidenceBudget, "证据评估与预算控制", 60}
	ConversationTraceStageAnswerGenerate = &ConversationTraceStage{answerGenerate, "回答生成", 70}
	ConversationTraceStageReActAgent     = &ConversationTraceStage{reActAgent, "ReAct Agent", 75}
	ConversationTraceStageAnswerEvaluate = &ConversationTraceStage{answerEvaluate, "回答评估", 76}
	ConversationTraceStageRecommendation = &ConversationTraceStage{recommendation, "推荐问题", 80}
	ConversationTraceStageCacheWrite     = &ConversationTraceStage{cacheWrite, "缓存写入", 85}
	ConversationTraceStageFinalize       = &ConversationTraceStage{finalize, "收尾归档", 90}
)
