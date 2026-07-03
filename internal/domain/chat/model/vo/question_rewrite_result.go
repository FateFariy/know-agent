package vo

// QuestionRewriteResult 问题改写结果
type QuestionRewriteResult struct {
	RewrittenQuestion string   // 改写后的问题
	SubQuestions      []string // 子问题列表
	RawModelOutput    string   // 原始模型输出
}

// NewQuestionRewriteResult 创建问题改写结果
func NewQuestionRewriteResult(rewrittenQuestion string, subQuestions []string) *QuestionRewriteResult {
	return &QuestionRewriteResult{
		RewrittenQuestion: rewrittenQuestion,
		SubQuestions:      subQuestions,
	}
}
