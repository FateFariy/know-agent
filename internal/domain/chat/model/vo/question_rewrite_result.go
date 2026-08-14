package vo

import "github.com/swiftbit/know-agent/common/utils"

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

// NormalizeSubQuestions 提取子问题列表，去重，若空则回退到 fallback
func (r *QuestionRewriteResult) NormalizeSubQuestions(fallback string) []string {
	if r == nil || len(r.SubQuestions) == 0 {
		return []string{utils.Trim(fallback)}
	}
	keyOf := func(q string) (string, string, bool) {
		trimmed := utils.Trim(q)
		return trimmed, trimmed, trimmed != ""
	}
	result := utils.FilterMapUniqueLimit(r.SubQuestions, -1, keyOf)
	if len(result) == 0 {
		return []string{utils.Trim(fallback)}
	}
	return result
}
