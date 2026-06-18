package vo

// RagRewriteResult 问题改写结果
type RagRewriteResult struct {
	RewrittenQuestion string   // 改写后的问题
	SubQuestions      []string // 子问题列表
	RawModelOutput    string   // 原始模型输出
}

// NewRagRewriteResult 创建问题改写结果
func NewRagRewriteResult(rewrittenQuestion string, subQuestions []string) *RagRewriteResult {
	return &RagRewriteResult{
		RewrittenQuestion: rewrittenQuestion,
		SubQuestions:      subQuestions,
	}
}
