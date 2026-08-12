package vo

// RagPromptAssemblyResult RAG 提示词组装结果
type RagPromptAssemblyResult struct {
	SystemPrompt             string   `json:"systemPrompt"`
	UserPrompt               string   `json:"userPrompt"`
	TotalBudget              int      `json:"totalBudget"`
	PerSubQuestionBudget     int      `json:"perSubQuestionBudget"`
	RenderedReferenceCount   int      `json:"renderedReferenceCount"`
	OmittedReferenceCount    int      `json:"omittedReferenceCount"`
	RenderedReferenceDetails []string `json:"renderedReferenceDetails"`
	OmittedReferenceDetails  []string `json:"omittedReferenceDetails"`
}
