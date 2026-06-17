package prompt

// Prompt模板名称常量
const (
	AgentQuestion                       = "agent-question"
	ChatQueryRewrite                    = "chat-query-rewrite"
	ConversationSummaryMerge            = "conversation-summary-merge"
	ConversationSummarySystem           = "conversation-summary-system"
	DocumentGraphOnlyIntent             = "document-graph-only-intent"
	DocumentLlmSplit                    = "document-llm-split"
	DocumentStructureAmbiguity          = "document-structure-ambiguity"
	DocumentStructureAmbiguityCandidate = "document-structure-ambiguity-candidate"
	RagAnswerDocumentReference          = "rag-answer-document-reference"
	RagAnswerNoEvidence                 = "rag-answer-no-evidence"
	RagAnswerOmittedEvidence            = "rag-answer-omitted-evidence"
	RagAnswerReuseReference             = "rag-answer-reuse-reference"
	RagAnswerSubQuestionEvidence        = "rag-answer-sub-question-evidence"
	RagAnswerSystem                     = "rag-answer-system"
	RagAnswerUser                       = "rag-answer-user"
	RagAnswerWebReference               = "rag-answer-web-reference"
	RecommendationUser                  = "recommendation-user"
)
