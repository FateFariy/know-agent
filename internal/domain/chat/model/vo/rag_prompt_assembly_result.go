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

// ToSnapshot 构建证据预算阶段快照
func (r *RagPromptAssemblyResult) ToSnapshot(retrievalCtx *RagRetrievalContext) map[string]any {
	evidence := retrievalCtx.SubQuestionEvidenceList
	_ = retrievalCtx.ValidateEvidenceBudgetScope()

	return map[string]any{
		"subQuestionCount":         len(evidence),
		"totalBudget":              r.TotalBudget,
		"perSubQuestionBudget":     r.PerSubQuestionBudget,
		"renderedReferenceCount":   r.RenderedReferenceCount,
		"omittedReferenceCount":    r.OmittedReferenceCount,
		"renderedReferenceDetails": r.RenderedReferenceDetails,
		"omittedReferenceDetails":  r.OmittedReferenceDetails,
		"systemPrompt":             r.SystemPrompt,
		"userPrompt":               r.UserPrompt,
		"subQuestionBudgetDetails": retrievalCtx.BuildSubQuestionBudgetDetails(r),
	}
}
