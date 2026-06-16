package vo

type ConversationTraceStage struct {
	Code  string
	Name  string
	Order int
}

const (
	memory         = "MEMORY"
	intent         = "INTENT"
	rewrite        = "REWRITE"
	route          = "ROUTE"
	graphQuery     = "GRAPH_QUERY"
	ragRetrieve    = "RAG_RETRIEVE"
	evidenceBudget = "EVIDENCE_BUDGET"
	answerGenerate = "ANSWER_GENERATE"
	reActAgent     = "REACT_AGENT"
	recommendation = "RECOMMENDATION"
	finalize       = "FINALIZE"
)

var (
	ConversationTraceStageMemory         = &ConversationTraceStage{memory, "会话记忆", 10}
	ConversationTraceStageIntent         = &ConversationTraceStage{intent, "意图分析", 20}
	ConversationTraceStageRewrite        = &ConversationTraceStage{rewrite, "问题改写", 30}
	ConversationTraceStageRoute          = &ConversationTraceStage{route, "路由判定", 40}
	ConversationTraceStageGraphQuery     = &ConversationTraceStage{graphQuery, "结构图查询", 45}
	ConversationTraceStageRAGRetrieve    = &ConversationTraceStage{ragRetrieve, "RAG 检索", 50}
	ConversationTraceStageEvidenceBudget = &ConversationTraceStage{evidenceBudget, "证据评估与预算控制", 60}
	ConversationTraceStageAnswerGenerate = &ConversationTraceStage{answerGenerate, "回答生成", 70}
	ConversationTraceStageReActAgent     = &ConversationTraceStage{reActAgent, "ReAct Agent", 75}
	ConversationTraceStageRecommendation = &ConversationTraceStage{recommendation, "推荐问题", 80}
	ConversationTraceStageFinalize       = &ConversationTraceStage{finalize, "收尾归档", 90}
)

var ConversationTraceStageMap = map[string]*ConversationTraceStage{
	memory:         ConversationTraceStageMemory,
	intent:         ConversationTraceStageIntent,
	rewrite:        ConversationTraceStageRewrite,
	route:          ConversationTraceStageRoute,
	graphQuery:     ConversationTraceStageGraphQuery,
	ragRetrieve:    ConversationTraceStageRAGRetrieve,
	evidenceBudget: ConversationTraceStageEvidenceBudget,
	answerGenerate: ConversationTraceStageAnswerGenerate,
	reActAgent:     ConversationTraceStageReActAgent,
	recommendation: ConversationTraceStageRecommendation,
	finalize:       ConversationTraceStageFinalize,
}

func ConversationTraceStageFromCode(code string) (*ConversationTraceStage, bool) {
	v, ok := ConversationTraceStageMap[code]
	return v, ok
}
